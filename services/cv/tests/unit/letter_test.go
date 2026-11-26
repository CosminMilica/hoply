package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"cv/internal/claude"
	"cv/internal/data/repositories"
	"cv/internal/grounding"
	"cv/internal/identity"
	"cv/internal/models"
	"cv/internal/render"
	"cv/internal/routes"
	"cv/internal/services"
	"cv/internal/writer"
	"cv/tests/testutil"
)

var longNote = strings.Repeat("I would like to hear how the team works on the role and what it needs first. ", 4)

var letterOutput = json.RawMessage(`{"plan":{"hook":"Containers","proofs":[{"priority":"Containers","facts":["P1.B1"]}],"why":"The role"},
"letter":[
 {"sentences":[{"text":"You need someone to own the Snowflake Administrator DevOps work day to day.","facts":["J1"]},
   {"text":"I run 12 containers in my home lab with an uptime of 99.9%.","facts":["P1.B1"]}]},
 {"sentences":[{"text":"I'm passionate about automation.","facts":["E1"]},{"text":"I automated every server with Terraform.","facts":["E1"]}]},
 {"sentences":[{"text":"I'd welcome a conversation about the role at Capgemini Romania — and where I can help first.","facts":["J1"]}]}],
"connection_note":[{"text":"I applied for the role at Capgemini Romania and run 12 containers at home.","facts":["J1","P1.B1"]},{"text":"` + longNote + `","facts":["J1"]}],
"message":{"subject":"Application: 40 services automated","sentences":[{"text":"I applied for the role at Capgemini Romania today.","facts":["J1"]}]},
"follow_up":{"subject":"Following up","sentences":[{"text":"Any news!","facts":["J1"]}]}}`)

func letterCatalogue(t *testing.T) grounding.Catalogue {
	t.Helper()
	c, job := catalogue(t)
	return c.WithJob(job)
}

func TestTheLetterCitesTheJobAndKeepsOnlyCheckedSentences(t *testing.T) {
	c := letterCatalogue(t)
	equal(t, c.Facts["J1"].Text, "Snowflake Administrator DevOps at Capgemini Romania")
	equal(t, c.JobTitle, "Snowflake Administrator DevOps")
	answer, err := grounding.ParseLetter(letterOutput)
	if err != nil {
		t.Fatal(err)
	}
	texts, refusals := c.AssembleLetter(answer, grounding.Addressing{Person: "Andrei Popescu", Recruiter: "Ana Pop", Company: "Capgemini Romania",
		Title: "Snowflake Administrator DevOps"})
	equal(t, texts.CoverLetter, "Dear Ana Pop,\n\nYou need someone to own the Snowflake Administrator DevOps work day to day. "+
		"I run 12 containers in my home lab with an uptime of 99.9%.\n\n"+
		"I'd welcome a conversation about the role at Capgemini Romania, and where I can help first.\n\nKind regards,\nAndrei Popescu")
	equal(t, texts.ConnectionNote, "Hi Ana, I applied for the role at Capgemini Romania and run 12 containers at home.")
	equal(t, texts.Message, models.Message{Subject: "Application for Snowflake Administrator DevOps",
		Body: "Hi Ana,\n\nI applied for the role at Capgemini Romania today.\n\nBest regards,\nAndrei Popescu"})
	equal(t, texts.FollowUp, models.Message{})
	equal(t, len(refusals), 5)
	equal(t, strings.Contains(strings.Join(refusals, "\n"), `says "passionate"`), true)

	anonymous, _ := c.AssembleLetter(answer, grounding.Addressing{Company: "Capgemini Romania", Title: "x"})
	equal(t, strings.HasPrefix(anonymous.CoverLetter, "Dear Capgemini Romania hiring team,\n\n"), true)
	equal(t, strings.HasSuffix(anonymous.CoverLetter, "\n\nKind regards,"), true)
	equal(t, strings.HasPrefix(anonymous.Message.Body, "Hello,\n\n"), true)
	equal(t, strings.HasPrefix(anonymous.ConnectionNote, "I applied"), true)
	nobody, _ := c.AssembleLetter(answer, grounding.Addressing{})
	equal(t, strings.HasPrefix(nobody.CoverLetter, "Dear hiring team,"), true)
	empty, _ := c.AssembleLetter(grounding.LetterAnswer{}, grounding.Addressing{})
	equal(t, empty, models.LetterTexts{})

	if reason := c.CheckLetter(grounding.Sentence{Text: "I know Snowflake well.", Facts: []string{"E1"}}); !strings.Contains(reason, "Snowflake") {
		t.Fatalf("a missing skill outside the job title: %q", reason)
	}
	// "İ" grows when lower-cased, which moved the title's position in the original text.
	for _, text := range []string{"İ applied for the snowflake administrator devops role.", "İİİİ: Snowflake Administrator DevOps"} {
		if reason := c.CheckLetter(grounding.Sentence{Text: text, Facts: []string{"E1"}}); reason != "" {
			t.Fatalf("the job title names the role, not a claim: %q %q", text, reason)
		}
	}
	if _, err := grounding.ParseLetter(json.RawMessage(`[]`)); err == nil {
		t.Fatal("a letter is an object")
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(grounding.LetterSchema()), &schema); err != nil || !strings.Contains(grounding.LetterSystem, "connection") {
		t.Fatalf("schema: %v", err)
	}
	equal(t, grounding.Closes([]models.CloseSkill{{Skill: "Argo CD", Via: []string{"FluxCD"}, Category: "GitOps"}}),
		[]grounding.Equivalent{{Skill: "Argo CD", Via: []string{"FluxCD"}, Category: "GitOps"}})
}

