package unit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"cv/internal/apperrors"
	"cv/internal/data/repositories"
	"cv/internal/identity"
	"cv/internal/models"
	"cv/internal/routes"
	"cv/internal/services"
	"cv/internal/writer"
	"cv/tests/testutil"
)

const (
	cvsPath = "/api/v1/cvs"
	jobID   = "00000000-0000-4000-8000-000000000001"
)

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

// memoryCvs is an in-memory CvRepository keyed by owner, newest first.
type memoryCvs struct {
	mu   sync.Mutex
	cvs  map[string][]models.Cv
	last repositories.NewCv
	fail error
}

func newMemoryCvs() *memoryCvs { return &memoryCvs{cvs: map[string][]models.Cv{}} }

func ownerKey(ident identity.IdentityContext) string {
	return ident.TenantID.String() + "/" + ident.Subject
}

func (m *memoryCvs) Create(_ context.Context, ident identity.IdentityContext, cv repositories.NewCv) (*models.Cv, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	m.last = cv
	now := time.Now()
	created := models.Cv{ID: uuid.New(), JobID: cv.JobID, Status: models.StatusQueued, CreatedAt: now, UpdatedAt: now,
		JobTitle: cv.JobTitle, Company: cv.Company, FileName: cv.FileName}
	m.cvs[ownerKey(ident)] = append([]models.Cv{created}, m.cvs[ownerKey(ident)]...)
	return &created, nil
}

func (m *memoryCvs) Get(_ context.Context, ident identity.IdentityContext, id uuid.UUID) (*models.Cv, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	for _, cv := range m.cvs[ownerKey(ident)] {
		if cv.ID == id {
			return &cv, nil
		}
	}
	return nil, nil
}

func (m *memoryCvs) List(_ context.Context, ident identity.IdentityContext, jobID *uuid.UUID) ([]models.Cv, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	items := []models.Cv{}
	for _, cv := range m.cvs[ownerKey(ident)] {
		if jobID == nil || cv.JobID == *jobID {
			items = append(items, cv)
		}
	}
	return items, nil
}

func (m *memoryCvs) Delete(_ context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return false, m.fail
	}
	list := m.cvs[ownerKey(ident)]
	kept := slices.DeleteFunc(slices.Clone(list), func(cv models.Cv) bool { return cv.ID == id })
	m.cvs[ownerKey(ident)] = kept
	return len(kept) < len(list), nil
}

// fakeSources serves the sample profile and the seed jobs.
type fakeSources struct {
	profile    models.Profile
	jobs       map[uuid.UUID]models.Job
	profileErr error
	jobErr     error
}

func newFakeSources(t *testing.T) *fakeSources {
	job := catalogJob(t, 0)
	return &fakeSources{profile: sampleProfile(t), jobs: map[uuid.UUID]models.Job{job.ID: job}}
}

func (f *fakeSources) Profile(context.Context) (models.Profile, json.RawMessage, error) {
	if f.profileErr != nil {
		return models.Profile{}, nil, f.profileErr
	}
	raw, _ := json.Marshal(f.profile)
	return f.profile, raw, nil
}

func (f *fakeSources) Job(_ context.Context, id uuid.UUID) (models.Job, json.RawMessage, error) {
	if f.jobErr != nil {
		return models.Job{}, nil, f.jobErr
	}
	job, ok := f.jobs[id]
	if !ok {
		return job, nil, apperrors.NewNotFound("Job", id.String())
	}
	raw, _ := json.Marshal(job)
	return job, raw, nil
}

type cvHarness struct {
	repo    *memoryCvs
	sources *fakeSources
	queued  int
}

func (h *cvHarness) handler(ident *identity.IdentityContext) http.Handler {
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Cvs: services.NewCvService(h.repo, h.sources, func() { h.queued++ }), Writer: writer.New(nil)})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ident != nil {
			r = r.WithContext(identity.WithContext(r.Context(), *ident))
		}
		router.ServeHTTP(w, r)
	})
}

func newCvHarness(t *testing.T) (*cvHarness, http.Handler) {
	h := &cvHarness{repo: newMemoryCvs(), sources: newFakeSources(t)}
	dev := identity.Dev()
	return h, h.handler(&dev)
}

func errorField(body map[string]any) any {
	envelope, _ := body["error"].(map[string]any)
	details, _ := envelope["context"].(map[string]any)
	return details["field"]
}

