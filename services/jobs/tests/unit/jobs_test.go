package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"jobs/internal/apperrors"
	"jobs/internal/fetch"
	"jobs/internal/identity"
	"jobs/internal/models"
	"jobs/internal/routes"
	"jobs/internal/services"
	"jobs/tests/testutil"
)

const jobsPath = "/api/v1/jobs"

// memoryJobs is an in-memory JobRepository: a catalog plus owned jobs. It
// records the last filter instead of applying it; SQL filtering is covered by
// the integration suite.
type memoryJobs struct {
	mu     sync.Mutex
	jobs   []models.Job
	owners map[uuid.UUID]string
	filter models.JobFilter
	fail   error
}

func newMemoryJobs() *memoryJobs {
	catalog := models.Job{ID: uuid.New(), Title: "Catalog job", Source: "catalog", Sample: true, MustHave: []string{"Go"},
		NiceToHave: []string{}, Responsibilities: []string{}, Perks: []string{}, Areas: []string{"Backend"}, CreatedAt: time.Now()}
	return &memoryJobs{jobs: []models.Job{catalog}, owners: map[uuid.UUID]string{}}
}

func owner(ident identity.IdentityContext) string {
	return ident.TenantID.String() + "/" + ident.Subject
}

func (m *memoryJobs) visible(ident identity.IdentityContext, job models.Job) bool {
	o, owned := m.owners[job.ID]
	return !owned || o == owner(ident)
}

func (m *memoryJobs) List(_ context.Context, ident identity.IdentityContext, filter models.JobFilter) ([]models.JobCard, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	m.filter = filter
	jobs := []models.JobCard{}
	for _, job := range m.jobs {
		if m.visible(ident, job) {
			jobs = append(jobs, models.CardOf(job))
		}
	}
	if filter.Limit > 0 && len(jobs) > filter.Limit {
		jobs = jobs[:filter.Limit]
	}
	return jobs, nil
}

func (m *memoryJobs) Count(_ context.Context, ident identity.IdentityContext, _ models.JobFilter) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	total := 0
	for _, job := range m.jobs {
		if m.visible(ident, job) {
			total++
		}
	}
	return total, nil
}

func (m *memoryJobs) Get(_ context.Context, ident identity.IdentityContext, id uuid.UUID) (*models.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	for _, job := range m.jobs {
		if job.ID == id && m.visible(ident, job) {
			return &job, nil
		}
	}
	return nil, nil
}

func (m *memoryJobs) Create(_ context.Context, ident identity.IdentityContext, d models.JobInput) (*models.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	job := models.Job{ID: uuid.New(), Title: d.Title, Company: d.Company, City: d.City, Country: d.Country, WorkMode: d.WorkMode,
		Level: d.Level, YearsRequired: d.YearsRequired, MustHave: d.MustHave, NiceToHave: d.NiceToHave,
		Responsibilities: d.Responsibilities, About: d.About, Perks: []string{}, Areas: []string{}, URL: d.URL, Source: d.Source,
		ReadFrom: d.ReadFrom, AddedByMe: true, CreatedAt: time.Now()}
	m.jobs = append([]models.Job{job}, m.jobs...)
	m.owners[job.ID] = owner(ident)
	return &job, nil
}

func (m *memoryJobs) SetRequirements(_ context.Context, ident identity.IdentityContext, id uuid.UUID, data models.RequirementsInput) (*models.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	if o, owned := m.owners[id]; !owned || o != owner(ident) {
		return nil, nil
	}
	for i := range m.jobs {
		if m.jobs[i].ID == id {
			m.jobs[i].MustHave, m.jobs[i].NiceToHave, m.jobs[i].RequirementsFrom = data.MustHave, data.NiceToHave, "analysis"
			job := m.jobs[i]
			return &job, nil
		}
	}
	return nil, nil
}

func (m *memoryJobs) Delete(_ context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return false, m.fail
	}
	if o, owned := m.owners[id]; !owned || o != owner(ident) {
		return false, nil
	}
	before := len(m.jobs)
	m.jobs = slices.DeleteFunc(m.jobs, func(job models.Job) bool { return job.ID == id })
	return len(m.jobs) < before, nil
}

