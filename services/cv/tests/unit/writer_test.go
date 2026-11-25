package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"cv/internal/claude"
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
	equal(t, failing.calls, writer.Attempts)

	unreadable := &fakeModel{replies: []modelReply{{output: json.RawMessage(`[]`)}}}
	_, by = writeWith(t, writer.New(unreadable))
	equal(t, by.RulesReason, "claude_failed")

	recovering := &fakeModel{replies: []modelReply{{err: claude.ErrFailed}, {output: syntheticOutput(t)}}}
	_, by = writeWith(t, writer.New(recovering))
	equal(t, by.WrittenBy, "claude")
	equal(t, recovering.calls, 2)
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
