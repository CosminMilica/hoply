package unit_test

import (
	"context"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"jobs/internal/fetch"
	"jobs/internal/identity"
	"jobs/tests/testutil"
)

// The fixtures below are synthetic: shaped like each board's public API answer
// (checked once against live boards), with invented companies and ad text.

const boardDescription = `<div class="content-intro"><p>Northwind Cloud runs payment infrastructure for small shops across Europe.</p></div>` +
	`<p><strong>What You’ll Be Doing</strong></p><ul><li>Run Kubernetes clusters on AWS</li><li>Build CI/CD with Terraform</li></ul>` +
	`<p><strong>What Makes You a Great Fit</strong></p><ul><li>4+ years with Go and PostgreSQL</li></ul>` +
	`<p><strong>Bonus Points For</strong></p><ul><li>Kafka</li></ul>` +
	`<p><strong>Why You’ll Thrive at Northwind:</strong></p><ul><li>100% remote, Python guild, 9 years of history</li></ul>`

func jsonBody(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// boardServer answers the board API paths with fixtures and records hits.
type boardServer struct {
	*httptest.Server
	mu   sync.Mutex
	hits []string
}

func newBoardServer(t *testing.T, routes map[string]string) *boardServer {
	t.Helper()
	b := &boardServer{}
	b.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		b.hits = append(b.hits, r.URL.Path+"?"+r.URL.RawQuery)
		b.mu.Unlock()
		body, ok := routes[r.URL.Path]
		switch {
		case !ok:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		case strings.HasPrefix(body, "<"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, body)
		default:
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprint(w, body)
		}
	}))
	t.Cleanup(b.Close)
	return b
}

const bambooHRDescription = `<p><strong>What you will do</strong></p><ul><li>You write firmware in C++ for sensors.</li></ul>` +
	`<p><strong>Requirements</strong></p><ul><li>3+ years with C++ and Linux.</li></ul>`

func boardFixtures(t *testing.T) map[string]string {
	return map[string]string{
		"/careers/42/detail": jsonBody(t, map[string]any{"result": map[string]any{"jobOpening": map[string]any{
			"jobOpeningName": "Embedded Software Engineer", "locationType": "2", "datePosted": "2026-11-20",
			"location":    map[string]any{"city": "Iasi", "state": "Iasi", "addressCountry": "Romania"},
			"description": bambooHRDescription,
		}}}),
		"/v1/boards/northwind/jobs/4001": jsonBody(t, map[string]any{
			"id": 4001, "title": "Platform Engineer | Remote", "company_name": "Northwind Cloud",
			"location": map[string]any{"name": "Cluj-Napoca, Romania (Hybrid)"}, "content": stdhtml.EscapeString(boardDescription),
			"absolute_url": "https://job-boards.greenhouse.io/northwind/jobs/4001",
		}),
		"/v0/postings/northwind-cloud/aa11": jsonBody(t, map[string]any{
			"id": "aa11", "text": "Site Reliability Engineer", "categories": map[string]any{"location": "Bucharest, Romania"},
			"workplaceType": "hybrid", "country": "RO", "description": "<div><strong>The Role</strong></div><div>Keep payments up.</div>",
			"lists": []map[string]any{{"text": "Requirements", "content": "<li>Linux</li><li>Prometheus</li>"},
				{"text": "Nice to have", "content": "<li>Grafana</li>"}},
			"additional": "<div>Equal opportunity employer. Docker everywhere.</div>",
		}),
		"/posting-api/job-board/northwind": jsonBody(t, map[string]any{"apiVersion": "1", "jobs": []map[string]any{
			{"id": "other", "title": "Other", "descriptionHtml": "<p>x</p>"},
			{"id": "bb22", "title": "Data Engineer", "location": "Europe", "isRemote": true, "workplaceType": "",
				"address":         map[string]any{"postalAddress": map[string]any{"addressCountry": "European Union"}},
				"descriptionHtml": "<h2>What you'll do</h2><ul><li>Model data in Snowflake</li></ul>"},
		}}),
		"/v1/companies/NorthwindGroup/postings/7440001": jsonBody(t, map[string]any{
			"id": "7440001", "name": "Junior Network Administrator", "company": map[string]any{"name": "Northwind Group", "identifier": "NorthwindGroup"},
			"location": map[string]any{"city": "Iasi", "country": "ro", "remote": false, "hybrid": true},
			"jobAd": map[string]any{"sections": map[string]any{
				"companyDescription":    map[string]any{"title": "Company Description", "text": "<p>We make Kubernetes parts.</p>"},
				"jobDescription":        map[string]any{"title": "Job Description", "text": "<p>You keep the office network running for 300 colleagues every day.</p>"},
				"qualifications":        map[string]any{"title": "Qualifications", "text": "<ul><li>DNS and DHCP</li><li>1 year of networking</li></ul>"},
				"additionalInformation": map[string]any{"title": "Additional Information", "text": "<p>Python lunches.</p>"},
			}},
		}),
		"/api/v1/widget/accounts/northwind": jsonBody(t, map[string]any{"name": "Northwind Labs", "description": "", "jobs": []map[string]any{
			{"title": "ML Engineer", "shortcode": "AB12CD", "city": "Paris", "country": "France", "telecommuting": true,
				"description": "<p><strong>About You</strong></p><ul><li>PyTorch</li></ul>"},
		}}),
	}
}

