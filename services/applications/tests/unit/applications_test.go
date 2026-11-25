package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"applications/internal/apperrors"
	"applications/internal/identity"
	"applications/internal/models"
	"applications/internal/routes"
	"applications/internal/services"
	"applications/tests/testutil"
)

const (
	applicationsPath = "/api/v1/applications"
	skipsPath        = "/api/v1/skips"
)

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

// memoryApplications is an in-memory ApplicationRepository keyed by owner. It
// keeps rows in change order; the SQL status rules are covered by the
// integration suite.
type memoryApplications struct {
	mu           sync.Mutex
	applications map[string][]models.Application
	skips        map[string][]uuid.UUID
	fail         error
}

func newMemoryApplications() *memoryApplications {
	return &memoryApplications{applications: map[string][]models.Application{}, skips: map[string][]uuid.UUID{}}
}

func ownerKey(ident identity.IdentityContext) string {
	return ident.TenantID.String() + "/" + ident.Subject
}

func (m *memoryApplications) List(_ context.Context, ident identity.IdentityContext) ([]models.Application, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	return append([]models.Application{}, m.applications[ownerKey(ident)]...), nil
}

func (m *memoryApplications) Put(_ context.Context, ident identity.IdentityContext, jobID uuid.UUID, status string) (*models.Application, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	now := time.Now()
	list := m.applications[ownerKey(ident)]
	application := models.Application{JobID: jobID, Status: status, CreatedAt: now}
	list = slices.DeleteFunc(list, func(a models.Application) bool {
		if a.JobID == jobID {
			application.CreatedAt = a.CreatedAt
			return true
		}
		return false
	})
	application.UpdatedAt = now
	m.applications[ownerKey(ident)] = append([]models.Application{application}, list...)
	return &application, nil
}

func (m *memoryApplications) Delete(_ context.Context, ident identity.IdentityContext, jobID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return false, m.fail
	}
	list := m.applications[ownerKey(ident)]
	kept := slices.DeleteFunc(slices.Clone(list), func(a models.Application) bool { return a.JobID == jobID })
	m.applications[ownerKey(ident)] = kept
	return len(kept) < len(list), nil
}

func (m *memoryApplications) Skips(_ context.Context, ident identity.IdentityContext) ([]uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	return append([]uuid.UUID{}, m.skips[ownerKey(ident)]...), nil
}

func (m *memoryApplications) Skip(_ context.Context, ident identity.IdentityContext, jobID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return m.fail
	}
	list := slices.DeleteFunc(m.skips[ownerKey(ident)], func(id uuid.UUID) bool { return id == jobID })
	m.skips[ownerKey(ident)] = append([]uuid.UUID{jobID}, list...)
	return nil
}

func (m *memoryApplications) Unskip(_ context.Context, ident identity.IdentityContext, jobID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return m.fail
	}
	m.skips[ownerKey(ident)] = slices.DeleteFunc(m.skips[ownerKey(ident)], func(id uuid.UUID) bool { return id == jobID })
	return nil
}

func (m *memoryApplications) ClearSkips(_ context.Context, ident identity.IdentityContext) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return m.fail
	}
	delete(m.skips, ownerKey(ident))
	return nil
}

func applicationsHandler(repo *memoryApplications, ident *identity.IdentityContext) http.Handler {
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Applications: services.NewApplicationService(repo)})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ident != nil {
			r = r.WithContext(identity.WithContext(r.Context(), *ident))
		}
		router.ServeHTTP(w, r)
	})
}

func devApplications(repo *memoryApplications) http.Handler {
	dev := identity.Dev()
	return applicationsHandler(repo, &dev)
}

func errorField(body map[string]any) any {
	envelope, _ := body["error"].(map[string]any)
	details, _ := envelope["context"].(map[string]any)
	return details["field"]
}