// fakeReader answers every link with page or err.
type fakeReader struct {
	page *fetch.Page
	err  error
	url  string
}

func (f *fakeReader) Read(_ context.Context, rawURL string) (*fetch.Page, error) {
	f.url = rawURL
	return f.page, f.err
}

func jobsHandler(repo *memoryJobs, reader services.PageReader, ident *identity.IdentityContext) http.Handler {
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Jobs: services.NewJobService(repo, reader)})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ident != nil {
			r = r.WithContext(identity.WithContext(r.Context(), *ident))
		}
		router.ServeHTTP(w, r)
	})
}

func devJobs(repo *memoryJobs, reader services.PageReader) http.Handler {
	dev := identity.Dev()
	return jobsHandler(repo, reader, &dev)
}

func errorContext(body map[string]any, key string) any {
	envelope, _ := body["error"].(map[string]any)
	details, _ := envelope["context"].(map[string]any)
	return details[key]
}

func validJob() map[string]any {
	return map[string]any{"title": "Platform Engineer", "source": "text", "read_from": "text"}
}

func with(base map[string]any, key string, value any) map[string]any {
	copied := map[string]any{}
	for k, v := range base {
		copied[k] = v
	}
	copied[key] = value
	return copied
}

func TestCleanJobAcceptsEveryReadFrom(t *testing.T) {
	for _, readFrom := range []string{"ats_api", "json_ld", "page_text", "text"} {
		if _, err := services.CleanJob(models.JobInput{Title: "Platform Engineer", Source: "link", ReadFrom: readFrom}); err != nil {
			t.Errorf("read_from %q refused: %v", readFrom, err)
		}
	}
}

