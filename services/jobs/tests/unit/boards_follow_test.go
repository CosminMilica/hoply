package unit_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"jobs/internal/fetch"
	"jobs/internal/identity"
	"jobs/internal/importer"
	"jobs/internal/models"
	"jobs/internal/routes"
	"jobs/internal/services"
	"jobs/tests/testutil"
)

// answer is what a fixture server sends for a path.
type answer struct{ media, body string }

// fixtureServer answers paths with fixtures of any media type and records each
// request as "METHOD path body".
type fixtureServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []string
}

func newFixtureServer(t *testing.T, routes map[string]answer) *fixtureServer {
	t.Helper()
	f := &fixtureServer{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.requests = append(f.requests, strings.TrimSpace(r.Method+" "+r.URL.Path+" "+string(body)))
		f.mu.Unlock()
		a, ok := routes[r.URL.Path]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", a.media)
		_, _ = io.WriteString(w, a.body)
	}))
	t.Cleanup(f.Close)
	return f
}

func asJSON(t *testing.T, value any) answer { return answer{"application/json", jsonBody(t, value)} }

const romaniaID = "f2e609fe92974a55a05fc1cdc2852122"

func workdayFixtures(t *testing.T) map[string]answer {
	facets := []map[string]any{{"facetParameter": "locationMainGroup", "values": []map[string]any{
		{"facetParameter": "locationCountry", "descriptor": "Location Country", "values": []map[string]any{
			{"descriptor": "Germany", "id": "de", "count": 3}, {"descriptor": "Romania", "id": romaniaID, "count": 4}}},
		{"facetParameter": "locations", "values": []map[string]any{{"descriptor": "Romania - Bucharest", "id": "buc"}}},
	}}}
	return map[string]answer{
		"/wday/cxs/northwind/Careers/jobs": asJSON(t, map[string]any{"total": 4, "facets": facets, "jobPostings": []map[string]any{
			{"title": "DevOps Engineer", "externalPath": "/job/Bucharest/DevOps-Engineer_R1", "locationsText": "Romania - Bucharest", "postedOn": "Posted Today"},
			{"title": "Data Engineer", "externalPath": "/job/Iasi/Data-Engineer_R2", "locationsText": "2 Locations", "postedOn": "Posted Yesterday"},
			{"title": "SRE", "externalPath": "/job/Remote/SRE_R3", "locationsText": "Romania - Remote", "postedOn": "Posted 3 Days Ago"},
			{"title": "Account Executive", "externalPath": "/job/Bucharest/AE_R4", "locationsText": "Romania - Bucharest", "postedOn": "Posted 30+ Days Ago"},
		}}),
		"/wday/cxs/northwind/Careers/job/Bucharest/DevOps-Engineer_R1": asJSON(t, map[string]any{
			"hiringOrganization": map[string]any{"name": "Northwind S.R.L."},
			"jobPostingInfo": map[string]any{"title": "DevOps Engineer", "location": "Romania - Bucharest", "additionalLocations": []string{"Romania - Iasi"},
				"startDate": "2026-11-20", "remoteType": "Hybrid", "country": map[string]any{"descriptor": "Romania"},
				"jobDescription": "<p>Run things.</p><p><b>Requirements</b></p><ul><li>Kubernetes</li></ul>"},
		}),
		"/wday/cxs/northwind/Careers/job/Iasi/Data-Engineer_R2": asJSON(t, map[string]any{"jobPostingInfo": map[string]any{}}),
		"/wday/cxs/elsewhere/Careers/jobs":                      asJSON(t, map[string]any{"total": 1, "facets": []map[string]any{}, "jobPostings": []map[string]any{}}),
		"/wday/cxs/cities/Careers/jobs": asJSON(t, map[string]any{"total": 1, "facets": []map[string]any{
			{"facetParameter": "timeType", "values": []map[string]any{{"descriptor": "Full time", "id": "ft"}}},
			{"facetParameter": "locationMainGroup", "values": []map[string]any{{"facetParameter": "locations", "values": []map[string]any{
				{"descriptor": "Bucharest - Dacia One", "id": "dacia"}, {"descriptor": "Amsterdam", "id": "ams"}, {"descriptor": "Cluj-Napoca, Romania", "id": "cluj"},
			}}}},
		}, "jobPostings": []map[string]any{{"title": "SRE", "externalPath": "/job/Bucharest/SRE_R9", "locationsText": "Bucharest - Dacia One", "postedOn": "Posted Today"}}}),
		"/wday/cxs/broken/Careers/jobs": {"application/json", "{"},
	}
}