func TestTheLetterIsExportedWithTheCVsHeaderInEveryTemplate(t *testing.T) {
	letter := render.Letter{Header: sampleDocument(t), Date: "25 November 2026",
		Text: "Dear team,\r\n\r\nI run <12> containers.\n  \nKind regards,\nAndrei Popescu"}
	for _, template := range render.Templates {
		page, err := render.LetterHTML(letter, template)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"<title>Andrei Popescu – Cover letter</title>", `<p class="date">25 November 2026</p>`,
			`<p class="para">I run &lt;12&gt; containers.</p>`, `<p class="para">Kind regards,<br>Andrei Popescu</p>`} {
			if !strings.Contains(page, want) {
				t.Errorf("%s lacks %q", template, want)
			}
		}
		raw, err := render.LetterDOCX(letter, template)
		if err != nil {
			t.Fatal(err)
		}
		document := docxPart(t, raw, "word/document.xml")
		if !strings.Contains(document, "Kind regards,</w:t></w:r><w:r><w:br/></w:r><w:r><w:t xml:space=\"preserve\">Andrei Popescu") ||
			!strings.Contains(document, "I run &lt;12&gt; containers.") || !strings.Contains(document, ">25 November 2026<") {
			t.Errorf("%s DOCX: %s", template, document)
		}
	}
	plain, _ := render.LetterHTML(render.Letter{Header: models.CvDocument{Headline: "Engineer"}, Text: "Hi"}, "")
	if !strings.Contains(plain, "<title>Cover letter</title>") || !strings.Contains(plain, `<p class="name">Engineer</p>`) {
		t.Fatalf("no name: %s", plain)
	}
	if raw, err := render.LetterDOCX(render.Letter{Header: models.CvDocument{Headline: "Engineer"}, Text: "Hi"}, "modern"); err != nil ||
		!strings.Contains(docxPart(t, raw, "word/document.xml"), ">Engineer<") {
		t.Fatalf("no name DOCX: %v", err)
	}
}

func letterTask(t *testing.T) *repositories.Task {
	t.Helper()
	queued := task(t)
	queued.Profile = richProfile(t)
	return queued
}