func TestCleanJob(t *testing.T) {
	yearsValue := 3
	cleaned, err := services.CleanJob(models.JobInput{
		Title: "  Platform Engineer ", Company: " Northwind ", City: " Cluj-Napoca ", Country: " Romania ", WorkMode: " hybrid ",
		Level: "mid", YearsRequired: &yearsValue, MustHave: []string{" Go ", "go", "", "Kubernetes"},
		NiceToHave: []string{"KUBERNETES", " Kafka ", "kafka"}, Responsibilities: []string{" Run clusters ", " ", "Run clusters"},
		About: "  About  ", URL: " https://jobs.example.com/1 ", Source: " link ", ReadFrom: "json_ld",
	})
	if err != nil || cleaned.Title != "Platform Engineer" || cleaned.Company != "Northwind" || cleaned.WorkMode != "hybrid" ||
		!slices.Equal(cleaned.MustHave, []string{"Go", "Kubernetes"}) || !slices.Equal(cleaned.NiceToHave, []string{"Kafka"}) ||
		!slices.Equal(cleaned.Responsibilities, []string{"Run clusters", "Run clusters"}) || cleaned.About != "About" ||
		cleaned.URL != "https://jobs.example.com/1" || cleaned.Source != "link" {
		t.Fatalf("cleaned job: %+v %v", cleaned, err)
	}
	if cleaned, err := services.CleanJob(models.JobInput{Title: "t", Source: "text", ReadFrom: "text"}); err != nil ||
		cleaned.MustHave == nil || cleaned.NiceToHave == nil || cleaned.Responsibilities == nil {
		t.Fatalf("missing lists become empty lists: %+v %v", cleaned, err)
	}
	forty, fortyOne := make([]string, 40), make([]string, 41)
	thirtyOne := make([]string, 31)
	for i := range fortyOne {
		fortyOne[i] = fmt.Sprintf("skill %d", i)
	}
	copy(forty, fortyOne)
	for i := range thirtyOne {
		thirtyOne[i] = "duty"
	}
	ok := func(mutate func(*models.JobInput)) models.JobInput {
		input := models.JobInput{Title: "t", Source: "text", ReadFrom: "text"}
		mutate(&input)
		return input
	}
	if _, err := services.CleanJob(ok(func(j *models.JobInput) {
		j.Title, j.MustHave, j.NiceToHave, j.Responsibilities = strings.Repeat("t", 160), forty, []string{strings.Repeat("n", 60)}, thirtyOne[:30]
		j.About, j.URL, j.YearsRequired = strings.Repeat("a", 4000), "http://x.ro/"+strings.Repeat("p", 1988), new(int)
	})); err != nil {
		t.Fatalf("limits are inclusive: %v", err)
	}
	forty1 := 41
	minus := -1
	for _, tc := range []struct {
		field  string
		mutate func(*models.JobInput)
	}{
		{"title", func(j *models.JobInput) { j.Title = " " }},
		{"title", func(j *models.JobInput) { j.Title = strings.Repeat("t", 161) }},
		{"company", func(j *models.JobInput) { j.Company = strings.Repeat("c", 121) }},
		{"city", func(j *models.JobInput) { j.City = strings.Repeat("c", 121) }},
		{"country", func(j *models.JobInput) { j.Country = strings.Repeat("c", 121) }},
		{"work_mode", func(j *models.JobInput) { j.WorkMode = "office" }},
		{"level", func(j *models.JobInput) { j.Level = "Senior" }},
		{"years_required", func(j *models.JobInput) { j.YearsRequired = &forty1 }},
		{"years_required", func(j *models.JobInput) { j.YearsRequired = &minus }},
		{"must_have", func(j *models.JobInput) { j.MustHave = fortyOne }},
		{"must_have", func(j *models.JobInput) { j.MustHave = []string{strings.Repeat("m", 61)} }},
		{"nice_to_have", func(j *models.JobInput) { j.NiceToHave = fortyOne }},
		{"responsibilities", func(j *models.JobInput) { j.Responsibilities = thirtyOne }},
		{"responsibilities", func(j *models.JobInput) { j.Responsibilities = []string{strings.Repeat("r", 401)} }},
		{"about", func(j *models.JobInput) { j.About = strings.Repeat("a", 4001) }},
		{"url", func(j *models.JobInput) { j.URL = "ftp://x.ro/job" }},
		{"url", func(j *models.JobInput) { j.URL = "not a link" }},
		{"url", func(j *models.JobInput) { j.URL = "https:///nohost" }},
		{"url", func(j *models.JobInput) { j.URL = "http://x.ro/" + strings.Repeat("p", 1989) }},
		{"url", func(j *models.JobInput) { j.URL = "http://[::1" }},
		{"source", func(j *models.JobInput) { j.Source = "catalog" }},
		{"source", func(j *models.JobInput) { j.Source = "" }},
		{"read_from", func(j *models.JobInput) { j.ReadFrom = "" }},
	} {
		_, err := services.CleanJob(ok(tc.mutate))
		if appErr := apperrors.As(err); appErr.Code != apperrors.ValidationFailed || appErr.Context["field"] != tc.field {
			t.Errorf("%s: %v %v", tc.field, err, appErr.Context)
		}
	}
}

