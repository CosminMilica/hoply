package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"cv/internal/claude"
	"cv/internal/grounding"
	"cv/internal/models"
	"cv/internal/writer"
)

type modelReply struct {
	output json.RawMessage
	err    error
}

// fakeModel answers each run with the next reply, the last one again once they run out.
type fakeModel struct {
	replies []modelReply
	calls   int
	noToken bool
	last    claude.Request
}

func (f *fakeModel) Run(_ context.Context, req claude.Request) (json.RawMessage, error) {
	f.last = req
	reply := f.replies[min(f.calls, len(f.replies)-1)]
	f.calls++
	return reply.output, reply.err
}

func (f *fakeModel) Model() string  { return "claude-sonnet-5-5" }
func (f *fakeModel) HasToken() bool { return !f.noToken }

func syntheticOutput(t *testing.T) json.RawMessage {
	t.Helper()
	var envelope struct {
		StructuredOutput json.RawMessage `json:"structured_output"`
	}
	readJSON(t, "../testdata/claude/success.synthetic.json", &envelope)
	return envelope.StructuredOutput
}

func writeWith(t *testing.T, w *writer.Writer) (models.Written, models.Authorship) {
	t.Helper()
	queued := task(t)
	queued.CreatedAt = now
	written, by, err := w.Write(context.Background(), queued)
	if err != nil {
		t.Fatal(err)
	}
	return written, by
}

func TestWriterWritesWithClaudeAndChecksEverySentence(t *testing.T) {
	model := &fakeModel{replies: []modelReply{{output: syntheticOutput(t)}}}
	written, by := writeWith(t, writer.New(model))
	equal(t, by, models.Authorship{WrittenBy: "claude", Model: "claude-sonnet-5-5", Refused: 2})
	equal(t, written.Tailored.Experience[0].Bullets[0], "Developed and maintained CI/CD pipelines for internal services with Git.")
	equal(t, written.Profile.Headline, "DevOps Engineer")
	equal(t, written.Ats.Total, 12)
	equal(t, model.last.System != "" && model.last.Schema != "" && model.last.Prompt != "", true)

	// Asked for on 2026-08-31 the person had 5 years, so the summary's "6 years" is refused too.
	earlier := task(t)
	_, by, _ = writer.New(model).Write(context.Background(), earlier)
	equal(t, by.Refused, 3)
}

func TestWriterRetriesOnceThenUsesTheRules(t *testing.T) {
	failing := &fakeModel{replies: []modelReply{{err: claude.ErrFailed}}}
	_, by := writeWith(t, writer.New(failing))
	equal(t, by, models.Authorship{WrittenBy: "rules", RulesReason: "claude_failed"})
	equal(t, failing.calls, 1+writer.Attempts)

	unreadable := &fakeModel{replies: []modelReply{{output: json.RawMessage(`[]`)}}}
	_, by = writeWith(t, writer.New(unreadable))
	equal(t, by.RulesReason, "claude_failed")

	recovering := &fakeModel{replies: []modelReply{{output: analysisOutput}, {err: claude.ErrFailed}, {output: syntheticOutput(t)}}}
	_, by = writeWith(t, writer.New(recovering))
	equal(t, by.WrittenBy, "claude")
	equal(t, recovering.calls, 4)
}

func TestWriterUsesTheRulesWithoutAWorkingToken(t *testing.T) {
	_, by := writeWith(t, writer.New(&fakeModel{replies: []modelReply{{err: claude.ErrNoToken}}, noToken: true}))
	equal(t, by, models.Authorship{WrittenBy: "rules", RulesReason: "no_token"})

	model := &fakeModel{replies: []modelReply{{err: claude.ErrLoginNeeded}, {output: syntheticOutput(t)}}}
	w := writer.New(model)
	_, by = writeWith(t, w)
	equal(t, by.RulesReason, "login_needed")
	equal(t, w.Status().State, "login_needed")
	_, by = writeWith(t, w)
	equal(t, by.WrittenBy, "claude")
	equal(t, w.Status().State, "ok")
}