func readyCv(t *testing.T) models.Cv {
	t.Helper()
	document := sampleDocument(t)
	return models.Cv{ID: uuid.New(), Status: models.StatusReady, Tailored: &document,
		Tailoring: &models.Tailoring{Positioning: "The automation engineer", Close: []models.CloseSkill{}}}
}

func TestWriterWritesTheLetterFromTheCV(t *testing.T) {
	store := &memoryAnalyses{kept: map[uuid.UUID]json.RawMessage{}}
	model := &fakeModel{replies: []modelReply{{output: analysisOutput}, {output: letterOutput}}}
	letter := writer.New(model).KeepAnalyses(store).Letter(context.Background(), letterTask(t), readyCv(t), "Ana Pop")
	equal(t, letter.Status, models.LetterReady)
	equal(t, letter.Recruiter, "Ana Pop")
	equal(t, letter.Model, "claude-sonnet-5-5")
	equal(t, letter.Refused, 5)
	equal(t, strings.HasPrefix(letter.Texts.CoverLetter, "Dear Ana Pop,"), true)
	equal(t, strings.Contains(model.last.System, "cover letter"), true)
	equal(t, strings.Contains(model.last.Prompt, `"positioning":"The automation engineer"`), true)

	off := writer.New(nil).Letter(context.Background(), letterTask(t), readyCv(t), "")
	equal(t, off, models.Letter{Status: models.LetterFailed, Reason: models.RulesOff})
	noLogin := &fakeModel{replies: []modelReply{{err: claude.ErrLoginNeeded}}}
	equal(t, writer.New(noLogin).Letter(context.Background(), letterTask(t), readyCv(t), "").Reason, models.LoginNeeded)
	refused := json.RawMessage(`{"letter":[{"sentences":[{"text":"I'm passionate.","facts":["E1"]}]}]}`)
	failing := &fakeModel{replies: []modelReply{{output: analysisOutput}, {output: json.RawMessage(`[]`)}, {output: refused}}}
	equal(t, writer.New(failing).Letter(context.Background(), letterTask(t), readyCv(t), "").Reason, models.ClaudeFailed)
	equal(t, failing.calls, 3)
	limited := &fakeModel{replies: []modelReply{{output: analysisOutput}, {err: &claude.LimitError{Until: time.Now().Add(time.Hour)}}}}
	w := writer.New(limited)
	equal(t, w.Letter(context.Background(), letterTask(t), readyCv(t), "").Reason, models.UsageLimit)
	equal(t, w.Letter(context.Background(), letterTask(t), readyCv(t), "").Reason, models.UsageLimit)
	equal(t, limited.calls, 2)
	unreadable := &fakeModel{replies: []modelReply{{output: analysisOutput}, {err: claude.ErrFailed}}}
	equal(t, writer.New(unreadable).Letter(context.Background(), letterTask(t), readyCv(t), "").Reason, models.ClaudeFailed)
}

// memoryLetters keeps letters in memory; Source answers the sample task for a known CV.
type memoryLetters struct {
	mu      sync.Mutex
	kept    map[uuid.UUID]models.Letter
	cvs     *memoryCvs
	fail    error
	settled chan struct{}
}

func (m *memoryLetters) Letter(_ context.Context, _ identity.IdentityContext, cvID uuid.UUID) (*models.Letter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if letter, ok := m.kept[cvID]; ok {
		return &letter, m.fail
	}
	return nil, m.fail
}

func (m *memoryLetters) SaveLetter(_ context.Context, _ identity.IdentityContext, cvID uuid.UUID, letter models.Letter) error {
	m.mu.Lock()
	if letter.UpdatedAt.IsZero() {
		letter.UpdatedAt = time.Now()
	}
	m.kept[cvID] = letter
	m.mu.Unlock()
	if letter.Status != models.LetterWriting && m.settled != nil {
		m.settled <- struct{}{}
	}
	return nil
}

