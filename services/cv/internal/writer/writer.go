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
	catalogue := grounding.Build(task.Profile, job, task.Request, task.CreatedAt)

	logID := task.ID.String()
	analysis, stop := w.analyze(ctx, logID, task.Job.ID, job)
	if stop != nil {
		return byRules(stop.reason, stop.until)
	}
	var draft *grounding.Answer
	for attempt := 1; attempt <= Attempts && draft == nil; attempt++ {
		raw, stop := w.run(ctx, logID, "write", claude.Request{
			System: grounding.System, Schema: grounding.Schema(), Prompt: catalogue.Prompt(job, task.Request, catalogue.Known, analysis),
		})
		if stop != nil {
			return byRules(stop.reason, stop.until)
		}
		if raw == nil {
			continue
		}
		answer, err := grounding.ParseAnswer(raw)
		if err != nil {
			slog.Warn("claude answer unreadable", "cv_id", task.ID, "step", "write", "attempt", attempt, "error", err)
			continue
		}
		draft = &answer
	}
	if draft == nil {
		return byRules(models.ClaudeFailed, nil)
	}
	final := *draft
	if reviewed := w.review(ctx, task, job, catalogue, analysis, *draft); reviewed != nil {
		final = *reviewed
	}

	catalogue = catalogue.WithEquivalents(final.Plan.Equivalents)
	tailored, refusals := catalogue.Assemble(final, job, rules.Tailored)
	for _, reason := range refusals {
		slog.Info("claude sentence refused", "cv_id", task.ID, "reason", reason)
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

// analyze is what the job needs: kept from an earlier CV for the job, or read
// now and kept. Without one the CV is still written, from the job alone.
func (w *Writer) analyze(ctx context.Context, logID string, jobID uuid.UUID, job cvdoc.Job) (*grounding.Analysis, *stop) {
	if w.analyses != nil {
		if raw, err := w.analyses.Analysis(ctx, jobID); err == nil && raw != nil {
			if analysis, err := grounding.ParseAnalysis(raw, job); err == nil {
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
	return &analysis, nil
}

// review is the draft improved by a hiring manager's reading; nil keeps the draft.
func (w *Writer) review(ctx context.Context, task *repositories.Task, job cvdoc.Job, catalogue grounding.Catalogue,
	analysis *grounding.Analysis, draft grounding.Answer) *grounding.Answer {
	raw, stop := w.run(ctx, task.ID.String(), "review", claude.Request{
		System: grounding.ReviewSystem, Schema: grounding.ReviewSchema(),
		Prompt: catalogue.ReviewPrompt(job, task.Request, catalogue.Known, analysis, draft),
	})
	if stop != nil || raw == nil {
		return nil
	}
	reviewed, err := grounding.ParseAnswer(raw)
	if err != nil || len(reviewed.Summary) == 0 {
		slog.Warn("claude review unreadable", "cv_id", task.ID, "error", err)
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
	read := cvdoc.ReadJob(job)
	logID := "probe " + job.ID.String()
	analysis, stop := w.analyze(ctx, logID, job.ID, read)
	if stop != nil {
		return nil, ErrNoQuestions
	}
	catalogue := grounding.Build(profile, read, models.CvRequest{}, w.now())
	for attempt := 1; attempt <= Attempts; attempt++ {
		raw, stop := w.run(ctx, logID, "probe", claude.Request{
			System: grounding.ProbeSystem, Schema: grounding.ProbeSchema(), Prompt: catalogue.ProbePrompt(analysis),
		})
		if stop != nil {
			return nil, ErrNoQuestions
		}
		if raw == nil {
			continue
		}
		questions, err := catalogue.ParseProbe(raw)
		if err == nil && len(questions) > 0 {
			return questions, nil
		}
		slog.Warn("claude questions unreadable", "for", logID, "error", err)
	}
	return nil, ErrNoQuestions
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