func TestWriterWaitsOutAUsageLimit(t *testing.T) {
	until := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	model := &fakeModel{replies: []modelReply{{err: &claude.LimitError{Until: until}}}}
	w := writer.New(model)
	_, by := writeWith(t, w)
	equal(t, by.RulesReason, "usage_limit")
	equal(t, by.LimitUntil.Equal(until), true)
	_, by = writeWith(t, w)
	equal(t, by.RulesReason, "usage_limit")
	equal(t, model.calls, 1)
	status := w.Status()
	equal(t, status.State, "usage_limit")
	equal(t, status.LimitUntil.Equal(until), true)
}

func TestWriterStopsWithItsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := writer.New(&fakeModel{replies: []modelReply{{err: claude.ErrFailed}}}).Write(ctx, task(t)); !errors.Is(err, context.Canceled) {
		t.Fatalf("a stopping service does not finish the CV: %v", err)
	}
}

func TestWriterStatus(t *testing.T) {
	equal(t, writer.New(nil).Status(), models.WriterStatus{Writer: "rules", State: "off"})
	equal(t, writer.New(&fakeModel{noToken: true}).Status(), models.WriterStatus{Writer: "claude", Model: "claude-sonnet-5-5", State: "no_token"})
	equal(t, writer.New(&fakeModel{}).Status().State, "ok")

	t.Setenv("CV_WRITER", "")
	equal(t, writer.FromEnv().Status().Writer, "rules")
	t.Setenv("CV_WRITER", "claude-cli")
	t.Setenv("CLAUDE_TOKEN_FILE", "")
	equal(t, writer.FromEnv().Status(), models.WriterStatus{Writer: "claude", Model: claude.DefaultModel, State: "no_token"})
}

var analysisOutput = json.RawMessage(`{"role":"Run the Snowflake platform","context":"Consultancy","seniority":"senior",
"priorities":[{"theme":"Infrastructure as Code","ad_words":["Terraform"],"why":"Manual setup"}],
"responsibilities":[{"index":1,"theme":"Infrastructure as Code"},{"index":3,"theme":"Infrastructure as Code"}],"traits":[],"terms":["Terraform"],
"requirements":[{"name":"Terraform","importance":"must"},{"name":"Python","importance":"nice"},{"name":"terraform","importance":"nice"}]}`)

// memoryAnalyses is an in-memory Analyses store.
type memoryAnalyses struct {
	kept  map[uuid.UUID]json.RawMessage
	saved int
	fail  error
}

func (m *memoryAnalyses) Analysis(_ context.Context, jobID uuid.UUID) (json.RawMessage, error) {
	return m.kept[jobID], m.fail
}

func (m *memoryAnalyses) SaveAnalysis(_ context.Context, jobID uuid.UUID, analysis json.RawMessage, _ string) error {
	m.saved++
	m.kept[jobID] = analysis
	return m.fail
}

func reviewedOutput(t *testing.T) json.RawMessage {
	t.Helper()
	var answer map[string]any
	if err := json.Unmarshal(syntheticOutput(t), &answer); err != nil {
		t.Fatal(err)
	}
	answer["summary"] = []any{map[string]any{"text": "DevOps Engineer who automates CI/CD pipelines with Git and Python.", "facts": []any{"S1", "E1.B1"}}}
	answer["plan"] = map[string]any{"positioning": "The automation engineer", "evidence": []any{
		map[string]any{"theme": "Infrastructure as Code", "facts": []any{"E1.B1"}, "strength": "partial"}},
		"responsibilities": []any{map[string]any{"index": 1, "facts": []any{"E1.B1"}}}}
	answer["changes"] = []any{"Led with the pipelines"}
	raw, _ := json.Marshal(answer)
	return raw
}