func boardReader(api, eu, pages *boardServer) *fetch.Reader {
	return fetch.New(fetch.Options{AllowLoopback: true, Hosts: map[string]string{
		"boards-api.greenhouse.io": api.URL, "api.lever.co": api.URL, "api.eu.lever.co": eu.URL, "api.ashbyhq.com": api.URL,
		"api.smartrecruiters.com": api.URL, "apply.workable.com": api.URL, "northwind.bamboohr.com": api.URL,
		"job-boards.greenhouse.io": pages.URL, "jobs.ashbyhq.com": pages.URL, "jobs.lever.co": pages.URL,
	}})
}

func TestBoardAPIs(t *testing.T) {
	api := newBoardServer(t, boardFixtures(t))
	eu := newBoardServer(t, map[string]string{"/v0/postings/northwind-cloud/aa11": boardFixtures(t)["/v0/postings/northwind-cloud/aa11"]})
	pages := newBoardServer(t, map[string]string{})
	reader := boardReader(api, eu, pages)
	cases := []struct {
		link                                    string
		title, company, location, city, country string
		workMode, contains                      string
	}{
		{"https://job-boards.greenhouse.io/northwind/jobs/4001", "Platform Engineer | Remote", "Northwind Cloud",
			"Cluj-Napoca, Romania (Hybrid)", "", "", "", "What You’ll Be Doing\n- Run Kubernetes clusters on AWS"},
		{"https://boards.greenhouse.io/embed/job_app?for=northwind&token=4001", "Platform Engineer | Remote", "Northwind Cloud", "", "", "", "", ""},
		{"https://job-boards.eu.greenhouse.io/northwind/jobs/4001", "Platform Engineer | Remote", "Northwind Cloud", "", "", "", "", ""},
		{"https://careers.northwind.example/open?gh_jid=4001&for=northwind", "Platform Engineer | Remote", "Northwind Cloud", "", "", "", "", ""},
		{"https://jobs.lever.co/northwind-cloud/aa11", "Site Reliability Engineer", "Northwind Cloud", "Bucharest, Romania", "", "RO",
			"hybrid", "The Role\nKeep payments up.\nRequirements\n- Linux"},
		{"https://jobs.eu.lever.co/northwind-cloud/aa11/apply", "Site Reliability Engineer", "Northwind Cloud", "", "", "", "hybrid", ""},
		{"https://jobs.ashbyhq.com/northwind/bb22", "Data Engineer", "Northwind", "Europe", "", "European Union", "remote",
			"What you'll do\n- Model data in Snowflake"},
		{"https://jobs.smartrecruiters.com/NorthwindGroup/7440001-junior-network-administrator", "Junior Network Administrator",
			"Northwind Group", "", "Iasi", "ro", "hybrid", "You keep the office network"},
		{"https://northwind.bamboohr.com/careers/42", "Embedded Software Engineer", "Northwind", "", "Iasi", "Romania", "hybrid",
			"What you will do\n- You write firmware in C++ for sensors."},
		{"https://apply.workable.com/northwind/j/ab12cd/", "ML Engineer", "Northwind Labs", "", "Paris", "France", "remote", "About You\n- PyTorch"},
	}
	for _, tc := range cases {
		got, err := reader.Read(context.Background(), tc.link)
		if err != nil {
			t.Fatalf("%s: %v", tc.link, err)
		}
		if got.ReadFrom != "ats_api" || got.Title != tc.title || got.Company != tc.company ||
			(tc.location != "" && got.Location != tc.location) || (tc.city != "" && got.City != tc.city) ||
			(tc.country != "" && got.Country != tc.country) || (tc.workMode != "" && got.WorkMode != tc.workMode) ||
			!strings.Contains(got.Text, tc.contains) {
			t.Errorf("%s: %+v", tc.link, got)
		}
	}
	if len(eu.hits) != 1 || !strings.HasPrefix(eu.hits[0], "/v0/postings/northwind-cloud/aa11") {
		t.Fatalf("the EU Lever host uses the EU API: %v", eu.hits)
	}
}

