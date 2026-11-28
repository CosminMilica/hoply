package unit_test

import (
	"context"
	"errors"
	stdhtml "html"
	"slices"
	"strings"
	"testing"
	"time"

	"jobs/internal/extract"
	"jobs/internal/fetch"
	"jobs/internal/importer"
	"jobs/internal/models"
)

// The list fixtures are synthetic, shaped like each platform's public list answer.
func listFixtures(t *testing.T) map[string]string {
	return map[string]string{
		"/v1/boards/northwind/jobs": jsonBody(t, map[string]any{"jobs": []map[string]any{
			{"id": 4001, "title": "DevOps Engineer", "company_name": "Northwind Cloud", "location": map[string]any{"name": "Hybrid"},
				"offices": []map[string]any{{"location": "Cluj-Napoca, Romania"}}, "content": stdhtml.EscapeString(boardDescription),
				"absolute_url": "https://job-boards.eu.greenhouse.io/northwind/jobs/4001", "first_published": "2026-11-20T10:00:00-05:00"},
			{"id": 4002, "title": "Account Executive", "location": map[string]any{"name": "Bucharest"}, "updated_at": "bad"},
		}}),
		"/v0/postings/northwind-cloud": jsonBody(t, []map[string]any{
			{"id": "aa11", "text": "Site Reliability Engineer", "hostedUrl": "https://jobs.lever.co/northwind-cloud/aa11", "createdAt": 1763650800000,
				"categories": map[string]any{"location": "Remote", "allLocations": []string{"Remote"}}, "country": "RO", "workplaceType": "remote",
				"description": "<div>Keep payments up.</div>", "lists": []map[string]any{{"text": "Requirements", "content": "<li>Linux</li>"}}},
		}),
		"/posting-api/job-board/northwind": jsonBody(t, map[string]any{"jobs": []map[string]any{
			{"id": "bb22", "title": "Data Engineer", "jobUrl": "https://jobs.ashbyhq.com/northwind/bb22", "publishedAt": "2026-11-01T08:00:00.000+00:00",
				"location": "Berlin", "secondaryLocations": []map[string]any{{"location": "Iași, Romania"}}, "isListed": true,
				"descriptionHtml": "<h2>What you'll do</h2><ul><li>Model data in Snowflake</li></ul>"},
			{"id": "cc33", "title": "Hidden Engineer", "isListed": false},
		}}),
		"/posting-api/job-board/broken": "[",
	}
}

func listReader(api *boardServer) *fetch.Reader {
	return fetch.New(fetch.Options{AllowLoopback: true, Hosts: map[string]string{
		"boards-api.greenhouse.io": api.URL, "api.lever.co": api.URL, "api.eu.lever.co": api.URL, "api.ashbyhq.com": api.URL,
	}})
}