func TestWriterAnalyzesOnceWritesAndReviews(t *testing.T) {
	store := &memoryAnalyses{kept: map[uuid.UUID]json.RawMessage{}}
	model := &fakeModel{replies: []modelReply{{output: analysisOutput}, {output: syntheticOutput(t)}, {output: reviewedOutput(t)}}}
	w := writer.New(model).KeepAnalyses(store)
	written, by := writeWith(t, w)
	equal(t, model.calls, 3)
	equal(t, store.saved, 1)
	equal(t, by.WrittenBy, "claude")
	equal(t, written.Tailored.Summary, "DevOps Engineer who automates CI/CD pipelines with Git and Python.")
	equal(t, written.Tailoring, &models.Tailoring{Positioning: "The automation engineer",
		Priorities: []models.TailoredPriority{{Theme: "Infrastructure as Code", Strength: "partial"}}, Close: []models.CloseSkill{},
		Changes: []string{"Led with the pipelines"}})
	equal(t, checkStartingWith(t, written.Ats.Checks, "Job responsibilities").Label, "Job responsibilities mirrored: 1 of 2")

	again := &fakeModel{replies: []modelReply{{output: syntheticOutput(t)}, {output: reviewedOutput(t)}}}
	writeWith(t, writer.New(again).KeepAnalyses(store))
	equal(t, again.calls, 2)
	equal(t, strings.Contains(again.last.Prompt, `"draft"`), true)
	equal(t, strings.Contains(again.last.System, "hiring manager"), true)
}

func TestWriterKeepsTheDraftWhenTheReviewFails(t *testing.T) {
	model := &fakeModel{replies: []modelReply{{output: analysisOutput}, {output: syntheticOutput(t)}, {err: claude.ErrFailed}}}
	written, by := writeWith(t, writer.New(model).KeepAnalyses(&memoryAnalyses{kept: map[uuid.UUID]json.RawMessage{}, fail: errors.New("down")}))
	equal(t, by.WrittenBy, "claude")
	equal(t, written.Tailored.Experience[0].Bullets[0], "Developed and maintained CI/CD pipelines for internal services with Git.")

	unreadable := &fakeModel{replies: []modelReply{{output: json.RawMessage(`[]`)}, {output: syntheticOutput(t)}, {output: json.RawMessage(`{"summary":[]}`)}}}
	_, by = writeWith(t, writer.New(unreadable))
	equal(t, by.WrittenBy, "claude")
	equal(t, unreadable.calls, 3)
}

func TestWriterStopsAtAUsageLimitWhileAnalyzing(t *testing.T) {
	until := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	model := &fakeModel{replies: []modelReply{{err: &claude.LimitError{Until: until}}}}
	_, by := writeWith(t, writer.New(model))
	equal(t, by.RulesReason, "usage_limit")
	equal(t, model.calls, 1)
}

var probeOutput = json.RawMessage(`{"questions":[{"target":"E1.B1","theme":"CI/CD","question":"How many services use your pipelines?","why":"Scale.","example":"<number>"}]}`)