func TestQuestionsThroughRoutes(t *testing.T) {
	_, handler := newCvHarness(t)
	status, body := testutil.Do(t, handler, http.MethodGet, "/api/v1/jobs/"+jobID+"/questions", nil)
	if status != http.StatusOK || body["job_id"] != jobID || body["base_score"] != float64(56) {
		t.Fatalf("questions: %d %v", status, body)
	}
	if gaps := body["gaps"].([]any); len(gaps) != 4 || gaps[0].(map[string]any)["skill"] != "Snowflake" {
		t.Fatalf("gaps: %v", gaps)
	}
	if status, body := testutil.Do(t, handler, http.MethodGet, "/api/v1/jobs/"+uuid.NewString()+"/questions", nil); status != http.StatusNotFound ||
		testutil.ErrorCode(body) != "NOT_FOUND" {
		t.Fatalf("unknown job: %d %v", status, body)
	}
}

func TestCvsThroughRoutes(t *testing.T) {
	h, handler := newCvHarness(t)
	if status, list := testutil.Do(t, handler, http.MethodGet, cvsPath, nil); status != http.StatusOK || fmt.Sprint(list["items"]) != "[]" {
		t.Fatalf("empty list: %d %v", status, list)
	}
	status, cv := testutil.Do(t, handler, http.MethodPost, cvsPath, map[string]any{"job_id": " " + jobID + " ", "focus": []string{"SQL"}})
	if status != http.StatusAccepted || cv["status"] != "queued" || cv["file_name"] != "Andrei_Popescu_CV_Capgemini_Romania.pdf" ||
		cv["job_title"] != "Snowflake Administrator DevOps" || cv["company"] != "Capgemini Romania" || cv["tailored"] != nil {
		t.Fatalf("create: %d %v", status, cv)
	}
	if h.queued != 1 {
		t.Fatalf("the worker is woken once: %d", h.queued)
	}
	request := h.repo.last.Request
	if request.JobID != jobID || request.SkipRoles == nil || request.Confirmed == nil || len(request.Focus) != 1 {
		t.Fatalf("the request is stored clean: %+v", request)
	}
	var snapshot models.Profile
	if err := json.Unmarshal(h.repo.last.ProfileSnapshot, &snapshot); err != nil || snapshot.Name != "Andrei Popescu" {
		t.Fatalf("profile snapshot: %v %v", snapshot.Name, err)
	}
	id := cv["id"].(string)
	if status, got := testutil.Do(t, handler, http.MethodGet, cvsPath+"/"+id, nil); status != http.StatusOK || got["id"] != id {
		t.Fatalf("get: %d %v", status, got)
	}
	for _, query := range []string{"", "?job_id=" + jobID} {
		if _, list := testutil.Do(t, handler, http.MethodGet, cvsPath+query, nil); len(list["items"].([]any)) != 1 {
			t.Fatalf("list %q: %v", query, list)
		}
	}
	if _, list := testutil.Do(t, handler, http.MethodGet, cvsPath+"?job_id="+uuid.NewString(), nil); fmt.Sprint(list["items"]) != "[]" {
		t.Fatalf("another job's CVs: %v", list)
	}

	other := identity.New(uuid.New(), uuid.NewString())
	stranger := h.handler(&other)
	if status, _ := testutil.Do(t, stranger, http.MethodGet, cvsPath+"/"+id, nil); status != http.StatusNotFound {
		t.Fatalf("another person cannot read it: %d", status)
	}
	if status, _ := testutil.Do(t, stranger, http.MethodDelete, cvsPath+"/"+id, nil); status != http.StatusNotFound {
		t.Fatalf("another person cannot delete it: %d", status)
	}
	if status, _ := testutil.Do(t, handler, http.MethodDelete, cvsPath+"/"+id, nil); status != http.StatusNoContent {
		t.Fatalf("delete: %d", status)
	}
	if status, body := testutil.Do(t, handler, http.MethodGet, cvsPath+"/"+id, nil); status != http.StatusNotFound || testutil.ErrorCode(body) != "NOT_FOUND" {
		t.Fatalf("get deleted: %d %v", status, body)
	}
}

func TestWriterStatusThroughRoutes(t *testing.T) {
	_, handler := newCvHarness(t)
	status, body := testutil.Do(t, handler, http.MethodGet, "/api/v1/writer", nil)
	if status != http.StatusOK || body["writer"] != "rules" || body["state"] != "off" {
		t.Fatalf("writer: %d %v", status, body)
	}
}