func TestWorkdayList(t *testing.T) {
	server := newFixtureServer(t, workdayFixtures(t))
	reader := fetch.New(fetch.Options{AllowLoopback: true, Hosts: map[string]string{
		"northwind.wd3.myworkdayjobs.com": server.URL, "elsewhere.wd1.myworkdayjobs.com": server.URL, "broken.wd1.myworkdayjobs.com": server.URL,
		"cities.wd1.myworkdayjobs.com": server.URL,
	}})
	ctx := context.Background()
	postings, _, err := reader.List(ctx, "workday", "northwind.wd3.myworkdayjobs.com/Careers", importer.Romania)
	if err != nil || len(postings) != 4 {
		t.Fatalf("list: %+v %v", postings, err)
	}
	if !slices.Contains(server.requests, `POST /wday/cxs/northwind/Careers/jobs {"appliedFacets":{"locationCountry":["`+romaniaID+`"]},"limit":20,"offset":0,"searchText":""}`) {
		t.Fatalf("the list keeps to Romania's facet: %v", server.requests)
	}
	today := time.Now().UTC().Format(time.DateOnly)
	first := postings[0]
	if first.ID != "/job/Bucharest/DevOps-Engineer_R1" || first.URL != "https://northwind.wd3.myworkdayjobs.com/Careers/job/Bucharest/DevOps-Engineer_R1" ||
		!first.InCountry || first.PostedAt.Format(time.DateOnly) != today || first.Page.Text != "" {
		t.Fatalf("posting: %+v", first)
	}
	if postings[1].PostedAt.Format(time.DateOnly) != time.Now().UTC().AddDate(0, 0, -1).Format(time.DateOnly) ||
		postings[2].PostedAt.Format(time.DateOnly) != time.Now().UTC().AddDate(0, 0, -3).Format(time.DateOnly) || postings[3].PostedAt != nil {
		t.Fatalf("posted on: %v %v %v", postings[1].PostedAt, postings[2].PostedAt, postings[3].PostedAt)
	}
	if err := reader.Describe(ctx, &first); err != nil || first.Page.Company != "Northwind S.R.L." || first.Page.WorkMode != "hybrid" ||
		!strings.Contains(first.Page.Text, "Kubernetes") || first.PostedAt.Format(time.DateOnly) != "2026-11-20" || !slices.Contains(first.Places, "Romania - Iasi") {
		t.Fatalf("described: %+v %v", first, err)
	}
	if err := reader.Describe(ctx, &first); err != nil {
		t.Fatalf("a described posting stays as it is: %v", err)
	}
	if err := reader.Describe(ctx, &postings[1]); err == nil {
		t.Fatal("a job without a title fails")
	}
	if err := reader.Describe(ctx, &postings[2]); err == nil {
		t.Fatal("a missing job fails")
	}
	if jobs, _, err := reader.List(ctx, "workday", "cities.wd1.myworkdayjobs.com/Careers", importer.Romania); err != nil || len(jobs) != 1 ||
		!slices.Contains(server.requests, `POST /wday/cxs/cities/Careers/jobs {"appliedFacets":{"locations":["dacia","cluj"]},"limit":20,"offset":0,"searchText":""}`) {
		t.Fatalf("a site without a country facet keeps to its locations in Romania: %v %v %v", jobs, err, server.requests)
	}
	byName := fetch.Country{Name: "Romania"}
	if jobs, _, err := reader.List(ctx, "workday", "cities.wd1.myworkdayjobs.com/Careers", byName); err != nil || len(jobs) != 1 ||
		!slices.Contains(server.requests, `POST /wday/cxs/cities/Careers/jobs {"appliedFacets":{"locations":["cluj"]},"limit":20,"offset":0,"searchText":""}`) {
		t.Fatalf("without In, a place is in the country when it names it: %v %v", jobs, err)
	}
	if none, _, err := reader.List(ctx, "workday", "elsewhere.wd1.myworkdayjobs.com/Careers", importer.Romania); err != nil || len(none) != 0 {
		t.Fatalf("a site without jobs in Romania lists none: %v %v", none, err)
	}
	if _, _, err := reader.List(ctx, "workday", "broken.wd1.myworkdayjobs.com/Careers", importer.Romania); err == nil {
		t.Fatal("a broken answer fails")
	}
	if _, _, err := reader.List(ctx, "workday", "missing.wd1.myworkdayjobs.com/Careers", importer.Romania); err == nil {
		t.Fatal("an unreachable site fails")
	}
}