func TestCleanQuery(t *testing.T) {
	filter, err := services.CleanQuery(services.JobQuery{Q: "  devops ", City: " BUCUREȘTI ", Country: "România",
		IncludeRemote: "true", Areas: []string{" DevOps ", "devops", "", "Data"}, Level: "senior"})
	if err != nil || filter.Q != "devops" || !slices.Equal(filter.Cities, []string{"bucuresti", "bucharest"}) || filter.Country != "romania" ||
		!filter.IncludeRemote || !slices.Equal(filter.Areas, []string{"devops", "data"}) || filter.Level != "senior" {
		t.Fatalf("filter: %+v %v", filter, err)
	}
	if filter, err := services.CleanQuery(services.JobQuery{City: "Iasi"}); err != nil || !slices.Equal(filter.Cities, []string{"iasi", "iasi"}) {
		t.Fatalf("Iasi: %+v %v", filter, err)
	}
	if filter, err := services.CleanQuery(services.JobQuery{City: "Berlin"}); err != nil || !slices.Equal(filter.Cities, []string{"berlin"}) {
		t.Fatalf("Berlin: %+v %v", filter, err)
	}
	if filter, err := services.CleanQuery(services.JobQuery{}); err != nil || filter.Cities != nil || filter.IncludeRemote {
		t.Fatalf("empty: %+v %v", filter, err)
	}
	id := uuid.New()
	if filter, err := services.CleanQuery(services.JobQuery{IDs: " " + id.String() + ",,", Limit: "20"}); err != nil ||
		!slices.Equal(filter.IDs, []uuid.UUID{id}) || filter.Limit != 20 {
		t.Fatalf("ids: %+v %v", filter, err)
	}
	if filter, _ := services.CleanQuery(services.JobQuery{}); filter.Limit != services.MaxList || filter.IDs != nil {
		t.Fatalf("defaults: %+v", filter)
	}
	tooMany := strings.Repeat(id.String()+",", services.MaxIDs+1)
	for field, query := range map[string]services.JobQuery{
		"include_remote": {IncludeRemote: "maybe"}, "level": {Level: "Senior"}, "ids": {IDs: "not-an-id"}, "limit": {Limit: "0"},
	} {
		if _, err := services.CleanQuery(query); apperrors.As(err).Context["field"] != field {
			t.Errorf("%s: %v", field, err)
		}
	}
	if _, err := services.CleanQuery(services.JobQuery{IDs: tooMany}); apperrors.As(err).Context["field"] != "ids" {
		t.Errorf("too many ids: %v", err)
	}
}

func TestJobsThroughRoutes(t *testing.T) {
	repo := newMemoryJobs()
	handler := devJobs(repo, &fakeReader{})
	another := repo.jobs[0]
	another.ID = uuid.New()
	repo.jobs = append(repo.jobs, another)
	if status, list := testutil.Do(t, handler, http.MethodGet, jobsPath+"?limit=1", nil); status != http.StatusOK || len(list["items"].([]any)) != 1 ||
		list["total"] != float64(2) {
		t.Fatalf("a list cut at its limit counts every job: %d %v", status, list)
	}
	repo.jobs = repo.jobs[:1]
	status, list := testutil.Do(t, handler, http.MethodGet, jobsPath+"?q=go&city=Iasi&include_remote=1&area=DevOps&area=Data&level=mid&country=RO", nil)
	if status != http.StatusOK || list["total"] != float64(1) || len(list["items"].([]any)) != 1 ||
		repo.filter.Q != "go" || !slices.Equal(repo.filter.Areas, []string{"devops", "data"}) || !repo.filter.IncludeRemote {
		t.Fatalf("list: %d %v %+v", status, list, repo.filter)
	}
	catalogID := repo.jobs[0].ID.String()
	if status, job := testutil.Do(t, handler, http.MethodGet, jobsPath+"/"+catalogID, nil); status != http.StatusOK || job["title"] != "Catalog job" {
		t.Fatalf("get catalog: %d %v", status, job)
	}
	status, created := testutil.Do(t, handler, http.MethodPost, jobsPath, with(validJob(), "must_have", []string{"Go", "go"}))
	if status != http.StatusCreated || created["added_by_me"] != true || fmt.Sprint(created["must_have"]) != "[Go]" {
		t.Fatalf("create: %d %v", status, created)
	}
	path := jobsPath + "/" + created["id"].(string)
	other := identity.New(uuid.New(), uuid.NewString())
	stranger := jobsHandler(repo, &fakeReader{}, &other)
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		if status, body := testutil.Do(t, stranger, method, path, nil); status != http.StatusNotFound || testutil.ErrorCode(body) != "NOT_FOUND" {
			t.Fatalf("%s another person's job: %d %v", method, status, body)
		}
	}
	if _, list := testutil.Do(t, stranger, http.MethodGet, jobsPath, nil); list["total"] != float64(1) {
		t.Fatalf("another person sees only the catalog: %v", list)
	}
	if status, _ := testutil.Do(t, handler, http.MethodDelete, jobsPath+"/"+catalogID, nil); status != http.StatusNotFound {
		t.Fatalf("delete catalog job: %d", status)
	}
	if status, _ := testutil.Do(t, handler, http.MethodDelete, path, nil); status != http.StatusNoContent {
		t.Fatalf("delete own job: %d", status)
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, path}, {http.MethodDelete, path}, {http.MethodGet, jobsPath + "/not-a-uuid"},
		{http.MethodDelete, jobsPath + "/42"}, {http.MethodGet, jobsPath + "/read"},
	} {
		if status, body := testutil.Do(t, handler, tc.method, tc.path, nil); status != http.StatusNotFound || testutil.ErrorCode(body) != "NOT_FOUND" {
			t.Fatalf("%s %s: %d %v", tc.method, tc.path, status, body)
		}
	}
}