func TestBoardLists(t *testing.T) {
	api := newBoardServer(t, listFixtures(t))
	reader := listReader(api)
	ctx := context.Background()

	greenhouse, complete, err := reader.List(ctx, "greenhouse", "northwind", importer.Romania)
	if err != nil || len(greenhouse) != 2 || !complete {
		t.Fatalf("greenhouse: %v %v", greenhouse, err)
	}
	first := greenhouse[0]
	if first.ID != "4001" || first.URL != "https://job-boards.eu.greenhouse.io/northwind/jobs/4001" || first.PostedAt.Format(time.RFC3339) != "2026-11-20T15:00:00Z" ||
		!slices.Equal(first.Places, []string{"Hybrid", "Cluj-Napoca, Romania"}) || first.Page.ReadFrom != "ats_api" ||
		!strings.Contains(first.Page.Text, "Run Kubernetes clusters on AWS") || greenhouse[1].PostedAt != nil {
		t.Fatalf("greenhouse posting: %+v", first)
	}
	if !slices.Contains(api.hits, "/v1/boards/northwind/jobs?content=true") {
		t.Fatalf("greenhouse asks for the content: %v", api.hits)
	}

	lever, _, err := reader.List(ctx, "lever", "northwind-cloud", importer.Romania)
	if err != nil || len(lever) != 1 || lever[0].ID != "aa11" || lever[0].Page.Company != "Northwind Cloud" || lever[0].Page.Country != "RO" ||
		lever[0].PostedAt.Format(time.DateOnly) != "2025-11-20" || lever[0].Page.WorkMode != "remote" || !strings.Contains(lever[0].Page.Text, "Linux") {
		t.Fatalf("lever: %+v %v", lever, err)
	}

	ashby, _, err := reader.List(ctx, "ashby", "northwind", importer.Romania)
	if err != nil || len(ashby) != 1 || ashby[0].ID != "bb22" || !slices.Contains(ashby[0].Places, "Iași, Romania") ||
		ashby[0].PostedAt.Format(time.DateOnly) != "2026-11-01" {
		t.Fatalf("ashby lists only listed jobs: %+v %v", ashby, err)
	}

	var unreadable *fetch.Error
	if _, _, err := reader.List(ctx, "ashby", "broken", importer.Romania); !errors.As(err, &unreadable) || unreadable.Reason != fetch.NoJobData {
		t.Fatalf("a broken answer: %v", err)
	}
	if eu, _, err := reader.List(ctx, "lever-eu", "northwind-cloud", importer.Romania); err != nil || len(eu) != 1 || !slices.Contains(api.hits, "/v0/postings/northwind-cloud?mode=json") {
		t.Fatalf("lever's EU host: %+v %v", eu, err)
	}
	if _, _, err := reader.List(ctx, "lever", "missing", importer.Romania); !errors.As(err, &unreadable) || unreadable.Reason != fetch.FetchFailed {
		t.Fatalf("a missing board: %v", err)
	}
	if _, _, err := reader.List(ctx, "workday", "northwind", importer.Romania); err == nil {
		t.Fatal("an unknown platform fails")
	}
}