func TestSmartRecruitersList(t *testing.T) {
	api := newFixtureServer(t, map[string]answer{
		"/v1/companies/NorthwindGroup/postings": asJSON(t, map[string]any{"totalFound": 1, "content": []map[string]any{
			{"id": "7440001", "name": "Junior Network Administrator", "releasedDate": "2026-11-18T08:00:00.000Z", "company": map[string]any{"name": "Northwind Group"},
				"location": map[string]any{"city": "Iasi", "country": "ro", "fullLocation": "Iasi, Romania", "hybrid": true}},
		}}),
		"/v1/companies/NorthwindGroup/postings/7440001": {"application/json", boardFixtures(t)["/v1/companies/NorthwindGroup/postings/7440001"]},
		"/v1/companies/Broken/postings":                 {"application/json", "["},
	})
	reader := fetch.New(fetch.Options{AllowLoopback: true, Hosts: map[string]string{"api.smartrecruiters.com": api.URL}})
	ctx := context.Background()

	jobs, _, err := reader.List(ctx, "smartrecruiters", "NorthwindGroup", importer.Romania)
	if err != nil || len(jobs) != 1 || jobs[0].URL != "https://jobs.smartrecruiters.com/NorthwindGroup/7440001" || !jobs[0].InCountry ||
		jobs[0].Page.WorkMode != "hybrid" || jobs[0].PostedAt.Format(time.DateOnly) != "2026-11-18" {
		t.Fatalf("smartrecruiters: %+v %v", jobs, err)
	}
	if !slices.Contains(api.requests, "GET /v1/companies/NorthwindGroup/postings") {
		t.Fatalf("requests: %v", api.requests)
	}
	if err := reader.Describe(ctx, &jobs[0]); err != nil || !strings.Contains(jobs[0].Page.Text, "DNS and DHCP") || jobs[0].Page.City != "Iasi" {
		t.Fatalf("smartrecruiters described: %+v %v", jobs[0], err)
	}
	if _, _, err := reader.List(ctx, "smartrecruiters", "Broken", importer.Romania); err == nil {
		t.Fatal("a broken answer fails")
	}
}

func TestBoardsFromLinks(t *testing.T) {
	for link, want := range map[string]string{
		"https://job-boards.eu.greenhouse.io/ionos2/jobs/5030878101":                          "greenhouse ionos2",
		"https://boards.greenhouse.io/embed/job_app?for=northwind&token=4001":                 "greenhouse northwind",
		"https://careers.northwind.example/open?gh_jid=4001&for=northwind":                    "greenhouse northwind",
		"https://jobs.lever.co/northwind-cloud/aa11":                                          "lever northwind-cloud",
		"https://jobs.eu.lever.co/olx/bb22/apply":                                             "lever-eu olx",
		"https://jobs.ashbyhq.com/linear/d3bc1ced":                                            "ashby linear",
		"https://jobs.smartrecruiters.com/BoschGroup/744000":                                  "smartrecruiters BoschGroup",
		"https://crowdstrike.wd5.myworkdayjobs.com/crowdstrikecareers/job/Romania/X_R1":       "workday crowdstrike.wd5.myworkdayjobs.com/crowdstrikecareers",
		"https://lseg.wd3.myworkdayjobs.com/en-US/Careers/job/Bucharest/Software-Engineer_R1": "workday lseg.wd3.myworkdayjobs.com/Careers",
		"https://apply.workable.com/northwind/j/AB12":                                         "",
		"https://boards.greenhouse.io/embed/job_app":                                          "",
		"https://www.linkedin.com/jobs/view/42":                                               "",
		"mailto:jobs@example.com":                                                             "",
	} {
		platform, board, ok := fetch.BoardOf(link)
		if got := strings.TrimSpace(platform + " " + board); got != want || ok != (want != "") {
			t.Errorf("%s: %q %v, want %q", link, got, ok, want)
		}
	}
	for platform, board := range map[string]string{
		"workday": "northwind.example/Careers", "sitemap": "http://careers.example/sitemap.xml", "greenhouse": "a/b", "bamboo": "x",
	} {
		if fetch.CheckBoard(platform, board) == nil {
			t.Errorf("%s %s is refused", platform, board)
		}
	}
	if _, _, err := fetch.New(fetch.Options{}).List(context.Background(), "greenhouse", "a/b", importer.Romania); err == nil {
		t.Fatal("List checks the board")
	}
}

