// Package writer writes a queued CV: with Claude when the service is set to
// and Claude can answer, else with the rules engine (docs/plans/2026-11-25-cv-service.md, §5–6).
package writer

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"cv/internal/claude"
	"cv/internal/cvdoc"
	"cv/internal/data/repositories"
	"cv/internal/grounding"
	"cv/internal/models"
	"cv/internal/style"
)

// Claude is the model the writer asks; *claude.Client is one.
type Claude interface {
	Run(ctx context.Context, req claude.Request) (json.RawMessage, error)
	Model() string
	HasToken() bool
}

// Attempts is how many times one CV asks Claude before the rules write it.
const Attempts = 2

// Writer writes CVs. It remembers a usage limit, so CVs queued during one go
// straight to the rules, and a dead login until a run succeeds.
type Writer struct {
	claude Claude
	now    func() time.Time

	mu           sync.Mutex
	limitedUntil time.Time
	loginNeeded  bool

	analyses repositories.Analyses
	jobs     JobRequirements

	flying  sync.Mutex
	flights map[uuid.UUID]*flight
}

// flight is an analysis running now: callers asking for the same job while it runs share its result.
type flight struct {
	done     chan struct{}
	analysis *grounding.Analysis
	stop     *stop
}

// JobRequirements keeps a job's requirements as the analysis read them; the
// upstream client writes them to the jobs service.
type JobRequirements interface {
	SetRequirements(ctx context.Context, jobID uuid.UUID, must, nice []string) error
}

// KeepRequirements writes each new analysis's requirements back to the job.
func (w *Writer) KeepRequirements(jobs JobRequirements) *Writer {
	w.jobs = jobs
	return w
}

// KeepAnalyses keeps each job's analysis in store, so it is read once per job.
func (w *Writer) KeepAnalyses(store repositories.Analyses) *Writer {
	w.analyses = store
	return w
}

// New writes with claude, or only with the rules when claude is nil.
func New(claude Claude) *Writer {
	return &Writer{claude: claude, now: time.Now}
}

// FromEnv writes with Claude when CV_WRITER is "claude-cli" (see
// claude.ConfigFromEnv), else with the rules.
func FromEnv() *Writer {
	if strings.TrimSpace(os.Getenv("CV_WRITER")) == "claude-cli" {
		return New(claude.New(claude.ConfigFromEnv()))
	}
	return New(nil)
}

// Write writes the CV of task in three steps: what the job needs (read once per
// job and kept), a plan and a draft from the person's facts, and a hiring
// manager's review of the draft. It fails only when ctx ends, so a CV is never
// finished with the rules because the service is stopping.
func (w *Writer) Write(ctx context.Context, task *repositories.Task) (models.Written, models.Authorship, error) {
	job := cvdoc.ReadJob(task.Job)
	rules := cvdoc.Write(task.Profile, job, task.Request, task.CreatedAt)
	byRules := func(reason string, until *time.Time) (models.Written, models.Authorship, error) {
		return rules, models.Authorship{WrittenBy: models.WrittenByRules, RulesReason: reason, LimitUntil: until}, ctx.Err()
	}
	if w.claude == nil {
		return byRules(models.RulesOff, nil)
	}
	if until := w.limit(); !until.IsZero() {
		return byRules(models.UsageLimit, &until)
	}
	logID := task.ID.String()
	analysis, stop := w.analyze(ctx, logID, task.Job.ID, job)
	if stop != nil {
		return byRules(stop.reason, stop.until)
	}
	// The analysis reads the ad's requirements better than the rules: the CV is written around its reading.
	job = cvdoc.ReadJob(analysis.Apply(task.Job))
	rules = cvdoc.Write(task.Profile, job, task.Request, task.CreatedAt)
	catalogue := grounding.Build(task.Profile, job, task.Request, task.CreatedAt)
	draft, ok, stop := ask(ctx, w, logID, "write", claude.Request{
		System: grounding.System, Schema: grounding.Schema(), Prompt: catalogue.Prompt(job, task.Request, catalogue.Known, analysis),
	}, grounding.ParseAnswer)
	if stop != nil {
		return byRules(stop.reason, stop.until)
	}
	if !ok {
		return byRules(models.ClaudeFailed, nil)
	}
	final := draft
	drafted, _ := catalogue.WithEquivalents(draft.Plan.Equivalents).Assemble(draft, job, rules.Tailored)
	findings := style.Lint(drafted, catalogue.Known).Findings
	if reviewed := w.review(ctx, logID, task, job, catalogue, analysis, draft, findings); reviewed != nil {
		final = *reviewed
	}

	catalogue = catalogue.WithEquivalents(final.Plan.Equivalents)
	tailored, refusals := catalogue.Assemble(final, job, rules.Tailored)
	for _, reason := range refusals {
		slog.Info("claude sentence refused", "for", logID, "reason", reason)
	}
	atsJob, atsRequest := job, task.Request
	if analysis != nil && len(analysis.Responsibilities) > 0 {
		atsJob.KeyResponsibilities = analysis.Indexes()
		atsRequest.Confirmed = append(append([]int{}, task.Request.Confirmed...), catalogue.Mirrored(final, atsJob.KeyResponsibilities)...)
	}
	written := models.Written{
		Highlights: rules.Highlights,
		Profile:    rules.Profile,
		Tailored:   tailored,
		Ats:        cvdoc.Score(cvdoc.AtsChecks(tailored, atsJob, task.Profile, atsRequest, task.CreatedAt)),
		Tailoring:  tailoring(final, catalogue.Related),
	}
	return written, models.Authorship{WrittenBy: models.WrittenByClaude, Model: w.claude.Model(), Refused: len(refusals)}, nil
}

