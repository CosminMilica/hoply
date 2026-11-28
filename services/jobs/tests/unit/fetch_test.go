package unit_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"jobs/internal/fetch"
)

func TestRefusedAddresses(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "::1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254", "fe80::1",
		"fc00::1", "fd12::1", "224.0.0.1", "ff02::1", "0.0.0.0", "::", "100.64.0.1", "::ffff:10.0.0.1", "::ffff:127.0.0.1",
		"64:ff9b::a00:1", "2002:a00:1::1", "255.255.255.255", "240.0.0.1", "198.18.0.1"} {
		if !fetch.Refused(netip.MustParseAddr(raw)) {
			t.Errorf("%s must be refused", raw)
		}
	}
	for _, raw := range []string{"8.8.8.8", "1.1.1.1", "93.184.216.34", "2606:4700:4700::1111", "::ffff:8.8.8.8"} {
		if fetch.Refused(netip.MustParseAddr(raw)) {
			t.Errorf("%s must be allowed", raw)
		}
	}
	if !fetch.Refused(netip.Addr{}) {
		t.Error("an invalid address is refused")
	}
	cause := errors.New("cause")
	if wrapped := (&fetch.Error{Reason: fetch.FetchFailed, Err: cause}); !errors.Is(wrapped, cause) || wrapped.Error() != "page unreadable: fetch_failed: cause" {
		t.Errorf("fetch.Error wraps its cause: %v", wrapped)
	}
}

func TestLoginWalledHosts(t *testing.T) {
	for host, walled := range map[string]bool{
		"linkedin.com": true, "www.linkedin.com": true, "ro.LinkedIn.com.": true, "glassdoor.com": true,
		"www.glassdoor.co.uk": true, "glassdoor.ro": true, "notlinkedin.com": false, "myglassdoor.com": false,
		"ejobs.ro": false,
	} {
		if fetch.LoginWalled(host) != walled {
			t.Errorf("LoginWalled(%q) != %v", host, walled)
		}
	}
}

func reason(t *testing.T, err error) fetch.Reason {
	t.Helper()
	var unreadable *fetch.Error
	if !errors.As(err, &unreadable) {
		t.Fatalf("want a fetch.Error, got %v", err)
	}
	if unreadable.Error() == "" {
		t.Fatal("errors describe themselves")
	}
	return unreadable.Reason
}

func page(head, body string) string {
	return "<!doctype html><html><head><title>Ignored</title>" + head + "</head><body>" + body + "</body></html>"
}

func ldJSON(data string) string { return `<script type="application/ld+json">` + data + `</script>` }

func serve(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func html(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, content)
	}
}

var testReader = fetch.New(fetch.Options{AllowLoopback: true})

func TestReadsJobPostingJSONLD(t *testing.T) {
	description := `<p>We build <b>payments</b>.</p><h3>Requirements</h3><ul><li>Go &amp; Kubernetes</li><li>5+ years</li></ul>` +
		`<p>Nice to have:<br>Kafka</p>`
	graph := `{"@context":"https://schema.org","@graph":[{"@type":"Organization","name":"Other"},` +
		`{"@type":"JobPosting","title":"Senior Go Engineer &amp; SRE","hiringOrganization":{"@type":"Organization","name":"Northwind"},` +
		`"jobLocation":[{"@type":"Place"},{"@type":"Place","address":{"addressLocality":"Cluj","addressCountry":{"name":"RO"}}}],` +
		`"description":` + fmt.Sprintf("%q", description) + `}]}`
	var agent string
	server := serve(t, func(w http.ResponseWriter, r *http.Request) {
		agent = r.UserAgent()
		html(page(ldJSON(`{not json`)+ldJSON(graph), "<p>short</p>"))(w, r)
	})
	got, err := testReader.Read(context.Background(), server.URL+"/job/1")
	if err != nil {
		t.Fatal(err)
	}
	want := "We build payments.\nRequirements\n- Go & Kubernetes\n- 5+ years\nNice to have:\nKafka"
	if got.ReadFrom != "json_ld" || got.Title != "Senior Go Engineer & SRE" || got.Company != "Northwind" || got.City != "Cluj" ||
		got.Country != "RO" || got.WorkMode != "" || got.Text != want {
		t.Fatalf("JSON-LD page: %+v", got)
	}
	if agent != "crashr/0.1 (job reader)" {
		t.Fatalf("user agent: %q", agent)
	}

	escaped := `[{"@type":"WebSite"},{"@type":["Thing","JobPosting"],"title":"Remote QA","hiringOrganization":"Acme",` +
		`"jobLocationType":["TELECOMMUTE"],"jobLocation":{"address":{"addressLocality":"Iași","addressCountry":"Romania"}},` +
		`"description":"&lt;ul&gt;&lt;li&gt;Cypress&lt;/li&gt;&lt;li&gt;Playwright&lt;/li&gt;&lt;/ul&gt;"}]`
	server = serve(t, html(page(ldJSON(escaped), "")))
	got, err = testReader.Read(context.Background(), server.URL)
	if err != nil || got.Title != "Remote QA" || got.Company != "Acme" || got.WorkMode != "remote" || got.City != "Iași" ||
		got.Country != "Romania" || got.Text != "- Cypress\n- Playwright" {
		t.Fatalf("escaped JSON-LD: %+v %v", got, err)
	}

	typed := `{"@type":"http://schema.org/JobPosting","title":"Typed","jobLocation":{"address":"Bucharest"}}`
	server = serve(t, html(page(ldJSON(typed), "")))
	if got, err = testReader.Read(context.Background(), server.URL); err != nil || got.Title != "Typed" || got.City != "" || got.Text != "" {
		t.Fatalf("typed JSON-LD: %+v %v", got, err)
	}
}