// followedBoards keeps followed boards in memory.
type followedBoards struct {
	boards []models.Board
	err    error
}

func (f *followedBoards) Followed(context.Context) ([]models.Board, error) { return f.boards, f.err }

func (f *followedBoards) Follow(_ context.Context, b models.Board) (models.Board, bool, error) {
	if f.err != nil {
		return models.Board{}, false, f.err
	}
	for _, known := range f.boards {
		if known.Platform == b.Platform && known.Board == b.Board {
			return known, false, nil
		}
	}
	at := time.Now()
	b.FollowedAt = &at
	f.boards = append(f.boards, b)
	return b, true, nil
}

func TestImporterReadsFollowedBoardsAndOnlyWantedDescriptions(t *testing.T) {
	listed := []importer.Board{{Company: "A", Platform: "workday", Board: "a.wd1.myworkdayjobs.com/Careers"}}
	followed := &followedBoards{boards: []models.Board{
		{Company: "A again", Platform: "workday", Board: "a.wd1.myworkdayjobs.com/Careers"}, {Company: "B", Platform: "lever", Board: "b"},
	}}
	lister := &fakeLister{
		postings: map[string][]fetch.Posting{"a.wd1.myworkdayjobs.com/Careers": {
			{ID: "1", Places: []string{"2 Locations"}, InCountry: true, Page: fetch.Page{Title: "DevOps Engineer"}},
			{ID: "2", Places: []string{"Berlin"}, Page: fetch.Page{Title: "SRE"}},
			{ID: "3", InCountry: true, Page: fetch.Page{Title: "Account Executive"}},
			{ID: "4", InCountry: true, Page: fetch.Page{Title: "Data Engineer"}},
		}, "b": {{ID: "5", Places: []string{"Remote"}, Page: fetch.Page{Title: "Backend Developer", Country: "RO"}}}},
		unreadable: map[string]bool{"4": true},
	}
	store := &memoryImports{jobs: map[string]models.ImportedJob{}}
	im := importer.New(store, lister, listed, followed, freeLock, time.Millisecond)
	if boards := im.Boards(context.Background()); len(boards) != 2 || boards[1].Company != "B" {
		t.Fatalf("the followed boards not listed already: %+v", boards)
	}
	results := im.RunOnce(context.Background())
	if len(results) != 2 || results[0].Kept != 1 || results[0].Unread != 1 || results[0].Closed != 0 || results[1].Kept != 1 {
		t.Fatalf("results: %+v", results)
	}
	if !slices.Equal(lister.described, []string{"1", "4", "5"}) || store.jobs["1"].Country != "Romania" || !slices.Equal(store.closed, []string{"lever/b"}) {
		t.Fatalf("only the wanted jobs are described, and a board with an unread one closes nothing: %v %+v %v", lister.described, store.jobs["1"], store.closed)
	}
	if r := im.Import(context.Background(), importer.Board{Company: "B", Platform: "lever", Board: "b"}); r.Kept != 1 || r.New != 0 {
		t.Fatalf("one board on demand: %+v", r)
	}
	if len(store.listed) != 1 || !slices.Equal(store.listed[0], []string{"workday/a.wd1.myworkdayjobs.com/Careers", "lever/b"}) {
		t.Fatalf("the followed boards count as listed: %v", store.listed)
	}
	followed.err = errors.New("database down")
	if boards := im.Boards(context.Background()); len(boards) != 1 {
		t.Fatalf("without the followed boards, the listed ones still run: %+v", boards)
	}
	if im.RunOnce(context.Background()); len(store.listed) != 1 {
		t.Fatalf("without the followed boards, no job is closed as unlisted: %v", store.listed)
	}
}

// fakeBoardImporter lists boards and records the ones imported on demand.
type fakeBoardImporter struct {
	listed   []importer.Board
	imported []string
	fail     bool
}

func (f *fakeBoardImporter) Listed() []importer.Board { return f.listed }

func (f *fakeBoardImporter) Import(_ context.Context, b importer.Board) importer.Result {
	f.imported = append(f.imported, b.Platform+"/"+b.Board)
	if f.fail {
		return importer.Result{Board: b, Err: errors.New("board down")}
	}
	return importer.Result{Board: b}
}