// stop is why Claude cannot write now: the rules write the CV, for reason.
type stop struct {
	reason string
	until  *time.Time
}

// run asks Claude once. It returns the output; nil and no stop for a failure
// worth retrying; or a stop when Claude cannot write now.
func (w *Writer) run(ctx context.Context, logID, step string, request claude.Request) (json.RawMessage, *stop) {
	raw, err := w.claude.Run(ctx, request)
	var limit *claude.LimitError
	switch {
	case ctx.Err() != nil:
		return nil, &stop{reason: models.ClaudeFailed}
	case errors.Is(err, claude.ErrNoToken):
		return nil, &stop{reason: models.NoToken}
	case errors.Is(err, claude.ErrLoginNeeded):
		w.setLoginNeeded(true)
		return nil, &stop{reason: models.LoginNeeded}
	case errors.As(err, &limit):
		w.setLimit(limit.Until)
		return nil, &stop{reason: models.UsageLimit, until: &limit.Until}
	case err != nil:
		slog.Warn("claude run failed", "for", logID, "step", step, "error", err)
		return nil, nil
	}
	w.setLoginNeeded(false)
	return raw, nil
}

// ask asks Claude up to Attempts times, until read accepts an answer. ok is false when no answer was accepted; stop
// is set when Claude cannot write now.
func ask[T any](ctx context.Context, w *Writer, logID, step string, request claude.Request, read func(json.RawMessage) (T, error)) (T, bool, *stop) {
	var none T
	for attempt := 1; attempt <= Attempts; attempt++ {
		raw, halt := w.run(ctx, logID, step, request)
		if halt != nil {
			return none, false, halt
		}
		if raw == nil {
			continue
		}
		answer, err := read(raw)
		if err == nil {
			return answer, true, nil
		}
		slog.Warn("claude answer unusable", "for", logID, "step", step, "attempt", attempt, "error", err)
	}
	return none, false, nil
}

// analyze is what the job needs: kept from an earlier CV for the job, or read
// now and kept. Without one the CV is still written, from the job alone.
// analyze returns the job's analysis: the one kept, else Claude's, run once for callers asking at the same time.
func (w *Writer) analyze(ctx context.Context, logID string, jobID uuid.UUID, job cvdoc.Job) (*grounding.Analysis, *stop) {
	w.flying.Lock()
	if running, ok := w.flights[jobID]; ok {
		w.flying.Unlock()
		select {
		case <-running.done:
			return running.analysis, running.stop
		case <-ctx.Done():
			return nil, nil
		}
	}
	if w.flights == nil {
		w.flights = map[uuid.UUID]*flight{}
	}
	running := &flight{done: make(chan struct{})}
	w.flights[jobID] = running
	w.flying.Unlock()
	defer func() {
		w.flying.Lock()
		delete(w.flights, jobID)
		w.flying.Unlock()
		close(running.done)
	}()
	running.analysis, running.stop = w.analyzeOnce(ctx, logID, jobID, job)
	return running.analysis, running.stop
}

