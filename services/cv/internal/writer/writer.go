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

// Write writes the CV of task. It fails only when ctx ends, so a CV is never
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
	request := claude.Request{
		System: grounding.System,
		Schema: grounding.Schema(),
		Prompt: catalogue.Prompt(job, task.Request, catalogue.Known),
	}
	for attempt := 1; attempt <= Attempts; attempt++ {
		raw, err := w.claude.Run(ctx, request)
		var limit *claude.LimitError
		switch {
		case ctx.Err() != nil:
			return byRules(models.ClaudeFailed, nil)
		case errors.Is(err, claude.ErrNoToken):
			return byRules(models.NoToken, nil)
		case errors.Is(err, claude.ErrLoginNeeded):
			w.setLoginNeeded(true)
			return byRules(models.LoginNeeded, nil)
		case errors.As(err, &limit):
			w.setLimit(limit.Until)
			return byRules(models.UsageLimit, &limit.Until)
		case err != nil:
			slog.Warn("claude run failed", "cv_id", task.ID, "attempt", attempt, "error", err)
			continue
		}
		answer, err := grounding.ParseAnswer(raw)
		if err != nil {
			slog.Warn("claude answer unreadable", "cv_id", task.ID, "attempt", attempt, "error", err)
			continue
		}
		w.setLoginNeeded(false)
		tailored, refusals := catalogue.Assemble(answer, job, rules.Tailored)
		for _, reason := range refusals {
			slog.Info("claude sentence refused", "cv_id", task.ID, "reason", reason)
		}
		written := models.Written{
			Highlights: rules.Highlights,
			Profile:    rules.Profile,
			Tailored:   tailored,
			Ats:        cvdoc.Score(cvdoc.AtsChecks(tailored, job, task.Profile, task.Request, task.CreatedAt)),
		}
		return written, models.Authorship{WrittenBy: models.WrittenByClaude, Model: w.claude.Model(), Refused: len(refusals)}, nil
	}
	return byRules(models.ClaudeFailed, nil)
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