func TestBoardFallsBackToThePage(t *testing.T) {
	pageHTML := page(`<meta property="og:site_name" content="Northwind">`, `<main><h1>Fallback title</h1><p>`+jobText+`</p></main>`)
	api := newBoardServer(t, map[string]string{
		"/posting-api/job-board/northwind": `{"jobs":[]}`,
		"/v1/boards/broken/jobs/1":         "<html>not json</html>",
	})
	pages := newBoardServer(t, map[string]string{
		"/northwind/missing": pageHTML, "/northwind/jobs/9": pageHTML, "/broken/jobs/1": pageHTML, "/site/gone": pageHTML,
	})
	reader := boardReader(api, api, pages)
	for _, link := range []string{
		"https://jobs.ashbyhq.com/northwind/missing",        // the board does not list the job
		"https://job-boards.greenhouse.io/northwind/jobs/9", // the API answers 404
		"https://job-boards.greenhouse.io/broken/jobs/1",    // the API answers HTML
		"https://jobs.lever.co/site/gone",                   // 404 again
	} {
		got, err := reader.Read(context.Background(), link)
		if err != nil || got.ReadFrom != "page_text" || got.Title != "Fallback title" || got.Company != "Northwind" {
			t.Errorf("%s: %+v %v", link, got, err)
		}
	}
	if _, err := reader.Read(context.Background(), "https://jobs.ashbyhq.com/northwind/nowhere"); reason(t, err) != fetch.FetchFailed {
		t.Fatalf("the page's own failure is reported: %v", err)
	}
}

func TestBoardAnswersWithoutAJob(t *testing.T) {
	api := newBoardServer(t, map[string]string{
		"/v1/boards/empty/jobs/1":         `{"title":""}`,
		"/v0/postings/empty/1":            `{"text":""}`,
		"/posting-api/job-board/bad":      `[`,
		"/v1/companies/Empty/postings/1":  `{"name":""}`,
		"/api/v1/widget/accounts/empty":   `{"name":"x","jobs":[{"shortcode":"OTHER"}]}`,
		"/api/v1/widget/accounts/bad":     `{`,
		"/v1/companies/Broken/postings/2": `{`,
		"/careers/1/detail":               `{"result":{"jobOpening":{"jobOpeningName":""}}}`,
		"/careers/2/detail":               `{`,
	})
	pages := newBoardServer(t, map[string]string{})
	reader := fetch.New(fetch.Options{AllowLoopback: true, Hosts: map[string]string{
		"boards-api.greenhouse.io": api.URL, "api.lever.co": api.URL, "api.ashbyhq.com": api.URL, "api.smartrecruiters.com": api.URL,
		"apply.workable.com": api.URL, "job-boards.greenhouse.io": pages.URL, "jobs.lever.co": pages.URL, "jobs.ashbyhq.com": pages.URL,
		"jobs.smartrecruiters.com": pages.URL, "empty.bamboohr.com": api.URL,
	}})
	for _, link := range []string{"https://job-boards.greenhouse.io/empty/jobs/1", "https://jobs.lever.co/empty/1",
		"https://jobs.ashbyhq.com/bad/1", "https://jobs.smartrecruiters.com/Empty/1", "https://jobs.smartrecruiters.com/Broken/2-x",
		"https://apply.workable.com/empty/j/SHORT", "https://apply.workable.com/bad/j/SHORT",
		"https://empty.bamboohr.com/careers/1", "https://empty.bamboohr.com/careers/2"} {
		if _, err := reader.Read(context.Background(), link); reason(t, err) != fetch.FetchFailed {
			t.Errorf("%s: an empty board answer falls back to the page (404 here): %v", link, err)
		}
	}
	if !strings.Contains(strings.Join(api.hits, " "), "/api/v1/widget/accounts/empty?details=true") {
		t.Fatalf("Workable is asked for details: %v", api.hits)
	}
}