func (w *Writer) analyzeOnce(ctx context.Context, logID string, jobID uuid.UUID, job cvdoc.Job) (*grounding.Analysis, *stop) {
	if w.analyses != nil {
		if raw, err := w.analyses.Analysis(ctx, jobID); err == nil && raw != nil {
			// An analysis kept before it read requirements is read again.
			if analysis, err := grounding.ParseAnalysis(raw, job); err == nil && len(analysis.Requirements) > 0 {
				return &analysis, nil
			}
		}
	}
	raw, stop := w.run(ctx, logID, "analyze", claude.Request{
		System: grounding.AnalyzeSystem, Schema: grounding.AnalyzeSchema(), Prompt: grounding.AnalysisPrompt(job),
	})
	if stop != nil || raw == nil {
		return nil, stop
	}
	analysis, err := grounding.ParseAnalysis(raw, job)
	if err != nil {
		slog.Warn("claude analysis unreadable", "for", logID, "error", err)
		return nil, nil
	}
	if w.analyses != nil {
		if err := w.analyses.SaveAnalysis(ctx, jobID, raw, w.claude.Model()); err != nil {
			slog.Warn("job analysis not kept", "job_id", jobID, "error", err)
		}
	}
	if must, nice, ok := analysis.Split(); ok && w.jobs != nil {
		if err := w.jobs.SetRequirements(ctx, jobID, must, nice); err != nil {
			slog.Warn("job requirements not kept", "job_id", jobID, "error", err)
		}
	}
	return &analysis, nil
}

// review is the draft improved by a hiring manager's reading; nil keeps the draft.
func (w *Writer) review(ctx context.Context, logID string, task *repositories.Task, job cvdoc.Job, catalogue grounding.Catalogue,
	analysis *grounding.Analysis, draft grounding.Answer, findings []style.Finding) *grounding.Answer {
	raw, stop := w.run(ctx, logID, "review", claude.Request{
		System: grounding.ReviewSystem, Schema: grounding.ReviewSchema(),
		Prompt: catalogue.ReviewPrompt(job, task.Request, catalogue.Known, analysis, draft, findings),
	})
	if stop != nil || raw == nil {
		return nil
	}
	reviewed, err := grounding.ParseAnswer(raw)
	if err != nil || len(reviewed.Summary) == 0 {
		slog.Warn("claude review unreadable", "for", logID, "error", err)
		return nil
	}
	return &reviewed
}

// ErrNoQuestions is why Questions returned none: Claude is off, has no token,
// needs a login, is at its usage limit, or could not answer.
var ErrNoQuestions = errors.New("writer: no questions now")

// Questions asks Claude what would make the person's CV stronger for the job
// (prompts/probe-system.md), from the job's analysis and the profile's facts.
func (w *Writer) Questions(ctx context.Context, job models.Job, profile models.Profile) ([]models.ProbeQuestion, error) {
	if w.claude == nil || !w.limit().IsZero() {
		return nil, ErrNoQuestions
	}
	logID := "probe " + job.ID.String()
	analysis, stop := w.analyze(ctx, logID, job.ID, cvdoc.ReadJob(job))
	// Without the analysis the questions would not be about this job.
	if stop != nil || analysis == nil {
		return nil, ErrNoQuestions
	}
	read := cvdoc.ReadJob(analysis.Apply(job))
	catalogue := grounding.Build(profile, read, models.CvRequest{}, w.now())
	return w.probe(ctx, logID, grounding.ProbeSystem, catalogue, analysis)
}

// ProfileQuestions asks Claude what would make every CV of the person stronger
// (prompts/profile-probe-system.md): results for the facts every CV leans on.
func (w *Writer) ProfileQuestions(ctx context.Context, profile models.Profile) ([]models.ProbeQuestion, error) {
	if w.claude == nil || !w.limit().IsZero() {
		return nil, ErrNoQuestions
	}
	catalogue := grounding.Build(profile, cvdoc.Job{}, models.CvRequest{}, w.now())
	return w.probe(ctx, "profile probe", grounding.ProfileProbeSystem, catalogue, nil)
}

var errNoQuestionsAsked = errors.New("no questions in the answer")

func (w *Writer) probe(ctx context.Context, logID, system string, catalogue grounding.Catalogue, analysis *grounding.Analysis) ([]models.ProbeQuestion, error) {
	questions, ok, _ := ask(ctx, w, logID, "probe", claude.Request{
		System: system, Schema: grounding.ProbeSchema(), Prompt: catalogue.ProbePrompt(analysis),
	}, func(raw json.RawMessage) ([]models.ProbeQuestion, error) {
		questions, err := catalogue.ParseProbe(raw)
		if err == nil && len(questions) == 0 {
			err = errNoQuestionsAsked
		}
		return questions, err
	})
	if !ok {
		return nil, ErrNoQuestions
	}
	return questions, nil
}

// ErrNoAnalysis is why Analyze read nothing: Claude is off, cannot be asked now,
// or answered with nothing readable.
var ErrNoAnalysis = errors.New("writer: no analysis now")

// Analyze reads the job's ad, once per job: its analysis is kept and its
// requirements written back to the job.
func (w *Writer) Analyze(ctx context.Context, job models.Job) error {
	if w.claude == nil || !w.limit().IsZero() {
		return ErrNoAnalysis
	}
	analysis, stop := w.analyze(ctx, "analysis "+job.ID.String(), job.ID, cvdoc.ReadJob(job))
	if stop != nil || analysis == nil {
		return ErrNoAnalysis
	}
	return nil
}