func TestWriterAsksForQuestionsFromTheJobsAnalysis(t *testing.T) {
	store := &memoryAnalyses{kept: map[uuid.UUID]json.RawMessage{}}
	model := &fakeModel{replies: []modelReply{{output: analysisOutput}, {output: probeOutput}}}
	w := writer.New(model).KeepAnalyses(store)
	questions, err := w.Questions(context.Background(), catalogJob(t, 0), sampleProfile(t))
	if err != nil || len(questions) != 1 || questions[0].Place != "experience:r1" {
		t.Fatalf("questions: %+v %v", questions, err)
	}
	equal(t, strings.Contains(model.last.System, "career coach"), true)
	equal(t, store.saved, 1)
	equal(t, w.Available(), true)

	if _, err := writer.New(nil).Questions(context.Background(), catalogJob(t, 0), sampleProfile(t)); !errors.Is(err, writer.ErrNoQuestions) {
		t.Fatalf("without Claude: %v", err)
	}
	equal(t, writer.New(nil).Available(), false)
	failing := &fakeModel{replies: []modelReply{{output: analysisOutput}, {output: json.RawMessage(`[]`)}}}
	if _, err := writer.New(failing).Questions(context.Background(), catalogJob(t, 0), sampleProfile(t)); !errors.Is(err, writer.ErrNoQuestions) {
		t.Fatalf("unreadable twice: %v", err)
	}
	equal(t, failing.calls, 3)
	limited := &fakeModel{replies: []modelReply{{output: analysisOutput}, {err: &claude.LimitError{Until: time.Now().Add(time.Hour)}}}}
	if _, err := writer.New(limited).Questions(context.Background(), catalogJob(t, 0), sampleProfile(t)); !errors.Is(err, writer.ErrNoQuestions) {
		t.Fatalf("usage limit: %v", err)
	}
	unreadable := &fakeModel{replies: []modelReply{{output: json.RawMessage(`[]`)}, {output: probeOutput}}}
	if _, err := writer.New(unreadable).Questions(context.Background(), catalogJob(t, 0), sampleProfile(t)); !errors.Is(err, writer.ErrNoQuestions) || unreadable.calls != 1 {
		t.Fatalf("without the job's analysis, no questions about it: %v %d", err, unreadable.calls)
	}
	noLogin := &fakeModel{replies: []modelReply{{err: claude.ErrLoginNeeded}}}
	if _, err := writer.New(noLogin).Questions(context.Background(), catalogJob(t, 0), sampleProfile(t)); !errors.Is(err, writer.ErrNoQuestions) {
		t.Fatalf("login needed: %v", err)
	}
}

// memoryRequirements records what the writer sends back to the jobs service.
type memoryRequirements struct {
	must, nice []string
	sent       int
	fail       error
}

func (m *memoryRequirements) SetRequirements(_ context.Context, _ uuid.UUID, must, nice []string) error {
	m.sent++
	m.must, m.nice = must, nice
	return m.fail
}

func TestWriterWritesFromTheRequirementsTheAnalysisRead(t *testing.T) {
	store := &memoryAnalyses{kept: map[uuid.UUID]json.RawMessage{}}
	jobs := &memoryRequirements{}
	model := &fakeModel{replies: []modelReply{{output: analysisOutput}, {output: syntheticOutput(t)}, {output: reviewedOutput(t)}}}
	written, _ := writeWith(t, writer.New(model).KeepAnalyses(store).KeepRequirements(jobs))
	equal(t, jobs.sent, 1)
	equal(t, jobs.must, []string{"Terraform"})
	equal(t, jobs.nice, []string{"Python"})
	equal(t, checkStartingWith(t, written.Ats.Checks, "Keywords from the job").Label, "Keywords from the job: 1 of 2")

	stale := &memoryAnalyses{kept: map[uuid.UUID]json.RawMessage{}}
	for id := range store.kept {
		stale.kept[id] = json.RawMessage(`{"role":"old","priorities":[],"responsibilities":[]}`)
	}
	again := &fakeModel{replies: []modelReply{{output: analysisOutput}, {output: syntheticOutput(t)}, {output: reviewedOutput(t)}}}
	writeWith(t, writer.New(again).KeepAnalyses(stale).KeepRequirements(&memoryRequirements{fail: errors.New("down")}))
	equal(t, again.calls, 3)
	equal(t, stale.saved, 1)
}

func TestWriterAnalyzesAJobOnItsOwn(t *testing.T) {
	jobs := &memoryRequirements{}
	model := &fakeModel{replies: []modelReply{{output: analysisOutput}}}
	if err := writer.New(model).KeepRequirements(jobs).Analyze(context.Background(), catalogJob(t, 0)); err != nil {
		t.Fatal(err)
	}
	equal(t, jobs.must, []string{"Terraform"})
	if err := writer.New(nil).Analyze(context.Background(), catalogJob(t, 0)); !errors.Is(err, writer.ErrNoAnalysis) {
		t.Fatalf("without Claude: %v", err)
	}
	unreadable := &fakeModel{replies: []modelReply{{output: json.RawMessage(`[]`)}}}
	if err := writer.New(unreadable).Analyze(context.Background(), catalogJob(t, 0)); !errors.Is(err, writer.ErrNoAnalysis) {
		t.Fatalf("unreadable: %v", err)
	}
	failing := &fakeModel{replies: []modelReply{{err: claude.ErrLoginNeeded}}}
	if err := writer.New(failing).Analyze(context.Background(), catalogJob(t, 0)); !errors.Is(err, writer.ErrNoAnalysis) {
		t.Fatalf("login needed: %v", err)
	}
}