func boardsHandler(service *services.BoardService, ident *identity.IdentityContext) http.Handler {
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Boards: service})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ident != nil {
			r = r.WithContext(identity.WithContext(r.Context(), *ident))
		}
		router.ServeHTTP(w, r)
	})
}

func TestFollowingABoardThroughRoutes(t *testing.T) {
	repo := &followedBoards{}
	imports := &fakeBoardImporter{listed: []importer.Board{{Company: "Awin", Platform: "greenhouse", Board: "awin"}}}
	service := services.NewBoardService(repo, imports).WithBackground(func(run func()) { run() })
	dev := identity.Dev()
	handler := boardsHandler(service, &dev)
	of := "/api/v1/boards/of?url="

	status, body := testutil.Do(t, handler, http.MethodGet, of+"https://job-boards.greenhouse.io/awin/jobs/1", nil)
	if status != http.StatusOK || body["imported"] != true || body["company"] != "Awin" {
		t.Fatalf("a listed board: %d %v", status, body)
	}
	status, body = testutil.Do(t, handler, http.MethodGet, of+"https://jobs.lever.co/northwind/1", nil)
	if status != http.StatusOK || body["imported"] != false || body["platform"] != "lever" || body["board"] != "northwind" {
		t.Fatalf("a board not imported: %d %v", status, body)
	}
	if status, _ := testutil.Do(t, handler, http.MethodGet, of+"https://www.linkedin.com/jobs/view/42", nil); status != http.StatusNotFound {
		t.Fatalf("not a board: %d", status)
	}

	status, body = testutil.Do(t, handler, http.MethodPost, "/api/v1/boards", map[string]any{"url": "https://jobs.lever.co/northwind/1", "company": " Northwind "})
	if status != http.StatusCreated || body["company"] != "Northwind" || body["followed_at"] == nil || !slices.Equal(imports.imported, []string{"lever/northwind"}) {
		t.Fatalf("follow: %d %v %v", status, body, imports.imported)
	}
	if status, _ := testutil.Do(t, handler, http.MethodPost, "/api/v1/boards", map[string]any{"url": "https://jobs.lever.co/northwind/2", "company": "Northwind"}); status != http.StatusCreated ||
		len(imports.imported) != 1 {
		t.Fatalf("a board followed again is not imported again: %d %v", status, imports.imported)
	}
	if _, body := testutil.Do(t, handler, http.MethodGet, of+"https://jobs.lever.co/northwind/2", nil); body["imported"] != true {
		t.Fatalf("a followed board is imported: %v", body)
	}
	status, body = testutil.Do(t, handler, http.MethodPost, "/api/v1/boards", map[string]any{"url": "https://job-boards.greenhouse.io/awin/jobs/1", "company": "Awin"})
	if status != http.StatusCreated || body["followed_at"] != nil || len(repo.boards) != 1 {
		t.Fatalf("a listed board is not followed twice: %d %v", status, body)
	}
	imports.fail = true
	if status, _ := testutil.Do(t, handler, http.MethodPost, "/api/v1/boards", map[string]any{"url": "https://jobs.ashbyhq.com/linear/1", "company": "Linear"}); status != http.StatusCreated {
		t.Fatalf("a follow whose first import fails is still kept: %d", status)
	}

	for field, payload := range map[string]map[string]any{
		"company": {"url": "https://jobs.lever.co/x/1", "company": " "},
		"url":     {"url": "https://www.linkedin.com/jobs/view/42", "company": "LinkedIn"},
	} {
		if status, body := testutil.Do(t, handler, http.MethodPost, "/api/v1/boards", payload); status != http.StatusUnprocessableEntity || errorContext(body, "field") != field {
			t.Errorf("%s: %d %v", field, status, body)
		}
	}
	if status, _ := testutil.Do(t, handler, http.MethodPost, "/api/v1/boards", "not an object"); status != http.StatusBadRequest && status != http.StatusUnprocessableEntity {
		t.Fatalf("a bad body: %d", status)
	}

	repo.err = errors.New("database down")
	if status, _ := testutil.Do(t, handler, http.MethodGet, of+"https://jobs.lever.co/northwind/1", nil); status < 500 {
		t.Fatalf("a failing store: %d", status)
	}
	if status, _ := testutil.Do(t, handler, http.MethodPost, "/api/v1/boards", map[string]any{"url": "https://jobs.lever.co/z/1", "company": "Z"}); status < 500 {
		t.Fatalf("a failing store: %d", status)
	}
	anonymous := boardsHandler(services.NewBoardService(&followedBoards{}, nil), nil)
	if status, _ := testutil.Do(t, anonymous, http.MethodGet, of+"https://jobs.lever.co/northwind/1", nil); status != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", status)
	}
	if status, _ := testutil.Do(t, anonymous, http.MethodPost, "/api/v1/boards", map[string]any{}); status != http.StatusUnauthorized {
		t.Fatalf("anonymous follow: %d", status)
	}
	withoutImporter := boardsHandler(services.NewBoardService(&followedBoards{}, nil), &dev)
	if status, body := testutil.Do(t, withoutImporter, http.MethodPost, "/api/v1/boards", map[string]any{"url": "https://jobs.lever.co/n/1", "company": "N"}); status != http.StatusCreated {
		t.Fatalf("following without an importer stores the board: %d %v", status, body)
	}
}