// Available reports whether Claude can be asked now.
func (w *Writer) Available() bool {
	return w.claude != nil && w.claude.HasToken() && w.limit().IsZero()
}

// tailoring is what the CV page shows about how the CV was tailored.
func tailoring(answer grounding.Answer, related []cvdoc.Related) *models.Tailoring {
	priorities := []models.TailoredPriority{}
	for _, evidence := range answer.Plan.Evidence {
		priorities = append(priorities, models.TailoredPriority{Theme: evidence.Theme, Strength: evidence.Strength})
	}
	changes := answer.Changes
	if changes == nil {
		changes = []string{}
	}
	close := []models.CloseSkill{}
	for _, item := range related {
		close = append(close, models.CloseSkill{Skill: item.Skill, Via: item.Via, Category: item.Category})
	}
	return &models.Tailoring{Positioning: answer.Plan.Positioning, Priorities: priorities, Close: close, Changes: changes}
}

// Status is how the service writes CVs now.
func (w *Writer) Status() models.WriterStatus {
	if w.claude == nil {
		return models.WriterStatus{Writer: models.WrittenByRules, State: "off"}
	}
	status := models.WriterStatus{Writer: models.WrittenByClaude, Model: w.claude.Model(), State: "ok"}
	w.mu.Lock()
	defer w.mu.Unlock()
	switch {
	case !w.claude.HasToken():
		status.State = models.NoToken
	case w.now().Before(w.limitedUntil):
		until := w.limitedUntil
		status.State, status.LimitUntil = models.UsageLimit, &until
	case w.loginNeeded:
		status.State = models.LoginNeeded
	}
	return status
}

func (w *Writer) limit() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.now().Before(w.limitedUntil) {
		return w.limitedUntil
	}
	return time.Time{}
}

func (w *Writer) setLimit(until time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.limitedUntil = until
}

func (w *Writer) setLoginNeeded(dead bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.loginNeeded = dead
}

// Letter writes the cover letter and the messages to the recruiter for a ready
// CV (prompts/letter-system.md), from what the CV was written from and the CV
// itself. A letter that cannot be written now is failed, with the reason.
func (w *Writer) Letter(ctx context.Context, task *repositories.Task, cv models.Cv, recruiter string) models.Letter {
	failed := func(reason string, until *time.Time) models.Letter {
		return models.Letter{Status: models.LetterFailed, Recruiter: recruiter, Reason: reason, LimitUntil: until}
	}
	if w.claude == nil {
		return failed(models.RulesOff, nil)
	}
	if until := w.limit(); !until.IsZero() {
		return failed(models.UsageLimit, &until)
	}
	logID := "letter " + cv.ID.String()
	analysis, stop := w.analyze(ctx, logID, task.Job.ID, cvdoc.ReadJob(task.Job))
	if stop != nil {
		return failed(stop.reason, stop.until)
	}
	job := cvdoc.ReadJob(analysis.Apply(task.Job))
	catalogue := grounding.Build(task.Profile, job, task.Request, task.CreatedAt).WithJob(job)
	document := models.CvDocument{}
	if cv.Tailored != nil {
		document = *cv.Tailored
	}
	if cv.Tailoring != nil {
		catalogue = catalogue.WithEquivalents(grounding.Closes(cv.Tailoring.Close))
	}
	to := grounding.Addressing{Person: task.Profile.Name, Recruiter: recruiter, Company: job.Company, Title: job.Title}
	type assembled struct {
		texts    models.LetterTexts
		refusals []string
	}
	letter, ok, stop := ask(ctx, w, logID, "letter", claude.Request{
		System: grounding.LetterSystem, Schema: grounding.LetterSchema(), Prompt: catalogue.LetterPrompt(analysis, job, document, cv.Tailoring),
	}, func(raw json.RawMessage) (assembled, error) {
		answer, err := grounding.ParseLetter(raw)
		if err != nil {
			return assembled{}, err
		}
		texts, refusals := catalogue.AssembleLetter(answer, to)
		for _, reason := range refusals {
			slog.Info("claude sentence refused", "for", logID, "reason", reason)
		}
		if texts.CoverLetter == "" {
			return assembled{}, errLetterRefused
		}
		return assembled{texts, refusals}, nil
	})
	if stop != nil {
		return failed(stop.reason, stop.until)
	}
	if !ok {
		return failed(models.ClaudeFailed, nil)
	}
	return models.Letter{Status: models.LetterReady, Recruiter: recruiter, Texts: &letter.texts, Model: w.claude.Model(), Refused: len(letter.refusals)}
}

var errLetterRefused = errors.New("every sentence of the letter was refused")
