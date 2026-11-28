package unit_test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"cv/internal/cvdoc"
	"cv/internal/data/repositories"
	"cv/internal/identity"
	"cv/internal/models"
	"cv/internal/routes"
	"cv/internal/services"
	"cv/internal/writer"
	"cv/tests/testutil"
)

func TestReadinessCountsTheWebAppsChecks(t *testing.T) {
	equal(t, cvdoc.Readiness(models.Profile{}), 0)
	equal(t, cvdoc.ReadyForClaude(models.Profile{}), false)

	// Contact data alone reaches the percent, but tells nothing of the person's work.
	contact := models.Profile{Name: "Ana Pop", Headline: "Accountant", About: "About", Email: "a@b.ro", Phone: "+40",
		LinkedIn: "li", City: "Iași", Country: "Romania", WorkAuthorization: "citizen"}
	equal(t, cvdoc.Readiness(contact), 62)
	equal(t, cvdoc.HasWork(contact), false)
	equal(t, cvdoc.ReadyForClaude(contact), false)

	working := contact
	working.Projects = []models.Project{{Name: "Books", Bullets: []models.Bullet{{Text: "Kept the books of 12 clients"}}}}
	equal(t, cvdoc.HasWork(working), true)
	equal(t, cvdoc.ReadyForClaude(working), true)
	working.Headline, working.About = "", ""
	equal(t, cvdoc.Readiness(working), 46)
	equal(t, cvdoc.ReadyForClaude(working), false)

	full := sampleProfile(t)
	equal(t, cvdoc.Readiness(full) >= cvdoc.ReadyPercent, true)
	equal(t, cvdoc.ReadyForClaude(full), true)
}

// An incomplete profile gets no questions, no analysis, no CV and no letter: Claude is never asked.
func TestNothingIsAskedForAnIncompleteProfile(t *testing.T) {
	h := &cvHarness{repo: newMemoryCvs(), sources: newFakeSources(t), printer: &fakePrinter{}}
	h.sources.profile = models.Profile{Name: "Ana Pop", Email: "a@b.ro"}
	prober := &fakeProber{available: true}
	service := services.NewCvService(h.repo, h.sources, h.printer, func() {}).
		WithProbes(&memoryProbes{kept: map[string]repositories.StoredProbe{}}, prober)
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Cvs: service, Writer: writer.New(nil)})
	dev := identity.Dev()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r.WithContext(identity.WithContext(r.Context(), dev)))
	})

	for _, path := range []string{"/api/v1/probe", "/api/v1/jobs/" + jobID + "/probe"} {
		if status, body := testutil.Do(t, handler, http.MethodGet, path, nil); status != http.StatusOK || body["status"] != "incomplete" {
			t.Fatalf("%s: %d %v", path, status, body)
		}
	}
	if status, body := testutil.Do(t, handler, http.MethodPost, "/api/v1/jobs/"+jobID+"/analysis", nil); status != http.StatusAccepted ||
		body["status"] != "incomplete" {
		t.Fatalf("analysis: %d %v", status, body)
	}
	status, body := testutil.Do(t, handler, http.MethodPost, cvsPath, map[string]any{"job_id": jobID})
	envelope, _ := body["error"].(map[string]any)
	context, _ := envelope["context"].(map[string]any)
	if status != http.StatusConflict || testutil.ErrorCode(body) != "CONSTRAINT_VIOLATION" || context["reason"] != "profile_incomplete" ||
		context["percent"] != float64(15) || context["needed"] != float64(cvdoc.ReadyPercent) {
		t.Fatalf("cv: %d %v", status, body)
	}
	equal(t, prober.calls+prober.profileCalls+prober.analyzed, 0)
	if _, list := testutil.Do(t, handler, http.MethodGet, cvsPath, nil); len(list["items"].([]any)) != 0 {
		t.Fatalf("no CV kept: %v", list)
	}
}

func TestNoLetterForACvWrittenFromAnIncompleteProfile(t *testing.T) {
	h := &cvHarness{repo: newMemoryCvs(), sources: newFakeSources(t), printer: &fakePrinter{}}
	letters := &memoryLetters{kept: map[uuid.UUID]models.Letter{}, cvs: h.repo, settled: make(chan struct{}, 4), profile: models.Profile{Name: "Ana"}}
	author := &fakeLetterWriter{release: make(chan struct{})}
	service := services.NewCvService(h.repo, h.sources, h.printer, func() {}).WithLetters(letters, author)
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Cvs: service, Writer: writer.New(nil)})
	dev := identity.Dev()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r.WithContext(identity.WithContext(r.Context(), dev)))
	})
	_, cv := testutil.Do(t, handler, http.MethodPost, cvsPath, map[string]any{"job_id": jobID})
	id := cv["id"].(string)
	h.repo.ready(t, id)
	status, body := testutil.Do(t, handler, http.MethodPost, cvsPath+"/"+id+"/letter", nil)
	envelope, _ := body["error"].(map[string]any)
	context, _ := envelope["context"].(map[string]any)
	if status != http.StatusConflict || context["reason"] != "profile_incomplete" || len(author.asked) != 0 {
		t.Fatalf("letter: %d %v", status, body)
	}
}
