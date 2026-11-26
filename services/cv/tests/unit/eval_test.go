package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"cv/internal/claude"
	"cv/internal/data/repositories"
	"cv/internal/eval"
	"cv/internal/models"
	"cv/internal/writer"
)

// fixedWriter writes the rules' CV, or fails for one case.
type fixedWriter struct{ failFor string }

func (f fixedWriter) Write(ctx context.Context, task *repositories.Task) (models.Written, models.Authorship, error) {
	if task.Profile.Name == f.failFor {
		return models.Written{}, models.Authorship{}, errors.New("stopped")
	}
	return writer.New(nil).Write(ctx, task)
}

func evalCases(t *testing.T) []eval.Case {
	t.Helper()
	cases, err := eval.LoadCases("../../eval/cases", filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	return cases
}

func TestEvalWritesLintsAndKeepsARun(t *testing.T) {
	cases := evalCases(t)
	equal(t, len(cases), 2)
	equal(t, cases[0].Name, "finance-fpa-analyst")
	equal(t, cases[0].Job.ID, uuid.NewSHA1(uuid.NameSpaceURL, []byte("hoply-eval/finance-fpa-analyst")))
	results := eval.Write(context.Background(), fixedWriter{failFor: "Ioana Marin"}, cases, now)
	equal(t, len(results), 1)
	equal(t, results[0].By.WrittenBy, "rules")
	equal(t, results[0].Style.Sentences > 0, true)

	dir := filepath.Join(t.TempDir(), "run")
	if err := eval.SaveRun(dir, results); err != nil {
		t.Fatal(err)
	}
	loaded, err := eval.LoadRun(dir)
	if err != nil || len(loaded) != 1 || loaded[0].Case != "finance-fpa-analyst" {
		t.Fatalf("loaded: %v %v", loaded, err)
	}
	if _, err := eval.LoadRun(t.TempDir()); err == nil {
		t.Fatal("an empty run fails")
	}
	table := eval.Table(results)
	equal(t, strings.Contains(table, "| finance-fpa-analyst | rules |"), true)
	equal(t, strings.Contains(table, "| **mean** |"), true)

	bad := t.TempDir()
	if err := os.WriteFile(filepath.Join(bad, "broken.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := eval.LoadCases(bad); err == nil {
		t.Fatal("a broken case fails")
	}
	if _, err := eval.LoadRun(bad); err == nil {
		t.Fatal("a broken result fails")
	}
	if _, err := eval.LoadCases("["); err == nil {
		t.Fatal("a bad pattern fails")
	}
	if err := eval.SaveRun(filepath.Join(bad, "broken.json", "x"), results); err == nil {
		t.Fatal("a run under a file fails")
	}

	local := t.TempDir()
	c := cases[0]
	c.Name = eval.Slug("Contoso Retail Group", "FP&A Analyst")
	equal(t, c.Name, "contoso-retail-group-fp-a-analyst")
	if err := eval.SaveCase(local, c); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(filepath.Join(local, c.Name+".json"))
	equal(t, info.Mode().Perm(), os.FileMode(0o600))
	if err := eval.SaveCase(filepath.Join(local, c.Name+".json"), c); err == nil {
		t.Fatal("a case under a file fails")
	}

	store := eval.NewAnalyses()
	id := uuid.New()
	if err := store.SaveAnalysis(context.Background(), id, json.RawMessage(`{}`), "m"); err != nil {
		t.Fatal(err)
	}
	if raw, _ := store.Analysis(context.Background(), id); string(raw) != "{}" {
		t.Fatalf("kept: %s", raw)
	}
}

// judgeModel answers each run with the next reply, in turn.
type judgeModel struct {
	replies []modelReply
	prompts []string
}

func (j *judgeModel) Run(_ context.Context, req claude.Request) (json.RawMessage, error) {
	j.prompts = append(j.prompts, req.Prompt)
	reply := j.replies[min(len(j.prompts)-1, len(j.replies)-1)]
	return reply.output, reply.err
}

func TestEvalJudgesAndComparesBlind(t *testing.T) {
	cases := evalCases(t)
	document := sampleDocument(t)
	scores := json.RawMessage(`{"specific":4,"human":3,"evidence":4,"relevance":5,"readable":4,"plastic":["robust"],"note":"Fewer tools."}`)
	model := &judgeModel{replies: []modelReply{{output: scores}}}
	judged, err := eval.Judge(context.Background(), model, cases[0].Job, document)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, judged.Mean, 4.0)
	equal(t, strings.Contains(model.prompts[0], `"title": "FP&A Analyst"`), true)
	equal(t, strings.Contains(model.prompts[0], "EXPERIENCE"), true)
	for _, reply := range []modelReply{{output: json.RawMessage(`[]`)}, {err: claude.ErrFailed}} {
		if _, err := eval.Judge(context.Background(), &judgeModel{replies: []modelReply{reply}}, cases[0].Job, document); err == nil {
			t.Fatalf("%v fails", reply)
		}
	}

	results := []eval.Result{{Case: cases[0].Name, Written: models.Written{Tailored: document}}, {Case: "unknown"}}
	run := eval.JudgeRun(context.Background(), &judgeModel{replies: []modelReply{{output: scores}}}, cases, results)
	equal(t, run[0].Judge.Human, 3)
	equal(t, run[1].Judge == nil, true)
	equal(t, strings.Contains(eval.Table(run), "| 4.0 | 3 |"), true)
	unjudged := eval.JudgeRun(context.Background(), &judgeModel{replies: []modelReply{{err: claude.ErrFailed}}}, cases, results)
	equal(t, unjudged[0].Judge == nil, true)

	forward := json.RawMessage(`{"specific":"a","human":"a","evidence":"b","relevance":"tie","readable":"a","winner":"a","why":"A is concrete."}`)
	backward := json.RawMessage(`{"specific":"b","human":"a","evidence":"a","relevance":"tie","readable":"b","winner":"b","why":"The second is concrete."}`)
	compare := &judgeModel{replies: []modelReply{{output: forward}, {output: backward}}}
	other := document
	other.Summary = "Another summary."
	verdict, err := eval.Compare(context.Background(), compare, cases[0].Job, document, other)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, verdict.Winner, "a")
	equal(t, verdict.Dimensions, map[string]string{"specific": "a", "human": "tie", "evidence": "b", "relevance": "tie", "readable": "a"})
	equal(t, strings.Index(compare.prompts[1], "Another summary.") < strings.Index(compare.prompts[1], `"b":`), true)
	for _, replies := range [][]modelReply{{{err: claude.ErrFailed}}, {{output: forward}, {output: json.RawMessage(`{}`)}}} {
		if _, err := eval.Compare(context.Background(), &judgeModel{replies: replies}, cases[0].Job, document, other); err == nil {
			t.Fatal("a failed comparison fails")
		}
	}

	a := []eval.Result{{Case: cases[0].Name, Written: models.Written{Tailored: document}}, {Case: cases[1].Name}, {Case: "unknown"}}
	b := []eval.Result{{Case: cases[0].Name, Written: models.Written{Tailored: other}}, {Case: cases[1].Name}}
	verdicts := eval.CompareRuns(context.Background(), &judgeModel{replies: []modelReply{{output: forward}, {output: backward}, {err: claude.ErrFailed}}}, cases, a, b, false)
	equal(t, len(verdicts), 1)
	table := eval.CompareTable([]string{cases[0].Name, cases[1].Name}, verdicts)
	equal(t, strings.Contains(table, "| finance-fpa-analyst | a | a | tie | b | tie | a |"), true)
	equal(t, strings.Contains(table, "| **total** | a 1 · b 0 · tie 0 |"), true)
	equal(t, eval.Text(models.CvDocument{Headline: "Engineer"}), "Engineer")
}