const greenhouseLikePage = `<!doctype html><html><head><title>Job Application for Platform Engineer at Northwind Cloud</title></head>
<body><div class="app"><a href="/northwind">Back to jobs</a>
<div class="job__title"><h1 class="section-header">Platform Engineer</h1><div class="job__location">Remote</div></div>
<div class="job__description body"><p>Northwind Cloud runs payment infrastructure for small online shops across Europe, and we want releases to be boring.</p>
<p><strong>What You’ll Be Doing</strong></p><ul><li>Run Kubernetes clusters on AWS.</li><li>Build CI/CD pipelines with Terraform.</li></ul>
<p><strong>What Makes You a Great Fit</strong></p><ul><li>4+ years with Go and PostgreSQL.</li><li>Comfortable with Linux.</li></ul>
<p><strong>Bonus Points For</strong></p><ul><li>Kafka and Helm.</li></ul>
<p><strong>Why You’ll Thrive at Northwind Cloud:</strong></p><ul><li>Python guild, hybrid offices in Cluj, 12 years of history.</li></ul></div>
<form id="application-form"><h2>Apply for this job</h2><label>First Name</label><input type="text"><label>Do you know Java?</label><select><option>Yes</option></select></form>
</div></body></html>`

func TestReadGreenhouseLikePage(t *testing.T) {
	pages := newBoardServer(t, map[string]string{"/careers/platform": greenhouseLikePage})
	reader := fetch.New(fetch.Options{AllowLoopback: true})
	got, err := reader.Read(context.Background(), pages.URL+"/careers/platform")
	if err != nil {
		t.Fatal(err)
	}
	if got.ReadFrom != "page_text" || got.Title != "Platform Engineer" || got.Company != "Northwind Cloud" ||
		strings.Contains(got.Text, "First Name") || !strings.Contains(got.Text, "\nWhat You’ll Be Doing\n- Run Kubernetes") {
		t.Fatalf("page: %+v", got)
	}

	dev := identity.Dev()
	handler := jobsHandler(newMemoryJobs(), reader, &dev)
	status, draft := testutil.Do(t, handler, http.MethodPost, jobsPath+"/read", map[string]any{"url": pages.URL + "/careers/platform"})
	if status != http.StatusOK || draft["title"] != "Platform Engineer" || draft["company"] != "Northwind Cloud" ||
		fmt.Sprint(draft["must_have"]) != "[Kubernetes AWS CI/CD Terraform Go PostgreSQL Linux]" ||
		fmt.Sprint(draft["nice_to_have"]) != "[Kafka Helm]" || draft["years_required"] != float64(4) || draft["work_mode"] != "remote" ||
		fmt.Sprint(draft["responsibilities"]) != "[Run Kubernetes clusters on AWS. Build CI/CD pipelines with Terraform.]" ||
		draft["city"] != "" || draft["read_from"] != "page_text" ||
		draft["about"] != "Northwind Cloud runs payment infrastructure for small online shops across Europe, and we want releases to be boring." {
		t.Fatalf("draft: %d %v", status, draft)
	}
}

