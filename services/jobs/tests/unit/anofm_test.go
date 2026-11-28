package unit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"jobs/internal/fetch"
	"jobs/internal/importer"
)

// anofmJobPage is a job page of ANOFM's portal as it serves it: label and value spans, the contact among them.
const anofmJobPage = `<html><body><div class="container">
<div class="info-item"><span class="info-label">ID Ofertă:</span><span class="info-value">3441414</span></div>
<div class="info-item"><span class="info-label">Persoană de contact:</span><span class="info-value">PRICOP CORINA</span></div>
<div class="info-item"><span class="info-label">Telefon:</span><span class="info-value">0748194714</span></div>
<div class="info-item"><span class="info-label">Nivel de educație:</span><span class="info-value">Universitar - MASTER</span></div>
<div class="info-item"><span class="info-label">Tip normă:</span><span class="info-value">Cu normă întreagă</span></div>
<div class="info-item"><span class="info-label">Condiții de muncă:</span>
  <span class="info-value"><span class="badge">Muncă de birou</span> <span class="badge">Lucru cu publicul</span></span></div>
<div class="info-item"><span class="info-label">Responsabilități:</span><span class="info-value">Studii economice superioare
Experiență în utilizarea aplicației NEXUS

Stagiu CECCAR – avantaj</span></div>
</div></body></html>`

// anofmServer answers the portal's search a page at a time, and its job pages.
func anofmServer(t *testing.T, total int) (*httptest.Server, *[]map[string]any) {
	t.Helper()
	asked := []map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/entity/vw_public_job_posting":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			asked = append(asked, body)
			page, size := int(body["current"].(float64)), int(body["rowCount"].(float64))
			rows := []map[string]any{}
			for i := (page - 1) * size; i < min(page*size, total); i++ {
				rows = append(rows, map[string]any{"id": 3441414 + i, "created_at": "2026-11-08 20:13:58", "occupation": "CONTABIL SEF",
					"employer_name": fmt.Sprintf("FIRMA %d S.R.L.", i), "address_locality_name": "Iași > MUNICIPIUL IASI > IASI",
					"work_regime_name": []string{"Sediu", "Telemuncă", "Hibrid - sediu + telemuncă", "Altul"}[i%4], "contact_phone": "0748194714",
					"professional_experience_name": []string{"Experiență puțină - mai puțin de 3 ani", "Experiență medie - între 3 și 5 ani",
						"Experimentat - mai mult de 7 ani", "Fără experiență - 0 ani", ""}[i%5]})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"statusCode": 200, "rows": rows, "current": page, "total": total})
		case strings.HasPrefix(r.URL.Path, "/app/module/mediere/job/3441414"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(anofmJobPage))
		case strings.HasPrefix(r.URL.Path, "/app/module/mediere/job/"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<html><body>Oferta nu mai este disponibilă</body></html>"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server, &asked
}

func TestANOFMListsACountyAndReadsAJobsPage(t *testing.T) {
	server, asked := anofmServer(t, 230)
	reader := fetch.New(fetch.Options{AllowLoopback: true, Hosts: map[string]string{"mediere.anofm.ro": server.URL}})
	ctx := context.Background()

	postings, complete, err := reader.List(ctx, "anofm", "IS", importer.Romania)
	if err != nil || !complete || len(postings) != 230 || len(*asked) != 3 || (*asked)[2]["county_id"] != float64(23) {
		t.Fatalf("list: %d %v %v %v", len(postings), complete, err, *asked)
	}
	first := postings[0]
	if first.ID != "3441414" || first.URL != "https://mediere.anofm.ro/app/module/mediere/job/3441414" || first.Employer != "FIRMA 0 S.R.L." ||
		first.Page.Title != "Contabil sef" || first.Page.WorkMode != "onsite" || !first.InCountry || first.Places[0] != "Iași, MUNICIPIUL IASI, IASI" ||
		first.PostedAt.Format(time.DateTime) != "2026-11-08 20:13:58" || !first.NeedsDescribe() {
		t.Fatalf("posting: %+v", first)
	}
	same(t, []string{postings[1].Page.WorkMode, postings[2].Page.WorkMode, postings[3].Page.WorkMode}, []string{"remote", "hybrid", ""})
	levels := []string{}
	for _, p := range postings[:5] {
		levels = append(levels, p.Page.Level)
	}
	same(t, levels, []string{"junior", "mid", "senior", "junior", ""})

	if err := reader.Describe(ctx, &first); err != nil {
		t.Fatal(err)
	}
	want := "Responsabilități\n- Studii economice superioare\n- Experiență în utilizarea aplicației NEXUS\n- Stagiu CECCAR – avantaj\n" +
		"Condiții de muncă: Muncă de birou, Lucru cu publicul\nNivel de educație: Universitar - MASTER\nTip normă: Cu normă întreagă"
	same(t, first.Page.Text, want)
	same(t, strings.Contains(first.Page.Text, "0748194714") || strings.Contains(first.Page.Text, "PRICOP"), false)
	same(t, first.Page.Title, "Contabil sef")

	job, ok := importer.Job(importer.Board{Company: "ANOFM", Platform: "anofm", Board: "IS"}, first)
	if !ok || job.Company != "FIRMA 0 S.R.L." || job.City != "Iași" || job.Country != "Romania" || job.WorkMode != "onsite" ||
		job.Domains[0] != "accounting" || job.Origin != "anofm" || job.Level != "junior" {
		t.Fatalf("job: %+v %v", job, ok)
	}

	gone := postings[1]
	if err := reader.Describe(ctx, &gone); err == nil {
		t.Fatal("a page with no offer fails")
	}
	if err := fetch.CheckBoard("anofm", "XX"); err == nil {
		t.Fatal("an unknown county fails")
	}
	same(t, fetch.CheckBoard("anofm", "B"), nil)
}

func TestANOFMStopsAtABrokenAnswer(t *testing.T) {
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("["))
	}))
	t.Cleanup(broken.Close)
	reader := fetch.New(fetch.Options{AllowLoopback: true, Hosts: map[string]string{"mediere.anofm.ro": broken.URL}})
	if _, _, err := reader.List(context.Background(), "anofm", "IS", importer.Romania); err == nil {
		t.Fatal("a broken answer fails")
	}
}

func same(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\n got: %#v\nwant: %#v", got, want)
	}
}