func TestJobRequestValidation(t *testing.T) {
	handler := devJobs(newMemoryJobs(), &fakeReader{})
	for _, tc := range []struct {
		method, path string
		payload      any
		field        string
	}{
		{http.MethodGet, jobsPath + "?level=principal", nil, "level"},
		{http.MethodGet, jobsPath + "?include_remote=yes", nil, "include_remote"},
		{http.MethodPost, jobsPath, with(validJob(), "title", ""), "title"},
		{http.MethodPost, jobsPath, with(validJob(), "source", "catalog"), "source"},
		{http.MethodPost, jobsPath, with(validJob(), "years_required", 2.5), "years_required"},
		{http.MethodPost, jobsPath, with(validJob(), "must_have", "Go"), "must_have"},
		{http.MethodPost, jobsPath, with(validJob(), "must_have", []any{1}), "must_have"},
		{http.MethodPost, jobsPath, "{", ""},
		{http.MethodPost, jobsPath + "/read", map[string]any{}, "url"},
		{http.MethodPost, jobsPath + "/read", map[string]any{"url": "javascript:alert(1)"}, "url"},
		{http.MethodPost, jobsPath + "/read", map[string]any{"url": "https://x.ro/" + strings.Repeat("p", 2000)}, "url"},
		{http.MethodPost, jobsPath + "/read", map[string]any{"text": "too short"}, "text"},
		{http.MethodPost, jobsPath + "/read", map[string]any{"text": strings.Repeat("x", 50001)}, "text"},
		{http.MethodPost, jobsPath + "/read", map[string]any{"text": exampleAd, "url": "file:///etc/passwd"}, "url"},
		{http.MethodPost, jobsPath + "/read", map[string]any{"text": 5}, "text"},
	} {
		status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload)
		if status != http.StatusUnprocessableEntity || testutil.ErrorCode(body) != "VALIDATION_FAILED" || errorContext(body, "field") != nilIfEmpty(tc.field) {
			t.Errorf("%s %s %v: %d %v", tc.method, tc.path, tc.payload, status, body)
		}
	}
}

func nilIfEmpty(field string) any {
	if field == "" {
		return nil
	}
	return field
}

func TestReadPastedText(t *testing.T) {
	reader := &fakeReader{}
	handler := devJobs(newMemoryJobs(), reader)
	status, draft := testutil.Do(t, handler, http.MethodPost, jobsPath+"/read",
		map[string]any{"text": exampleAd, "company": " Hint SRL ", "url": " https://www.ejobs.ro/job/1 "})
	if status != http.StatusOK || draft["title"] != "DevOps Engineer" || draft["company"] != "Hint SRL" || draft["city"] != "Brașov" ||
		draft["source"] != "text" || draft["read_from"] != "text" || draft["host"] != "www.ejobs.ro" ||
		draft["url"] != "https://www.ejobs.ro/job/1" || draft["years_required"] != float64(3) || draft["level"] != "mid" ||
		fmt.Sprint(draft["nice_to_have"]) != "[Kubernetes AWS]" || reader.url != "" {
		t.Fatalf("draft from text: %d %v", status, draft)
	}
	status, bare := testutil.Do(t, handler, http.MethodPost, jobsPath+"/read", map[string]any{"text": strings.Repeat("Words without skills. ", 5)})
	if status != http.StatusOK || bare["host"] != "" || bare["years_required"] != nil || fmt.Sprint(bare["must_have"]) != "[]" {
		t.Fatalf("bare draft: %d %v", status, bare)
	}
}