func TestBoardDraftsThroughTheService(t *testing.T) {
	api := newBoardServer(t, boardFixtures(t))
	pages := newBoardServer(t, map[string]string{})
	dev := identity.Dev()
	handler := jobsHandler(newMemoryJobs(), boardReader(api, api, pages), &dev)
	cases := map[string]string{
		"https://job-boards.greenhouse.io/northwind/jobs/4001":      "Platform Engineer | Remote|Northwind Cloud|Cluj-Napoca|Romania|hybrid|[Kubernetes AWS CI/CD Terraform Go PostgreSQL]|[Kafka]",
		"https://jobs.lever.co/northwind-cloud/aa11":                "Site Reliability Engineer|Northwind Cloud|Bucharest|Romania|hybrid|[Linux Prometheus]|[Grafana]",
		"https://jobs.smartrecruiters.com/NorthwindGroup/7440001-x": "Junior Network Administrator|Northwind Group|Iași|Romania|hybrid|[DNS DHCP Networking]|[]",
		"https://apply.workable.com/northwind/j/AB12CD":             "ML Engineer|Northwind Labs|Paris|France|remote|[PyTorch]|[]",
		"https://northwind.bamboohr.com/careers/42":                 "Embedded Software Engineer|Northwind|Iași|Romania|hybrid|[C++ Linux]|[]",
		"https://jobs.ashbyhq.com/northwind/bb22":                   "Data Engineer|Northwind||European Union|remote|[Snowflake]|[]",
	}
	for link, want := range cases {
		status, draft := testutil.Do(t, handler, http.MethodPost, jobsPath+"/read", map[string]any{"url": link})
		got := strings.Join([]string{fmt.Sprint(draft["title"]), fmt.Sprint(draft["company"]), fmt.Sprint(draft["city"]),
			fmt.Sprint(draft["country"]), fmt.Sprint(draft["work_mode"]), fmt.Sprint(draft["must_have"]), fmt.Sprint(draft["nice_to_have"])}, "|")
		if status != http.StatusOK || got != want || draft["read_from"] != "ats_api" || draft["source"] != "link" {
			t.Errorf("%s:\n got %s (%d)\nwant %s", link, got, status, want)
		}
	}

}

func TestBoardAnswerSizeLimits(t *testing.T) {
	answer := func(padding int) string {
		jobs := []map[string]any{{"id": "pad", "title": "Padding", "descriptionHtml": strings.Repeat("x", padding)},
			{"id": "bb22", "title": "Data Engineer", "descriptionHtml": "<p>Snowflake</p>"}}
		return jsonBody(t, map[string]any{"jobs": jobs})
	}
	api := newBoardServer(t, map[string]string{
		"/posting-api/job-board/big":  answer(5 << 20),
		"/posting-api/job-board/huge": answer(11 << 20),
	})
	pageHTML := page("", `<main><h1>From the page</h1><p>`+jobText+`</p></main>`)
	pages := newBoardServer(t, map[string]string{"/big/bb22": pageHTML, "/huge/bb22": pageHTML})
	reader := fetch.New(fetch.Options{AllowLoopback: true, Hosts: map[string]string{"api.ashbyhq.com": api.URL, "jobs.ashbyhq.com": pages.URL}})
	if got, err := reader.Read(context.Background(), "https://jobs.ashbyhq.com/big/bb22"); err != nil || got.ReadFrom != "ats_api" || got.Title != "Data Engineer" {
		t.Fatalf("a 5 MB board answer is read: %+v %v", got, err)
	}
	if got, err := reader.Read(context.Background(), "https://jobs.ashbyhq.com/huge/bb22"); err != nil || got.ReadFrom != "page_text" || got.Title != "From the page" {
		t.Fatalf("an 11 MB board answer falls back to the page: %+v %v", got, err)
	}
	bigPage := newBoardServer(t, map[string]string{"/p": page("", strings.Repeat("<p>x</p>", 300_000))})
	if _, err := reader.Read(context.Background(), bigPage.URL+"/p"); reason(t, err) != fetch.TooLarge {
		t.Fatalf("pages keep the 2 MB cap: %v", err)
	}
}