func TestCvRequestValidation(t *testing.T) {
	h, handler := newCvHarness(t)
	for _, tc := range []struct {
		method, path string
		payload      any
		field        any
	}{
		{http.MethodPost, cvsPath, map[string]any{"job_id": "nope"}, "job_id"},
		{http.MethodPost, cvsPath, map[string]any{}, "job_id"},
		{http.MethodPost, cvsPath, map[string]any{"job_id": jobID, "focus": []string{"a", "b", "c", "d"}}, "focus"},
		{http.MethodPost, cvsPath, map[string]any{"job_id": jobID, "confirmed": []int{-1}}, "confirmed"},
		{http.MethodPost, cvsPath, map[string]any{"job_id": jobID, "confirmed": "one"}, nil},
		{http.MethodPost, cvsPath, "{", nil},
		{http.MethodGet, cvsPath + "?job_id=nope", nil, "job_id"},
		{http.MethodGet, cvsPath + "/nope", nil, "id"},
		{http.MethodDelete, cvsPath + "/nope", nil, "id"},
		{http.MethodGet, "/api/v1/jobs/nope/questions", nil, "job_id"},
	} {
		status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload)
		if status != http.StatusUnprocessableEntity || testutil.ErrorCode(body) != "VALIDATION_FAILED" || errorField(body) != tc.field {
			t.Errorf("%s %s %v: %d %v", tc.method, tc.path, tc.payload, status, body)
		}
	}
	if h.queued != 0 {
		t.Fatalf("nothing invalid is queued: %d", h.queued)
	}
}

func TestCvSourcesFailures(t *testing.T) {
	h, handler := newCvHarness(t)
	if status, _ := testutil.Do(t, handler, http.MethodPost, cvsPath, map[string]any{"job_id": uuid.NewString()}); status != http.StatusNotFound {
		t.Fatalf("unknown job: %d", status)
	}
	h.sources.profileErr = apperrors.NewDependencyUnavailable("profile")
	for _, tc := range []struct{ method, path string }{{http.MethodPost, cvsPath}, {http.MethodGet, "/api/v1/jobs/" + jobID + "/questions"}} {
		status, body := testutil.Do(t, handler, tc.method, tc.path, map[string]any{"job_id": jobID})
		if status != http.StatusServiceUnavailable || testutil.ErrorCode(body) != "DEPENDENCY_UNAVAILABLE" {
			t.Fatalf("%s without profile: %d %v", tc.path, status, body)
		}
	}
	h.sources.jobErr = apperrors.NewDependencyUnavailable("jobs")
	if status, _ := testutil.Do(t, handler, http.MethodGet, "/api/v1/jobs/"+jobID+"/questions", nil); status != http.StatusServiceUnavailable {
		t.Fatalf("without jobs: %d", status)
	}
	if h.queued != 0 {
		t.Fatalf("nothing is queued: %d", h.queued)
	}
}

func everyCvRoute() []struct {
	method, path string
	payload      any
} {
	id := uuid.NewString()
	return []struct {
		method, path string
		payload      any
	}{
		{http.MethodGet, "/api/v1/jobs/" + jobID + "/questions", nil},
		{http.MethodGet, cvsPath, nil},
		{http.MethodPost, cvsPath, map[string]any{"job_id": jobID}},
		{http.MethodGet, cvsPath + "/" + id, nil},
		{http.MethodDelete, cvsPath + "/" + id, nil},
	}
}

func TestCvsRequireAnIdentity(t *testing.T) {
	h, _ := newCvHarness(t)
	handler := h.handler(nil)
	for _, tc := range everyCvRoute() {
		if status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload); status != http.StatusUnauthorized || testutil.ErrorCode(body) != "AUTH_REQUIRED" {
			t.Errorf("%s %s without identity: %d %v", tc.method, tc.path, status, body)
		}
	}
}

func TestCvRepositoryFailuresSurface(t *testing.T) {
	h, handler := newCvHarness(t)
	h.repo.fail = apperrors.New(apperrors.DatabaseUnavailable, "down")
	for _, tc := range everyCvRoute()[1:] {
		if status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload); status != http.StatusServiceUnavailable {
			t.Errorf("%s %s: %d %v", tc.method, tc.path, status, body)
		}
	}
}