func TestReadLink(t *testing.T) {
	reader := &fakeReader{page: &fetch.Page{ReadFrom: "json_ld", Title: "Go Developer", Company: "Northwind", City: "cluj",
		Country: "RO", WorkMode: "remote", Text: "We write Go every day in a hybrid team based in Iași, 2 years minimum.\nRequirements\n- Go\n- Docker"}}
	handler := devJobs(newMemoryJobs(), reader)
	status, draft := testutil.Do(t, handler, http.MethodPost, jobsPath+"/read", map[string]any{"url": "https://jobs.example.com/go?ref=1"})
	if status != http.StatusOK || reader.url != "https://jobs.example.com/go?ref=1" || draft["title"] != "Go Developer" ||
		draft["company"] != "Northwind" || draft["city"] != "Cluj-Napoca" || draft["country"] != "Romania" || draft["work_mode"] != "remote" ||
		draft["source"] != "link" || draft["read_from"] != "json_ld" || draft["host"] != "jobs.example.com" ||
		fmt.Sprint(draft["must_have"]) != "[Go Docker]" || draft["level"] != "mid" {
		t.Fatalf("draft from a link: %d %v", status, draft)
	}
	status, hinted := testutil.Do(t, handler, http.MethodPost, jobsPath+"/read", map[string]any{"url": "https://jobs.example.com/go", "title": "My title"})
	if status != http.StatusOK || hinted["title"] != "My title" {
		t.Fatalf("hints win: %d %v", status, hinted)
	}
	reader.page = &fetch.Page{ReadFrom: "page_text", City: "Berlin", Country: "Deutschland", Text: "Backend Engineer\nWork from our Bucharest office."}
	if _, abroad := testutil.Do(t, handler, http.MethodPost, jobsPath+"/read", map[string]any{"url": "https://jobs.example.com/de"}); abroad["city"] != "Berlin" ||
		abroad["country"] != "Deutschland" || abroad["title"] != "Backend Engineer" || abroad["read_from"] != "page_text" {
		t.Fatalf("a foreign JobPosting location wins: %v", abroad)
	}
	for _, r := range []fetch.Reason{fetch.LoginRequired, fetch.NoJobData, fetch.FetchFailed, fetch.TooLarge, fetch.BlockedAddress} {
		reader.err = &fetch.Error{Reason: r}
		status, body := testutil.Do(t, handler, http.MethodPost, jobsPath+"/read", map[string]any{"url": "https://jobs.example.com/x"})
		if status != http.StatusUnprocessableEntity || testutil.ErrorCode(body) != "PAGE_UNREADABLE" || errorContext(body, "reason") != string(r) {
			t.Errorf("%s: %d %v", r, status, body)
		}
	}
	reader.err = errors.New("unexpected")
	if status, body := testutil.Do(t, handler, http.MethodPost, jobsPath+"/read", map[string]any{"url": "https://jobs.example.com/x"}); status != http.StatusInternalServerError {
		t.Fatalf("an unexpected reader error: %d %v", status, body)
	}
}

func everyJobRoute() []struct {
	method, path string
	payload      any
} {
	id := uuid.NewString()
	return []struct {
		method, path string
		payload      any
	}{
		{http.MethodGet, jobsPath, nil},
		{http.MethodGet, jobsPath + "/" + id, nil},
		{http.MethodPost, jobsPath, validJob()},
		{http.MethodDelete, jobsPath + "/" + id, nil},
	}
}

func TestJobsRequireAnIdentity(t *testing.T) {
	handler := jobsHandler(newMemoryJobs(), &fakeReader{}, nil)
	routes := append(everyJobRoute(), struct {
		method, path string
		payload      any
	}{http.MethodPost, jobsPath + "/read", map[string]any{"text": exampleAd}})
	for _, tc := range routes {
		if status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload); status != http.StatusUnauthorized || testutil.ErrorCode(body) != "AUTH_REQUIRED" {
			t.Errorf("%s %s without identity: %d %v", tc.method, tc.path, status, body)
		}
	}
}