func TestListsCutShortSayTheyAreIncomplete(t *testing.T) {
	workdayPage := make([]map[string]any, 20)
	for i := range workdayPage {
		workdayPage[i] = map[string]any{"title": "DevOps Engineer", "externalPath": fmt.Sprintf("/job/Bucharest/DevOps_R%d", i), "locationsText": "Romania - Bucharest"}
	}
	srPage := make([]map[string]any, 100)
	for i := range srPage {
		srPage[i] = map[string]any{"id": fmt.Sprint(i), "name": "SRE", "location": map[string]any{"city": "Iasi", "country": "ro"}}
	}
	server := newFixtureServer(t, map[string]answer{
		"/wday/cxs/big/Careers/jobs": asJSON(t, map[string]any{"total": 2000, "jobPostings": workdayPage, "facets": []map[string]any{
			{"facetParameter": "locationCountry", "values": []map[string]any{{"descriptor": "Romania", "id": romaniaID}}},
		}}),
		"/v1/companies/Big/postings": asJSON(t, map[string]any{"totalFound": 2000, "content": srPage}),
	})
	reader := fetch.New(fetch.Options{AllowLoopback: true, Hosts: map[string]string{"big.wd1.myworkdayjobs.com": server.URL, "api.smartrecruiters.com": server.URL}})
	if jobs, complete, err := reader.List(context.Background(), "workday", "big.wd1.myworkdayjobs.com/Careers", importer.Romania); err != nil || complete || len(jobs) != 400 {
		t.Fatalf("workday stops at 400 jobs: %d %v %v", len(jobs), complete, err)
	}
	if jobs, complete, err := reader.List(context.Background(), "smartrecruiters", "Big", importer.Romania); err != nil || complete || len(jobs) != 500 {
		t.Fatalf("smartrecruiters stops at 500 jobs: %d %v %v", len(jobs), complete, err)
	}
}

func TestImporterSkipsDescriptionsItReadLately(t *testing.T) {
	server := newFixtureServer(t, workdayFixtures(t))
	reader := fetch.New(fetch.Options{AllowLoopback: true, Hosts: map[string]string{"northwind.wd3.myworkdayjobs.com": server.URL}})
	boards := []importer.Board{{Company: "Northwind", Platform: "workday", Board: "northwind.wd3.myworkdayjobs.com/Careers"}}
	store := &memoryImports{jobs: map[string]models.ImportedJob{}, fresh: map[string]bool{"/job/Bucharest/DevOps-Engineer_R1": true}}
	r := importer.New(store, reader, boards, nil, freeLock, time.Millisecond).RunOnce(context.Background())[0]
	if !slices.Equal(store.seen, []string{"/job/Bucharest/DevOps-Engineer_R1"}) || r.Kept != 1 || r.Unread != 2 || r.Closed != 0 {
		t.Fatalf("a job read lately is only marked seen: %+v %v", r, store.seen)
	}
	for _, request := range server.requests {
		if strings.Contains(request, "DevOps-Engineer_R1") {
			t.Fatalf("its description is not read again: %v", server.requests)
		}
	}
	store.fresh = nil
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if r := importer.New(store, reader, boards, nil, freeLock, time.Hour).Import(ctx, boards[0]); !errors.Is(r.Err, context.DeadlineExceeded) || r.Kept != 1 {
		t.Fatalf("an import stopped while it waits between descriptions: %+v", r)
	}
	store.freshErr = errors.New("database down")
	if r := importer.New(store, reader, boards, nil, freeLock, time.Millisecond).RunOnce(context.Background())[0]; r.Err == nil {
		t.Fatalf("a board whose recent reads cannot be known fails: %+v", r)
	}
}
