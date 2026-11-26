package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"cv/internal/apperrors"
	"cv/internal/cvdoc"
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
	printer *fakePrinter
	queued  int
}

// fakePrinter plays Gotenberg.
type fakePrinter struct {
	page string
	err  error
}

func (p *fakePrinter) PDF(_ context.Context, page string) ([]byte, error) {
	p.page = page
	return []byte("%PDF-fake"), p.err
}

// ready marks the caller's CV ready with the sample profile's documents.
func (m *memoryCvs) ready(t *testing.T, id string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	written := cvdoc.Write(sampleProfile(t), engineJob(t, 0), request(), now)
	for owner, list := range m.cvs {
		for i := range list {
			if list[i].ID.String() == id {
				list[i].Status = models.StatusReady
				list[i].Tailored, list[i].Profile = &written.Tailored, &written.Profile
			}
		}
		m.cvs[owner] = list
	}
}

func (h *cvHarness) handler(ident *identity.IdentityContext) http.Handler {
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Cvs: services.NewCvService(h.repo, h.sources, h.printer, func() { h.queued++ }), Writer: writer.New(nil)})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ident != nil {
			r = r.WithContext(identity.WithContext(r.Context(), *ident))
		}
		router.ServeHTTP(w, r)
	})
}

func newCvHarness(t *testing.T) (*cvHarness, http.Handler) {
	h := &cvHarness{repo: newMemoryCvs(), sources: newFakeSources(t), printer: &fakePrinter{}}
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

func TestCvFilesThroughRoutes(t *testing.T) {
	h, handler := newCvHarness(t)
	_, cv := testutil.Do(t, handler, http.MethodPost, cvsPath, map[string]any{"job_id": jobID})
	file := cvsPath + "/" + cv["id"].(string) + "/file"
	if status, body := testutil.Do(t, handler, http.MethodGet, file+"?format=pdf", nil); status != http.StatusConflict || testutil.ErrorCode(body) != "CONSTRAINT_VIOLATION" {
		t.Fatalf("not ready: %d %v", status, body)
	}
	h.repo.ready(t, cv["id"].(string))
	for _, tc := range []struct{ query, contentType, name, prefix string }{
		{"?format=pdf", "application/pdf", "Andrei_Popescu_CV_Capgemini_Romania.pdf", "%PDF"},
		{"?format=pdf&version=profile", "application/pdf", "Andrei_Popescu_CV_Capgemini_Romania.pdf", "%PDF"},
		{"?format=docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "Andrei_Popescu_CV_Capgemini_Romania.docx", "PK"},
	} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, file+tc.query, nil))
		if res.Code != http.StatusOK || res.Header().Get("Content-Type") != tc.contentType ||
			res.Header().Get("Content-Disposition") != `attachment; filename=`+tc.name || !strings.HasPrefix(res.Body.String(), tc.prefix) {
			t.Fatalf("%s: %d %v %q", tc.query, res.Code, res.Header(), res.Body.String()[:min(20, res.Body.Len())])
		}
	}
	if !strings.Contains(h.printer.page, "<h2>Summary</h2>") {
		t.Fatalf("the printer gets the page: %q", h.printer.page)
	}
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, file+"?format=pdf&template=elegant", nil))
	if !strings.Contains(h.printer.page, "Caladea") {
		t.Fatal("the printer gets the chosen template")
	}
	for _, tc := range []struct{ query, field string }{{"?format=txt", "format"}, {"", "format"}, {"?format=pdf&version=draft", "version"}, {"?format=docx&template=fancy", "template"}} {
		if status, body := testutil.Do(t, handler, http.MethodGet, file+tc.query, nil); status != http.StatusUnprocessableEntity || errorField(body) != tc.field {
			t.Fatalf("%q: %d %v", tc.query, status, body)
		}
	}
	if status, _ := testutil.Do(t, handler, http.MethodGet, cvsPath+"/"+uuid.NewString()+"/file?format=pdf", nil); status != http.StatusNotFound {
		t.Fatalf("unknown CV: %d", status)
	}
	h.printer.err = apperrors.NewDependencyUnavailable("gotenberg")
	if status, _ := testutil.Do(t, handler, http.MethodGet, file+"?format=pdf", nil); status != http.StatusServiceUnavailable {
		t.Fatalf("no Gotenberg: %d", status)
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

// memoryProbes is an in-memory Probes store.
type memoryProbes struct {
	mu    sync.Mutex
	kept  map[string]repositories.StoredProbe
	saves int
}

func (m *memoryProbes) Probe(_ context.Context, ident identity.IdentityContext, jobID uuid.UUID) (*repositories.StoredProbe, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if probe, ok := m.kept[ownerKey(ident)+jobID.String()]; ok {
		return &probe, nil
	}
	return nil, nil
}

func (m *memoryProbes) SaveProbe(_ context.Context, ident identity.IdentityContext, jobID uuid.UUID, probe repositories.StoredProbe) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saves++
	probe.UpdatedAt = time.Now()
	m.kept[ownerKey(ident)+jobID.String()] = probe
	return nil
}