func TestClassifyTheTitle(t *testing.T) {
	type want struct{ domains, areas []string }
	it := func(areas ...string) want { return want{[]string{"it"}, areas} }
	for title, want := range map[string]want{
		"Senior DevOps Engineer":                                it("DevOps"),
		"Principal Software Engineer, Site Reliability":         it("DevOps", "Backend"),
		"Backend Developer (NodeJS)":                            it("Backend"),
		"Senior Golang Engineer":                                it("Backend"),
		"Frontend Software Engineer":                            it("Frontend"),
		"Senior React Developer":                                it("Frontend"),
		"Full-Stack Engineer (React)":                           it("Full stack"),
		"Software Engineer (React Native)":                      it("Mobile"),
		"Senior Backend Engineer, Mobile Platform":              it("Backend", "Mobile"),
		"iOS Developer":                                         it("Mobile"),
		"Lead Data Scientist":                                   it("Data", "AI"),
		"Generative AI Engineer":                                it("AI"),
		"Software QA Engineer":                                  it("QA"),
		"SDET - AI Gateway":                                     it("AI", "QA"),
		"Senior Security Operations Specialist (SOC)":           it("Security"),
		"Embedded C++ AUTOSAR Developer":                        it("Embedded"),
		"Linux Systems Engineer":                                it("Sysadmin"),
		"Junior Network Administrator":                          it("Sysadmin"),
		"Technical Product Owner":                               it("Product"),
		"Senior UX/UI Designer":                                 it("Design"),
		"Senior Sales Engineer":                                 {[]string{}, []string{}},
		"Account Executive":                                     {[]string{}, []string{}},
		"Consilier bancar Iași":                                 {[]string{"banking"}, []string{"Front office"}},
		"Relationship Manager E-commerce Iasi":                  {[]string{"banking"}, []string{"Corporate banking"}},
		"Ofițer credite IMM":                                    {[]string{"banking"}, []string{"Lending", "Corporate banking"}},
		"CONSILIER FINANCIAR BANCAR RETAIL CALAFAT":             {[]string{"banking"}, []string{"Front office"}},
		"Contabil primar":                                       {[]string{"accounting"}, []string{"General accounting"}},
		"Senior Accountant":                                     {[]string{"accounting"}, []string{"General accounting"}},
		"Accounts Payable Associate with German":                {[]string{"accounting"}, []string{"Payables"}},
		"Sr. Associate - F&A - I2C 5B":                          {[]string{"accounting"}, []string{"Receivables"}},
		"Specialist - F&A - R2R 4A":                             {[]string{"accounting"}, []string{"General accounting"}},
		"Payroll Administrator":                                 {[]string{"accounting", "hr"}, []string{"Payroll", "Compensation & benefits"}},
		"Pay Advisor & Benefits Coordinator Italy":              {[]string{"hr"}, []string{"Compensation & benefits"}},
		"Sr Manager Total Rewards EMEAI":                        {[]string{"hr"}, []string{"Compensation & benefits"}},
		"Bancassurance Account Manager":                         {[]string{"insurance"}, []string{"Insurance sales"}},
		"Health & Life Claim Handler":                           {[]string{"insurance"}, []string{"Claims"}},
		"HR Knowledge Specialist":                               {[]string{"hr"}, []string{"HR generalist"}},
		"Tax Specialist":                                        {[]string{"accounting"}, []string{"Tax"}},
		"Senior Management Accountant":                          {[]string{"accounting", "finance"}, []string{"General accounting", "FP&A & controlling"}},
		"Group Finance Controller":                              {[]string{"finance"}, []string{"FP&A & controlling"}},
		"FP&A Analyst":                                          {[]string{"finance"}, []string{"FP&A & controlling"}},
		"Analist financiar":                                     {[]string{"finance"}, []string{"FP&A & controlling"}},
		"Treasury Specialist":                                   {[]string{"finance"}, []string{"Treasury"}},
		"AML/KYC Analyst":                                       {[]string{"finance"}, []string{"Risk & compliance"}},
		"Specialist - Controllership - SOX Compliance":          {[]string{"finance"}, []string{"Risk & compliance", "Audit"}},
		"Auditor intern":                                        {[]string{"finance"}, []string{"Audit"}},
		"Fraud Detection Engineer":                              {[]string{"finance"}, []string{"Risk & compliance"}},
		"Regulatory Compliance Senior Analyst":                  {[]string{"finance"}, []string{"Risk & compliance"}},
		"Backend Engineer, Payments":                            it("Backend"),
		"Senior Software Engineer Regulatory News (AI/C#/Java)": it("Backend", "AI"),
		"Underwriter":                                           {[]string{"insurance"}, []string{"Underwriting"}},
		"Claims Handler":                                        {[]string{"insurance"}, []string{"Claims"}},
		"Agent de vânzări asigurări":                            {[]string{"insurance"}, []string{"Insurance sales"}},
		"Actuarial Analyst":                                     {[]string{"insurance"}, []string{"Actuarial"}},
		"Talent Acquisition Specialist":                         {[]string{"hr"}, []string{"Recruitment"}},
		"IT Recruiter":                                          {[]string{"hr"}, []string{"Recruitment"}},
		"HR Business Partner":                                   {[]string{"hr"}, []string{"HR generalist"}},
		"Specialist resurse umane":                              {[]string{"hr"}, []string{"HR generalist"}},
		"Compensation & Benefits Analyst":                       {[]string{"hr"}, []string{"Compensation & benefits"}},
		"Learning & Development Coordinator":                    {[]string{"hr"}, []string{"Training & development"}},
		"HR Administrator":                                      {[]string{"hr"}, []string{"HR administration"}},
		"Inspector resurse umane":                               {[]string{"hr"}, []string{"HR administration"}},
		"Specialist asigurarea calității":                       {[]string{}, []string{}},
		"Contabil":                                              {[]string{"accounting"}, []string{"General accounting"}},
		"Economist":                                             {[]string{"accounting"}, []string{"General accounting"}},
		"Casier":                                                {[]string{}, []string{}},
		"Casier bancar":                                         {[]string{"banking"}, []string{"Front office"}},
		"Agent de asigurare":                                    {[]string{"insurance"}, []string{"Insurance sales"}},
		"Controlor financiar":                                   {[]string{"finance"}, []string{"FP&A & controlling"}},
		"Programator":                                           it("Backend"),
		"Analist programator":                                   it("Backend"),
		"Administrator retea de calculatoare":                   it("Sysadmin"),
		"Administrator imobile":                                 {[]string{}, []string{}},
		"Office Administrator":                                  {[]string{}, []string{}},
		"Tester software":                                       it("QA"),
		"Specialist Finance Retail - French Speaker":            {[]string{"finance"}, []string{}},
	} {
		if got := extract.Classify(title); !slices.Equal(got.Domains, want.domains) || !slices.Equal(got.Areas, want.areas) {
			t.Errorf("%s: %v %v, want %v %v", title, got.Domains, got.Areas, want.domains, want.areas)
		}
	}
	if extract.DomainOf("Payroll") != "accounting" || extract.DomainOf("Kubernetes") != "" || len(extract.Areas) != 35 {
		t.Fatal("each area is in one domain")
	}
}