func TestApplicationsThroughRoutes(t *testing.T) {
	repo := newMemoryApplications()
	handler := devApplications(repo)
	if status, list := testutil.Do(t, handler, http.MethodGet, applicationsPath, nil); status != http.StatusOK || fmt.Sprint(list["items"]) != "[]" {
		t.Fatalf("empty list: %d %v", status, list)
	}
	first, second := uuid.NewString(), uuid.NewString()
	for _, tc := range []struct{ job, status string }{{first, "saved"}, {second, " cv_ready "}, {first, "applied"}} {
		status, body := testutil.Do(t, handler, http.MethodPut, applicationsPath+"/"+tc.job, map[string]any{"status": tc.status})
		if status != http.StatusOK || body["job_id"] != tc.job || body["created_at"] == nil {
			t.Fatalf("PUT %s %s: %d %v", tc.job, tc.status, status, body)
		}
	}
	_, list := testutil.Do(t, handler, http.MethodGet, applicationsPath, nil)
	items := list["items"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["job_id"] != first || items[0].(map[string]any)["status"] != "applied" ||
		items[1].(map[string]any)["status"] != "cv_ready" {
		t.Fatalf("list: %v", list)
	}
	if status, _ := testutil.Do(t, handler, http.MethodDelete, applicationsPath+"/"+first, nil); status != http.StatusNoContent {
		t.Fatalf("delete: %d", status)
	}
	if status, body := testutil.Do(t, handler, http.MethodDelete, applicationsPath+"/"+first, nil); status != http.StatusNotFound ||
		testutil.ErrorCode(body) != "NOT_FOUND" {
		t.Fatalf("delete again: %d %v", status, body)
	}

	other := identity.New(uuid.New(), uuid.NewString())
	stranger := applicationsHandler(repo, &other)
	if _, list := testutil.Do(t, stranger, http.MethodGet, applicationsPath, nil); fmt.Sprint(list["items"]) != "[]" {
		t.Fatalf("another person sees nothing: %v", list)
	}
	if status, _ := testutil.Do(t, stranger, http.MethodDelete, applicationsPath+"/"+second, nil); status != http.StatusNotFound {
		t.Fatalf("another person cannot delete: %d", status)
	}
}

func TestSkipsThroughRoutes(t *testing.T) {
	handler := devApplications(newMemoryApplications())
	first, second := uuid.NewString(), uuid.NewString()
	for _, job := range []string{first, second, second} {
		if status, _ := testutil.Do(t, handler, http.MethodPut, skipsPath+"/"+job, nil); status != http.StatusNoContent {
			t.Fatalf("skip %s: %d", job, status)
		}
	}
	if status, list := testutil.Do(t, handler, http.MethodGet, skipsPath, nil); status != http.StatusOK || fmt.Sprint(list["job_ids"]) != fmt.Sprint([]string{second, first}) {
		t.Fatalf("skips: %d %v", status, list)
	}
	for range 2 {
		if status, _ := testutil.Do(t, handler, http.MethodDelete, skipsPath+"/"+second, nil); status != http.StatusNoContent {
			t.Fatalf("unskip: %d", status)
		}
	}
	if status, _ := testutil.Do(t, handler, http.MethodDelete, skipsPath, nil); status != http.StatusNoContent {
		t.Fatalf("clear: %d", status)
	}
	if _, list := testutil.Do(t, handler, http.MethodGet, skipsPath, nil); fmt.Sprint(list["job_ids"]) != "[]" {
		t.Fatalf("after clear: %v", list)
	}
}

func TestApplicationRequestValidation(t *testing.T) {
	handler := devApplications(newMemoryApplications())
	job := uuid.NewString()
	for _, tc := range []struct {
		method, path string
		payload      any
		field        any
	}{
		{http.MethodPut, applicationsPath + "/" + job, map[string]any{"status": "rejected"}, "status"},
		{http.MethodPut, applicationsPath + "/" + job, map[string]any{}, "status"},
		{http.MethodPut, applicationsPath + "/" + job, map[string]any{"status": 3}, "status"},
		{http.MethodPut, applicationsPath + "/" + job, "{", nil},
		{http.MethodPut, applicationsPath + "/not-a-uuid", map[string]any{"status": "saved"}, "job_id"},
		{http.MethodDelete, applicationsPath + "/42", nil, "job_id"},
		{http.MethodPut, skipsPath + "/nope", nil, "job_id"},
		{http.MethodDelete, skipsPath + "/nope", nil, "job_id"},
	} {
		status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload)
		if status != http.StatusUnprocessableEntity || testutil.ErrorCode(body) != "VALIDATION_FAILED" || errorField(body) != tc.field {
			t.Errorf("%s %s %v: %d %v", tc.method, tc.path, tc.payload, status, body)
		}
	}
}

func everyApplicationRoute() []struct {
	method, path string
	payload      any
} {
	job := uuid.NewString()
	return []struct {
		method, path string
		payload      any
	}{
		{http.MethodGet, applicationsPath, nil},
		{http.MethodPut, applicationsPath + "/" + job, map[string]any{"status": "saved"}},
		{http.MethodDelete, applicationsPath + "/" + job, nil},
		{http.MethodGet, skipsPath, nil},
		{http.MethodPut, skipsPath + "/" + job, nil},
		{http.MethodDelete, skipsPath + "/" + job, nil},
		{http.MethodDelete, skipsPath, nil},
	}
}

func TestApplicationsRequireAnIdentity(t *testing.T) {
	handler := applicationsHandler(newMemoryApplications(), nil)
	for _, tc := range everyApplicationRoute() {
		if status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload); status != http.StatusUnauthorized || testutil.ErrorCode(body) != "AUTH_REQUIRED" {
			t.Errorf("%s %s without identity: %d %v", tc.method, tc.path, status, body)
		}
	}
}

func TestApplicationRepositoryFailuresSurface(t *testing.T) {
	repo := newMemoryApplications()
	repo.fail = apperrors.New(apperrors.DatabaseUnavailable, "down")
	handler := devApplications(repo)
	for _, tc := range everyApplicationRoute() {
		if status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload); status != http.StatusServiceUnavailable {
			t.Errorf("%s %s: %d %v", tc.method, tc.path, status, body)
		}
	}
}