// fakeProber answers with its questions, or fails.
type fakeProber struct {
	mu           sync.Mutex
	calls        int
	profileCalls int
	analyzed     int
	fail         bool
	available    bool
	release      chan struct{}
}

func (f *fakeProber) Questions(context.Context, models.Job, models.Profile) ([]models.ProbeQuestion, error) {
	if f.release != nil {
		<-f.release
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.fail {
		return nil, errors.New("no")
	}
	return []models.ProbeQuestion{{Theme: "CI/CD", Question: "How many services?", Place: "experience:r1"}}, nil
}

func (f *fakeProber) ProfileQuestions(context.Context, models.Profile) ([]models.ProbeQuestion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.profileCalls++
	if f.fail {
		return nil, errors.New("no")
	}
	return []models.ProbeQuestion{{Theme: "Infrastructure", Question: "How many accounts?", Place: "experience:r1", Choices: []string{"Under 5 accounts"}}}, nil
}

func (f *fakeProber) Analyze(context.Context, models.Job) error {
	if f.release != nil {
		<-f.release
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.analyzed++
	if f.fail {
		return errors.New("no")
	}
	return nil
}

func (f *fakeProber) Available() bool { return f.available }

func TestAnalysisStartsInTheBackground(t *testing.T) {
	h := &cvHarness{repo: newMemoryCvs(), sources: newFakeSources(t), printer: &fakePrinter{}}
	prober := &fakeProber{available: true, release: make(chan struct{})}
	service := services.NewCvService(h.repo, h.sources, h.printer, func() {}).WithProbes(&memoryProbes{kept: map[string]repositories.StoredProbe{}}, prober)
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Cvs: service, Writer: writer.New(nil)})
	dev := identity.Dev()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r.WithContext(identity.WithContext(r.Context(), dev)))
	})
	path := "/api/v1/jobs/" + jobID + "/analysis"
	for range 2 {
		if status, body := testutil.Do(t, handler, http.MethodPost, path, nil); status != http.StatusAccepted || body["status"] != "pending" {
			t.Fatalf("started: %d %v", status, body)
		}
	}
	close(prober.release)
	deadline := time.Now().Add(5 * time.Second)
	for {
		prober.mu.Lock()
		analyzed := prober.analyzed
		prober.mu.Unlock()
		if analyzed == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the analysis never ran")
		}
		time.Sleep(10 * time.Millisecond)
	}
	prober.available = false
	if _, body := testutil.Do(t, handler, http.MethodPost, path, nil); body["status"] != "unavailable" {
		t.Fatalf("Claude unavailable: %v", body)
	}
	if status, _ := testutil.Do(t, handler, http.MethodPost, "/api/v1/jobs/"+uuid.NewString()+"/analysis", nil); status != http.StatusNotFound {
		t.Fatalf("unknown job: %d", status)
	}
	if status, _ := testutil.Do(t, handler, http.MethodPost, "/api/v1/jobs/nope/analysis", nil); status != http.StatusUnprocessableEntity {
		t.Fatalf("bad id: %d", status)
	}
	_, plain := newCvHarness(t)
	if _, body := testutil.Do(t, plain, http.MethodPost, path, nil); body["status"] != "unavailable" {
		t.Fatalf("no prober: %v", body)
	}
}