func TestBoardsFile(t *testing.T) {
	boards, err := importer.ParseBoards([]byte("boards:\n  - {company: Northwind, platform: greenhouse, board: northwind}\n  - {company: Northwind, platform: ashby, board: northwind}\n"))
	if err != nil || len(boards) != 2 || boards[1] != (importer.Board{Company: "Northwind", Platform: "ashby", Board: "northwind"}) {
		t.Fatalf("boards: %+v %v", boards, err)
	}
	for _, bad := range []string{
		"boards: [",
		"boards:\n  - {company: Northwind, platform: workday, board: northwind}\n",
		"boards:\n  - {company: ' ', platform: lever, board: northwind}\n",
		"boards:\n  - {company: A, platform: lever, board: a}\n  - {company: B, platform: lever, board: a}\n",
	} {
		if _, err := importer.ParseBoards([]byte(bad)); err == nil {
			t.Errorf("%q fails", bad)
		}
	}

	t.Setenv("JOBS_IMPORT", "")
	if im, _, err := importer.FromEnv(nil, nil, nil); im != nil || err != nil {
		t.Fatalf("off by default: %v %v", im, err)
	}
	t.Setenv("JOBS_IMPORT", "on")
	if im, interval, err := importer.FromEnv(nil, nil, nil); im == nil || interval != importer.Interval || err != nil {
		t.Fatalf("on, over the boards file in the repo: %v %v %v", im, interval, err)
	}
	t.Setenv("JOBS_IMPORT_INTERVAL", "30m")
	if _, interval, err := importer.FromEnv(nil, nil, nil); interval != 30*time.Minute || err != nil {
		t.Fatalf("interval: %v %v", interval, err)
	}
	t.Setenv("JOBS_IMPORT_INTERVAL", "10s")
	if _, _, err := importer.FromEnv(nil, nil, nil); err == nil {
		t.Fatal("an interval under a minute fails")
	}
}