// fixedLetters writes a fixed letter, or none for one case.
type fixedLetters struct{ failFor string }

func (f fixedLetters) Letter(_ context.Context, task *repositories.Task, _ models.Cv, _ string) models.Letter {
	if task.Profile.Name == f.failFor {
		return models.Letter{Status: models.LetterFailed, Reason: models.ClaudeFailed}
	}
	return models.Letter{Status: models.LetterReady, Texts: &models.LetterTexts{
		CoverLetter:    "Dear team,\n\nI am excited to apply. Your forecast needs an owner.\n\nKind regards,\nMihai",
		ConnectionNote: "Hi, I applied for the FP&A role.",
		Message:        models.Message{Subject: "Application", Body: "Hello,\n\nI applied today.\n\nBest regards,\nMihai"},
		FollowUp:       models.Message{Subject: "Following up", Body: "Hello,\n\nAny news on the next steps?\n\nBest regards,\nMihai"},
	}}
}

func TestEvalWritesJudgesAndComparesLetters(t *testing.T) {
	cases := evalCases(t)
	results := []eval.Result{{Case: cases[0].Name, Written: models.Written{Tailored: sampleDocument(t)}}, {Case: cases[1].Name}, {Case: "unknown"}}
	written := eval.Letters(context.Background(), fixedLetters{failFor: "Ioana Marin"}, cases, results, now)
	equal(t, written[0].Letter.ConnectionNote, "Hi, I applied for the FP&A role.")
	equal(t, written[0].LetterStyle.ByRule["stock-word"], 1)
	equal(t, written[1].Letter == nil, true)
	equal(t, written[2].Letter == nil, true)
	equal(t, eval.LetterText(*written[0].Letter)["message"], "Subject: Application\n\nHello,\n\nI applied today.\n\nBest regards,\nMihai")
	equal(t, eval.LetterTable(results), "")

	scores := json.RawMessage(`{"specific":3,"human":2,"evidence":3,"relevance":4,"readable":4,"plastic":["excited to apply"],"note":"Start with them."}`)
	model := &judgeModel{replies: []modelReply{{output: scores}}}
	judged := eval.JudgeRun(context.Background(), model, cases, written)
	equal(t, judged[0].LetterJudge.Human, 2)
	equal(t, strings.Contains(model.prompts[1], `"connection_note": "Hi, I applied for the FP&A role."`), true)
	table := eval.LetterTable(judged)
	equal(t, strings.Contains(table, "| finance-fpa-analyst | 15 | 32 | 7 | 10 | 0.40 |"), true)
	equal(t, strings.Contains(table, "| 3.2 | 2 |"), true)
	failing := eval.JudgeRun(context.Background(), &judgeModel{replies: []modelReply{{output: scores}, {err: claude.ErrFailed}}}, cases, written)
	equal(t, failing[0].LetterJudge == nil, true)

	forward := json.RawMessage(`{"specific":"b","human":"b","evidence":"tie","relevance":"b","readable":"b","winner":"b","why":"B opens with them."}`)
	backward := json.RawMessage(`{"specific":"a","human":"a","evidence":"tie","relevance":"a","readable":"a","winner":"a","why":"The first opens with them."}`)
	verdicts := eval.CompareRuns(context.Background(), &judgeModel{replies: []modelReply{{output: forward}, {output: backward}}}, cases, written, written, true)
	equal(t, len(verdicts), 1)
	equal(t, verdicts[cases[0].Name].Winner, "b")
}