func TestProbeIsAskedInTheBackgroundAndKeptForTheFacts(t *testing.T) {
	h := &cvHarness{repo: newMemoryCvs(), sources: newFakeSources(t), printer: &fakePrinter{}}
	store := &memoryProbes{kept: map[string]repositories.StoredProbe{}}
	prober := &fakeProber{available: true, release: make(chan struct{})}
	service := services.NewCvService(h.repo, h.sources, h.printer, func() {}).WithProbes(store, prober)
	settled := make(chan struct{}, 4)
	service.OnProbeSettled(func() { settled <- struct{}{} })
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Cvs: service, Writer: writer.New(nil)})
	dev := identity.Dev()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r.WithContext(identity.WithContext(r.Context(), dev)))
	})
	path := "/api/v1/jobs/" + jobID + "/probe"

	for range 2 {
		if status, body := testutil.Do(t, handler, http.MethodGet, path, nil); status != http.StatusOK || body["status"] != "pending" {
			t.Fatalf("pending while asked: %d %v", status, body)
		}
		// While they are asked, a poll reads neither the job nor the profile.
		h.sources.jobErr, h.sources.profileErr = errors.New("jobs down"), errors.New("profile down")
	}
	h.sources.jobErr, h.sources.profileErr = nil, nil
	close(prober.release)
	<-settled
	status, body := testutil.Do(t, handler, http.MethodGet, path, nil)
	if status != http.StatusOK || body["status"] != "ready" || len(body["questions"].([]any)) != 1 {
		t.Fatalf("ready: %d %v", status, body)
	}
	equal(t, prober.calls, 1)

	h.sources.profile.Experience[0].Notes = []models.Note{{Text: "Used by 40 services"}}
	if _, body := testutil.Do(t, handler, http.MethodGet, path, nil); body["status"] != "pending" {
		t.Fatalf("new facts are asked about again: %v", body)
	}
	<-settled
	equal(t, prober.calls, 2)

	prober.fail = true
	h.sources.profile.Experience[0].Notes = nil
	testutil.Do(t, handler, http.MethodGet, path, nil)
	<-settled
	if _, body := testutil.Do(t, handler, http.MethodGet, path, nil); body["status"] != "unavailable" {
		t.Fatalf("a failure is unavailable for a while: %v", body)
	}
	equal(t, prober.calls, 3)

	prober.available = false
	h.sources.profile.Skills = append(h.sources.profile.Skills, "Go")
	if _, body := testutil.Do(t, handler, http.MethodGet, path, nil); body["status"] != "unavailable" {
		t.Fatalf("Claude unavailable: %v", body)
	}
	if status, _ := testutil.Do(t, handler, http.MethodGet, "/api/v1/jobs/"+uuid.NewString()+"/probe", nil); status != http.StatusNotFound {
		t.Fatalf("unknown job: %d", status)
	}
	if status, _ := testutil.Do(t, handler, http.MethodGet, "/api/v1/jobs/nope/probe", nil); status != http.StatusUnprocessableEntity {
		t.Fatalf("bad id: %d", status)
	}
}

func TestProbeIsUnavailableWithoutAProber(t *testing.T) {
	_, handler := newCvHarness(t)
	if _, body := testutil.Do(t, handler, http.MethodGet, "/api/v1/jobs/"+jobID+"/probe", nil); body["status"] != "unavailable" {
		t.Fatalf("no prober: %v", body)
	}
}

func TestProfileProbeIsAskedOnceForEveryJob(t *testing.T) {
	h := &cvHarness{repo: newMemoryCvs(), sources: newFakeSources(t), printer: &fakePrinter{}}
	store := &memoryProbes{kept: map[string]repositories.StoredProbe{}}
	prober := &fakeProber{available: true}
	service := services.NewCvService(h.repo, h.sources, h.printer, func() {}).WithProbes(store, prober)
	settled := make(chan struct{}, 4)
	service.OnProbeSettled(func() { settled <- struct{}{} })
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Cvs: service, Writer: writer.New(nil)})
	dev := identity.Dev()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r.WithContext(identity.WithContext(r.Context(), dev)))
	})
	if _, body := testutil.Do(t, handler, http.MethodGet, "/api/v1/probe", nil); body["status"] != "pending" {
		t.Fatalf("pending: %v", body)
	}
	<-settled
	status, body := testutil.Do(t, handler, http.MethodGet, "/api/v1/probe", nil)
	question := body["questions"].([]any)[0].(map[string]any)
	if status != http.StatusOK || body["status"] != "ready" || question["choices"].([]any)[0] != "Under 5 accounts" {
		t.Fatalf("ready: %d %v", status, body)
	}
	equal(t, prober.profileCalls, 1)
	equal(t, prober.calls, 0)
	h.sources.profileErr = apperrors.NewDependencyUnavailable("profile")
	if status, _ := testutil.Do(t, handler, http.MethodGet, "/api/v1/probe", nil); status != http.StatusServiceUnavailable {
		t.Fatalf("no profile: %d", status)
	}
	_, plain := newCvHarness(t)
	if _, body := testutil.Do(t, plain, http.MethodGet, "/api/v1/probe", nil); body["status"] != "unavailable" {
		t.Fatalf("no prober: %v", body)
	}
}