func (m *memoryLetters) Source(ctx context.Context, ident identity.IdentityContext, cvID uuid.UUID) (*repositories.Task, error) {
	cv, err := m.cvs.Get(ctx, ident, cvID)
	if cv == nil || err != nil {
		return nil, err
	}
	return &repositories.Task{ID: cvID, Request: request(), Job: models.Job{Title: "Engineer"}}, nil
}

// fakeLetterWriter writes a fixed letter once released.
type fakeLetterWriter struct {
	release chan struct{}
	asked   []string
}

func (f *fakeLetterWriter) Letter(_ context.Context, _ *repositories.Task, _ models.Cv, recruiter string) models.Letter {
	<-f.release
	f.asked = append(f.asked, recruiter)
	return models.Letter{Status: models.LetterReady, Recruiter: recruiter, Texts: &models.LetterTexts{CoverLetter: "Dear " + recruiter + ","}}
}

func TestLettersThroughRoutes(t *testing.T) {
	h := &cvHarness{repo: newMemoryCvs(), sources: newFakeSources(t), printer: &fakePrinter{}}
	letters := &memoryLetters{kept: map[uuid.UUID]models.Letter{}, cvs: h.repo, settled: make(chan struct{}, 4)}
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
	path := cvsPath + "/" + id + "/letter"

	if status, body := testutil.Do(t, handler, http.MethodGet, path, nil); status != http.StatusOK || body["status"] != "none" {
		t.Fatalf("none yet: %d %v", status, body)
	}
	if status, body := testutil.Do(t, handler, http.MethodPost, path, nil); status != http.StatusConflict {
		t.Fatalf("not ready: %d %v", status, body)
	}
	h.repo.ready(t, id)
	if status, body := testutil.Do(t, handler, http.MethodPost, path, map[string]any{"recruiter": strings.Repeat("a", 81)}); status != http.StatusUnprocessableEntity ||
		errorField(body) != "recruiter" {
		t.Fatalf("long name: %d %v", status, body)
	}
	if status, _ := testutil.Do(t, handler, http.MethodPost, path, "{"); status != http.StatusUnprocessableEntity {
		t.Fatalf("bad JSON: %d", status)
	}
	for range 2 {
		if status, body := testutil.Do(t, handler, http.MethodPost, path, map[string]any{"recruiter": "  Ana   Pop "}); status != http.StatusAccepted ||
			body["status"] != "writing" || body["recruiter"] != "Ana Pop" {
			t.Fatalf("writing: %d %v", status, body)
		}
	}
	close(author.release)
	<-letters.settled
	status, body := testutil.Do(t, handler, http.MethodGet, path, nil)
	if status != http.StatusOK || body["status"] != "ready" || body["texts"].(map[string]any)["cover_letter"] != "Dear Ana Pop," {
		t.Fatalf("ready: %d %v", status, body)
	}
	equal(t, author.asked, []string{"Ana Pop"})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, cvsPath+"/"+id+"/letter", nil))
	<-letters.settled
	if res.Code != http.StatusAccepted {
		t.Fatalf("an empty body asks again: %d", res.Code)
	}

	parsed := uuid.MustParse(id)
	letters.mu.Lock()
	letters.kept[parsed] = models.Letter{Status: models.LetterWriting, UpdatedAt: time.Now().Add(-time.Hour)}
	letters.mu.Unlock()
	if _, body := testutil.Do(t, handler, http.MethodGet, path, nil); body["status"] != "failed" || body["reason"] != "claude_failed" {
		t.Fatalf("left writing by a stopped service: %v", body)
	}

	file := path + "/file"
	for _, tc := range []struct{ query, contentType, name, prefix string }{
		{"?format=pdf&template=modern", "application/pdf", "Andrei_Popescu_Cover_Letter_Capgemini_Romania.pdf", "%PDF"},
		{"?format=docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "Andrei_Popescu_Cover_Letter_Capgemini_Romania.docx", "PK"},
	} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, file+tc.query, strings.NewReader(`{"text":"Dear team,\n\nI can help."}`)))
		if res.Code != http.StatusOK || res.Header().Get("Content-Type") != tc.contentType ||
			res.Header().Get("Content-Disposition") != `attachment; filename=`+tc.name || !strings.HasPrefix(res.Body.String(), tc.prefix) {
			t.Fatalf("%s: %d %v", tc.query, res.Code, res.Header())
		}
	}
	if !strings.Contains(h.printer.page, "I can help.") || !strings.Contains(h.printer.page, "Carlito") {
		t.Fatalf("the printer gets the letter in its template: %q", h.printer.page)
	}
	for _, tc := range []struct {
		query string
		body  any
		field string
	}{
		{"?format=txt", map[string]any{"text": "x"}, "format"}, {"?format=pdf&template=fancy", map[string]any{"text": "x"}, "template"},
		{"?format=pdf", map[string]any{"text": "  "}, "text"}, {"?format=pdf", map[string]any{"text": strings.Repeat("a", 8001)}, "text"},
	} {
		if status, body := testutil.Do(t, handler, http.MethodPost, file+tc.query, tc.body); status != http.StatusUnprocessableEntity || errorField(body) != tc.field {
			t.Fatalf("%q: %d %v", tc.query, status, body)
		}
	}
	if status, _ := testutil.Do(t, handler, http.MethodPost, file+"?format=pdf", "{"); status != http.StatusUnprocessableEntity {
		t.Fatalf("bad JSON: %d", status)
	}
	h.printer.err = errors.New("down")
	if status, _ := testutil.Do(t, handler, http.MethodPost, file+"?format=pdf", map[string]any{"text": "x"}); status == http.StatusOK {
		t.Fatal("a printer failure fails")
	}
	unknown := cvsPath + "/" + uuid.NewString() + "/letter"
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		if status, _ := testutil.Do(t, handler, method, unknown, map[string]any{}); status != http.StatusNotFound {
			t.Fatalf("%s unknown CV: %d", method, status)
		}
	}
	if status, _ := testutil.Do(t, handler, http.MethodPost, unknown+"/file?format=pdf", map[string]any{"text": "x"}); status != http.StatusNotFound {
		t.Fatalf("unknown CV file: %d", status)
	}
	for _, p := range []string{cvsPath + "/nope/letter", cvsPath + "/nope/letter/file"} {
		if status, _ := testutil.Do(t, handler, http.MethodPost, p, map[string]any{}); status != http.StatusUnprocessableEntity {
			t.Fatalf("bad id %s: %d", p, status)
		}
	}
	if status, _ := testutil.Do(t, handler, http.MethodGet, cvsPath+"/nope/letter", nil); status != http.StatusUnprocessableEntity {
		t.Fatal("bad id on GET")
	}
	letters.fail = errors.New("down")
	if status, _ := testutil.Do(t, handler, http.MethodGet, path, nil); status == http.StatusOK {
		t.Fatal("a store failure fails")
	}
}

func TestLettersWithoutAWriter(t *testing.T) {
	h, handler := newCvHarness(t)
	_, cv := testutil.Do(t, handler, http.MethodPost, cvsPath, map[string]any{"job_id": jobID})
	id := cv["id"].(string)
	h.repo.ready(t, id)
	if _, body := testutil.Do(t, handler, http.MethodGet, cvsPath+"/"+id+"/letter", nil); body["status"] != "none" {
		t.Fatalf("none: %v", body)
	}
	if _, body := testutil.Do(t, handler, http.MethodPost, cvsPath+"/"+id+"/letter", map[string]any{}); body["status"] != "failed" || body["reason"] != "off" {
		t.Fatalf("off: %v", body)
	}
}