var profileProbeOutput = json.RawMessage(`{"questions":[{"target":"E1.B1","theme":"Pipelines","question":"How many services use your pipelines?",
"why":"Scale.","example":"<number>","choices":[" Under 10 services ","10–50 services","10–50 services","","Over 50 services","Over 100 services"]}]}`)

func TestWriterAsksWhatMakesEveryCVStronger(t *testing.T) {
	model := &fakeModel{replies: []modelReply{{output: profileProbeOutput}}}
	questions, err := writer.New(model).ProfileQuestions(context.Background(), sampleProfile(t))
	if err != nil || len(questions) != 1 {
		t.Fatalf("questions: %+v %v", questions, err)
	}
	equal(t, questions[0].Choices, []string{"Under 10 services", "10–50 services", "Over 50 services", "Over 100 services"})
	equal(t, questions[0].Place, "experience:r1")
	equal(t, strings.Contains(model.last.System, "every job they will apply to"), true)
	equal(t, model.calls, 1)

	if _, err := writer.New(nil).ProfileQuestions(context.Background(), sampleProfile(t)); !errors.Is(err, writer.ErrNoQuestions) {
		t.Fatalf("without Claude: %v", err)
	}
	unreadable := &fakeModel{replies: []modelReply{{output: json.RawMessage(`[]`)}, {err: claude.ErrFailed}}}
	if _, err := writer.New(unreadable).ProfileQuestions(context.Background(), sampleProfile(t)); !errors.Is(err, writer.ErrNoQuestions) {
		t.Fatalf("unreadable: %v", err)
	}
	equal(t, unreadable.calls, 2)
	noLogin := &fakeModel{replies: []modelReply{{err: claude.ErrLoginNeeded}}}
	if _, err := writer.New(noLogin).ProfileQuestions(context.Background(), sampleProfile(t)); !errors.Is(err, writer.ErrNoQuestions) {
		t.Fatalf("login needed: %v", err)
	}
}

// gatedModel holds every analysis until released and counts them; other runs answer at once.
type gatedModel struct {
	mu       sync.Mutex
	analyses int
	release  chan struct{}
}

func (g *gatedModel) Run(_ context.Context, req claude.Request) (json.RawMessage, error) {
	if req.System != grounding.AnalyzeSystem {
		return probeOutput, nil
	}
	g.mu.Lock()
	g.analyses++
	g.mu.Unlock()
	<-g.release
	return analysisOutput, nil
}

func (*gatedModel) Model() string  { return "claude-sonnet-5-5" }
func (*gatedModel) HasToken() bool { return true }

func TestWriterAnalysesAJobOnceForCallersAskingAtOnce(t *testing.T) {
	model := &gatedModel{release: make(chan struct{})}
	w := writer.New(model)
	var wg sync.WaitGroup
	results := make([]int, 3)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			questions, _ := w.Questions(context.Background(), catalogJob(t, 0), sampleProfile(t))
			results[i] = len(questions)
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(model.release)
	wg.Wait()
	equal(t, model.analyses, 1)
	equal(t, results, []int{1, 1, 1})

	held := &gatedModel{release: make(chan struct{})}
	waiting := writer.New(held)
	go func() { _, _ = waiting.Questions(context.Background(), catalogJob(t, 0), sampleProfile(t)) }()
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := waiting.Questions(ctx, catalogJob(t, 0), sampleProfile(t)); !errors.Is(err, writer.ErrNoQuestions) {
		t.Fatalf("a caller that stops waiting gets no questions: %v", err)
	}
	close(held.release)
}