var jobText = strings.Repeat("We are hiring a platform engineer to run Kubernetes and Terraform in production. ", 5)

func TestReadsMainText(t *testing.T) {
	body := `<header>Site header</header><nav>Menu Jobs About</nav><script>var x = "script";</script><style>p{}</style>` +
		`<main><h1>Platform Engineer</h1><p>` + jobText + `</p><h2>Requirements</h2><ul><li>Linux</li><li>Docker</li></ul>` +
		`<form><input type="text" name="q"></form></main><footer>Footer text</footer>`
	server := serve(t, html(page("", body)))
	got, err := testReader.Read(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReadFrom != "page_text" || !strings.HasPrefix(got.Text, "Platform Engineer\n") || !strings.Contains(got.Text, "\n- Linux\n- Docker") {
		t.Fatalf("main text: %+v", got)
	}
	for _, chrome := range []string{"Site header", "Menu", "script", "Footer", "Ignored"} {
		if strings.Contains(got.Text, chrome) {
			t.Fatalf("main text keeps %q: %q", chrome, got.Text)
		}
	}
	article := serve(t, html(page("", `<div>Outside</div><article><p>`+jobText+`</p></article>`)))
	if got, err := testReader.Read(context.Background(), article.URL); err != nil || strings.Contains(got.Text, "Outside") {
		t.Fatalf("article text: %+v %v", got, err)
	}
	whole := serve(t, html(page("", `<header>h</header><div><p>`+jobText+`</p><br><span>end</span></div>`)))
	if got, err := testReader.Read(context.Background(), whole.URL); err != nil || !strings.HasSuffix(got.Text, "end") {
		t.Fatalf("body text: %+v %v", got, err)
	}
}

func TestUnreadablePages(t *testing.T) {
	big := strings.Repeat("x", 4096)
	small := fetch.New(fetch.Options{AllowLoopback: true, MaxBytes: 1024})
	cases := []struct {
		name    string
		handler http.HandlerFunc
		reader  *fetch.Reader
		path    string
		reason  fetch.Reason
	}{
		{"short text", html(page("", "<main><p>Apply now</p></main>")), testReader, "/", fetch.NoJobData},
		{"login form", html(page("", `<main><form><input type="PASSWORD" name="p"></form><p>`+jobText+`</p></main>`)), testReader, "/", fetch.LoginRequired},
		{"login path", html(page("", "<p>Sign in</p>")), testReader, "/account/login?next=/job", fetch.LoginRequired},
		{"not found", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "gone", http.StatusNotFound) }, testReader, "/", fetch.FetchFailed},
		{"json", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"title":"x"}`)
		}, testReader, "/", fetch.FetchFailed},
		{"no content type", func(w http.ResponseWriter, _ *http.Request) {
			w.Header()["Content-Type"] = nil
			fmt.Fprint(w, "plain")
		}, testReader, "/", fetch.FetchFailed},
		{"declared too large", html(page("", big)), small, "/", fetch.TooLarge},
		{"streamed too large", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			for range 4 {
				fmt.Fprint(w, big)
				w.(http.Flusher).Flush()
			}
		}, small, "/", fetch.TooLarge},
		{"redirect to private", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://10.255.255.1/job", http.StatusFound)
		}, testReader, "/", fetch.BlockedAddress},
		{"redirect to metadata", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://[fd00:ec2::254]/latest", http.StatusFound)
		}, testReader, "/", fetch.BlockedAddress},
		{"redirect to linkedin", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://www.linkedin.com/jobs/view/1", http.StatusFound)
		}, testReader, "/", fetch.LoginRequired},
		{"redirect to ftp", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "ftp://example.com/job", http.StatusFound)
		}, testReader, "/", fetch.FetchFailed},
		{"redirect loop", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, r.URL.Path+"x", http.StatusFound)
		}, testReader, "/", fetch.FetchFailed},
		{"slow", func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(300 * time.Millisecond)
			html(page("", jobText))(w, nil)
		}, fetch.New(fetch.Options{AllowLoopback: true, Timeout: 50 * time.Millisecond}), "/", fetch.FetchFailed},
		{"loopback in production", html(page("", jobText)), fetch.New(fetch.Options{}), "/", fetch.BlockedAddress},
	}
	for _, tc := range cases {
		server := serve(t, tc.handler)
		_, err := tc.reader.Read(context.Background(), server.URL+tc.path)
		if got := reason(t, err); got != tc.reason {
			t.Errorf("%s: reason %q, want %q (%v)", tc.name, got, tc.reason, err)
		}
	}
	hops := 0
	chain := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if hops++; hops <= 3 {
			http.Redirect(w, r, fmt.Sprintf("/hop%d", hops), http.StatusFound)
			return
		}
		html(page("", jobText))(w, r)
	})
	if _, err := testReader.Read(context.Background(), chain.URL); err != nil {
		t.Fatalf("three redirects are allowed: %v", err)
	}
	for _, link := range []string{"ftp://example.com/job", "http://[::1", "http:///nohost", "https://www.linkedin.com/jobs/view/1",
		"https://glassdoor.co.uk/job", "http://localhost:1/job", "http://169.254.169.254/latest/meta-data"} {
		want := map[string]fetch.Reason{"https://www.linkedin.com/jobs/view/1": fetch.LoginRequired, "https://glassdoor.co.uk/job": fetch.LoginRequired,
			"http://localhost:1/job": fetch.BlockedAddress, "http://169.254.169.254/latest/meta-data": fetch.BlockedAddress}[link]
		if want == "" {
			want = fetch.FetchFailed
		}
		if _, err := fetch.New(fetch.Options{}).Read(context.Background(), link); reason(t, err) != want {
			t.Errorf("%s: %v, want %s", link, err, want)
		}
	}
}

func TestPageTitleAndCompany(t *testing.T) {
	cases := []struct{ head, body, main, title, company string }{
		{`<meta property="og:title" content="OG title"><meta property="og:site_name" content="OG Site">`, "", "", "OG title", "OG Site"},
		{"", "", "", "Ignored", ""},
		{"", "<header><h1>Header h1</h1></header>", "", "Header h1", ""},
		{"", "<header><h1>Site</h1></header>", "<h1>Main h1</h1>", "Main h1", ""},
	}
	for _, tc := range cases {
		server := serve(t, html(page(tc.head, tc.body+"<main>"+tc.main+"<p>"+jobText+"</p></main>")))
		got, err := testReader.Read(context.Background(), server.URL)
		if err != nil || got.Title != tc.title || got.Company != tc.company {
			t.Errorf("%s %s: %+v %v", tc.head, tc.body, got, err)
		}
	}
	titled := serve(t, html(`<html><head><title>Job Application for Go Engineer at Acme Labs</title></head><body><p>`+jobText+`</p></body></html>`))
	if got, err := testReader.Read(context.Background(), titled.URL); err != nil || got.Company != "Acme Labs" || got.Title != "Job Application for Go Engineer at Acme Labs" {
		t.Fatalf("title at company: %+v %v", got, err)
	}
	inline := serve(t, html(page("", `<main><p><strong>Requirements:</strong> Go and Docker</p><p>We use <b>Kafka</b> daily.</p><p>`+jobText+`</p></main>`)))
	if got, err := testReader.Read(context.Background(), inline.URL); err != nil ||
		!strings.Contains(got.Text, "Requirements:\nGo and Docker\nWe use Kafka daily.") {
		t.Fatalf("a leading bold run is its own line, an inline one is not: %+v %v", got, err)
	}
}