func TestJobRepositoryFailuresSurface(t *testing.T) {
	repo := newMemoryJobs()
	repo.fail = apperrors.New(apperrors.DatabaseUnavailable, "down")
	handler := devJobs(repo, &fakeReader{})
	for _, tc := range everyJobRoute() {
		if status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload); status != http.StatusServiceUnavailable {
			t.Errorf("%s %s: %d %v", tc.method, tc.path, status, body)
		}
	}
}

func TestRequirementsFromTheAnalysisThroughRoutes(t *testing.T) {
	repo := newMemoryJobs()
	handler := devJobs(repo, &fakeReader{})
	_, created := testutil.Do(t, handler, http.MethodPost, jobsPath, validJob())
	path := jobsPath + "/" + created["id"].(string) + "/requirements"
	status, job := testutil.Do(t, handler, http.MethodPut, path, map[string]any{"must_have": []string{" Kubernetes ", "kubernetes", "AWS"}, "nice_to_have": []string{"NestJS", "aws"}})
	if status != http.StatusOK || fmt.Sprint(job["must_have"]) != "[Kubernetes AWS]" || fmt.Sprint(job["nice_to_have"]) != "[NestJS]" ||
		job["requirements_from"] != "analysis" {
		t.Fatalf("set: %d %v", status, job)
	}
	for _, tc := range []struct {
		path    string
		payload any
		status  int
	}{
		{jobsPath + "/" + uuid.NewString() + "/requirements", map[string]any{}, http.StatusNotFound},
		{jobsPath + "/nope/requirements", map[string]any{}, http.StatusNotFound},
		{path, map[string]any{"must_have": []string{strings.Repeat("x", 200)}}, http.StatusUnprocessableEntity},
		{path, "{", http.StatusUnprocessableEntity},
	} {
		if status, body := testutil.Do(t, handler, http.MethodPut, tc.path, tc.payload); status != tc.status {
			t.Errorf("%s %v: %d %v", tc.path, tc.payload, status, body)
		}
	}
	stranger := jobsHandler(repo, &fakeReader{}, &identity.IdentityContext{TenantID: uuid.New(), Subject: "someone"})
	if status, _ := testutil.Do(t, stranger, http.MethodPut, path, map[string]any{}); status != http.StatusNotFound {
		t.Fatalf("another person's job: %d", status)
	}
	if status, _ := testutil.Do(t, jobsHandler(repo, &fakeReader{}, nil), http.MethodPut, path, map[string]any{}); status != http.StatusUnauthorized {
		t.Fatalf("without identity: %d", status)
	}
	repo.fail = apperrors.New(apperrors.DatabaseUnavailable, "down")
	if status, _ := testutil.Do(t, handler, http.MethodPut, path, map[string]any{}); status != http.StatusServiceUnavailable {
		t.Fatalf("database down: %d", status)
	}
}

func TestJobCardsCarryNoWholeAd(t *testing.T) {
	job := models.Job{Title: "SRE", About: strings.Repeat("ș", models.CardAbout+20), Responsibilities: []string{"a", "b", "c", "d"},
		Qualifications: []string{"5 years"}, Perks: []string{"Gym"}, AboutCompany: "We build payments."}
	card := models.CardOf(job)
	if len([]rune(card.About)) != models.CardAbout || !slices.Equal(card.Responsibilities, []string{"a", "b", "c"}) {
		t.Fatalf("card: %+v", card)
	}
	raw, _ := json.Marshal(models.JobList{Items: []models.JobCard{card}, Total: 1})
	for _, field := range []string{"qualifications", "perks", "about_company", "bonuses", "product", "read_from"} {
		if strings.Contains(string(raw), `"`+field+`"`) {
			t.Fatalf("the list carries %s: %s", field, raw)
		}
	}
	if short := models.CardOf(models.Job{About: "Short."}); short.About != "Short." || len(short.Responsibilities) != 0 {
		t.Fatalf("a short ad stays whole: %+v", short)
	}
}