func TestJobsKeptFromABoard(t *testing.T) {
	board := importer.Board{Company: "Northwind Cloud", Platform: "greenhouse", Board: "northwind"}
	posted := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	posting := fetch.Posting{ID: "4001", URL: "https://example.com/4001", PostedAt: &posted, Places: []string{"Hybrid", "Iasi, Romania"},
		Page: fetch.Page{Title: "Senior DevOps Engineer", ReadFrom: "ats_api", Text: "We run Kubernetes.\nRequirements\n- Kubernetes\n- Terraform"}}
	job, ok := importer.Job(board, posting)
	if !ok || job.Title != "Senior DevOps Engineer" || job.Company != "Northwind Cloud" || job.City != "Iași" || job.Country != "Romania" ||
		job.WorkMode != "hybrid" || job.Level != "senior" || !slices.Equal(job.Areas, []string{"DevOps"}) || !slices.Equal(job.Domains, []string{"it"}) || !slices.Contains(job.MustHave, "Kubernetes") ||
		job.Source != "import" || job.Origin != "greenhouse" || job.ExternalID != "4001" || job.Board != "northwind" || job.PostedAt != &posted {
		t.Fatalf("job: %+v %v", job, ok)
	}

	remote := posting
	remote.Places, remote.Page.Country, remote.Page.WorkMode = []string{"Remote"}, "RO", "remote"
	if job, ok := importer.Job(board, remote); !ok || job.Country != "Romania" || job.City != "" || job.WorkMode != "remote" {
		t.Fatalf("a remote job in Romania: %+v %v", job, ok)
	}
	for name, mutate := range map[string]func(*fetch.Posting){
		"outside Romania": func(p *fetch.Posting) { p.Places = []string{"Berlin, Germany"} },
		"in no area":      func(p *fetch.Posting) { p.Page.Title = "Account Executive" },
		"without an id":   func(p *fetch.Posting) { p.ID = "" },
	} {
		p := posting
		mutate(&p)
		if _, ok := importer.Job(board, p); ok {
			t.Errorf("a job %s is not kept", name)
		}
	}
}

// memoryImports records what the importer stores; fresh holds the ids whose descriptions count as recent.
type memoryImports struct {
	jobs        map[string]models.ImportedJob
	closed      []string
	listed      [][]string
	fresh       map[string]bool
	seen        []string
	failWith    error
	freshErr    error
	unlistedErr error
}

func (m *memoryImports) Fresh(context.Context, string, string, time.Time) (map[string]bool, error) {
	return m.fresh, m.freshErr
}

func (m *memoryImports) Seen(_ context.Context, _, _ string, ids []string, _ time.Time) error {
	m.seen = append(m.seen, ids...)
	return m.failWith
}

func (m *memoryImports) CloseUnlisted(_ context.Context, origins, boards []string) (int64, error) {
	keys := []string{}
	for i := range origins {
		keys = append(keys, origins[i]+"/"+boards[i])
	}
	m.listed = append(m.listed, keys)
	return 2, m.unlistedErr
}

func (m *memoryImports) Import(_ context.Context, job models.ImportedJob, _ time.Time) (bool, error) {
	if m.failWith != nil {
		return false, m.failWith
	}
	_, known := m.jobs[job.ExternalID]
	m.jobs[job.ExternalID] = job
	return !known, nil
}

func (m *memoryImports) Close(_ context.Context, origin, board string, _ time.Time) (int64, error) {
	m.closed = append(m.closed, origin+"/"+board)
	return 1, nil
}

// fakeLister answers each board with its postings, cut short for the boards in
// partial, or fails for the boards in fail; it describes a posting by filling
// its text, or fails for the ids in unreadable.
type fakeLister struct {
	postings   map[string][]fetch.Posting
	fail       map[string]bool
	partial    map[string]bool
	unreadable map[string]bool
	calls      []string
	described  []string
}

func (f *fakeLister) List(_ context.Context, platform, board string, country fetch.Country) ([]fetch.Posting, bool, error) {
	f.calls = append(f.calls, platform+"/"+board+"@"+country.Code)
	if f.fail[board] {
		return nil, false, errors.New("board down")
	}
	return f.postings[board], !f.partial[board], nil
}

func (f *fakeLister) Describe(_ context.Context, p *fetch.Posting) error {
	f.described = append(f.described, p.ID)
	if f.unreadable[p.ID] {
		return errors.New("page down")
	}
	p.Page.Text = "Requirements\n- Kubernetes"
	return nil
}

func freeLock(context.Context) (func(), bool, error) { return func() {}, true, nil }

func TestImporterRunsEveryBoard(t *testing.T) {
	devops := fetch.Posting{ID: "1", Places: []string{"Bucharest"}, Page: fetch.Page{Title: "DevOps Engineer"}}
	sales := fetch.Posting{ID: "2", Places: []string{"Bucharest"}, Page: fetch.Page{Title: "Account Executive"}}
	boards := []importer.Board{
		{Company: "A", Platform: "greenhouse", Board: "a"}, {Company: "B", Platform: "lever", Board: "b"},
		{Company: "C", Platform: "ashby", Board: "c"}, {Company: "D", Platform: "ashby", Board: "d"},
	}
	lister := &fakeLister{postings: map[string][]fetch.Posting{"a": {devops, sales}, "d": {}}, fail: map[string]bool{"b": true}}
	store := &memoryImports{jobs: map[string]models.ImportedJob{}}
	im := importer.New(store, lister, boards, nil, freeLock, time.Millisecond)

	results := im.RunOnce(context.Background())
	if len(results) != 4 || results[0].Listed != 2 || results[0].Kept != 1 || results[0].New != 1 || results[0].Closed != 1 ||
		results[1].Err == nil || results[2].Listed != 0 || results[3].Closed != 0 {
		t.Fatalf("results: %+v", results)
	}
	if !slices.Equal(store.closed, []string{"greenhouse/a"}) {
		t.Fatalf("only a board that listed jobs closes the missing ones: %v", store.closed)
	}
	if len(store.listed) != 1 || !slices.Equal(store.listed[0], []string{"greenhouse/a", "lever/b", "ashby/c", "ashby/d"}) {
		t.Fatalf("a full run closes the jobs of boards no longer listed: %v", store.listed)
	}
	if again := im.RunOnce(context.Background()); again[0].New != 0 || again[0].Kept != 1 {
		t.Fatalf("a known job is not new: %+v", again[0])
	}

	lister.partial = map[string]bool{"a": true}
	if cut := im.RunOnce(context.Background()); !cut[0].Partial || cut[0].Kept != 1 || len(store.closed) != 2 {
		t.Fatalf("a list cut short closes nothing: %+v %v", cut[0], store.closed)
	}
	lister.partial = nil
	store.unlistedErr = errors.New("database down")
	if again := im.RunOnce(context.Background()); len(again) != 4 {
		t.Fatalf("a failure to close unlisted jobs is only logged: %+v", again)
	}
	store.failWith = errors.New("database down")
	if failed := im.RunOnce(context.Background()); failed[0].Err == nil || failed[0].Kept != 0 {
		t.Fatalf("a failed write stops that board: %+v", failed[0])
	}

	busy := importer.New(store, lister, boards, nil, func(context.Context) (func(), bool, error) { return nil, false, nil }, 0)
	broken := importer.New(store, lister, boards, nil, func(context.Context) (func(), bool, error) { return nil, false, errors.New("no db") }, 0)
	if busy.RunOnce(context.Background()) != nil || broken.RunOnce(context.Background()) != nil {
		t.Fatal("no run without the lock")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stopped := importer.New(store, &fakeLister{}, boards, nil, freeLock, time.Hour)
	runs := len(store.listed)
	if results := stopped.RunOnce(ctx); len(results) != 1 || len(store.listed) != runs {
		t.Fatalf("a stopped run ends between boards and closes nothing unlisted: %+v %v", results, store.listed)
	}
	lister.calls = nil
	importer.New(store, lister, boards[:1], nil, freeLock, 0).Run(ctx, time.Hour)
	if !slices.Equal(lister.calls, []string{"greenhouse/a@RO"}) {
		t.Fatalf("Run imports at once, then stops with its context: %v", lister.calls)
	}
}
