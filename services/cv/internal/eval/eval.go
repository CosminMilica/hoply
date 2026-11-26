// Package eval measures how the CV writer writes: it writes a CV for each case
// (a profile and a job), lints it for machine-made patterns, has a judge model
// score it, and compares two versions of the writer blind. It runs by hand with
// scripts/cv-eval.sh (services/cv/eval/README.md), never as a gate.
package eval

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"cv/internal/claude"
	"cv/internal/cvdoc"
	"cv/internal/data/repositories"
	"cv/internal/models"
	"cv/internal/style"
)

// Case is one profile and one job to write a CV for.
type Case struct {
	Name    string           `json:"name"`
	Profile models.Profile   `json:"profile"`
	Job     models.Job       `json:"job"`
	Request models.CvRequest `json:"request"`
}

// LoadCases reads every *.json case in the directories, by name; a directory
// that does not exist is skipped.
func LoadCases(dirs ...string) ([]Case, error) {
	cases := []Case{}
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			var c Case
			if err := readJSON(file, &c); err != nil {
				return nil, fmt.Errorf("%s: %w", file, err)
			}
			if c.Name == "" {
				c.Name = strings.TrimSuffix(filepath.Base(file), ".json")
			}
			if c.Job.ID == uuid.Nil {
				c.Job.ID = uuid.NewSHA1(uuid.NameSpaceURL, []byte("hoply-eval/"+c.Name))
			}
			cases = append(cases, c)
		}
	}
	slices.SortFunc(cases, func(a, b Case) int { return strings.Compare(a.Name, b.Name) })
	return cases, nil
}

// Result is one case's CV as a version of the writer wrote it, and its measures.
type Result struct {
	Case    string            `json:"case"`
	Written models.Written    `json:"written"`
	By      models.Authorship `json:"by"`
	Seconds float64           `json:"seconds"`
	Style   style.Report      `json:"style"`
	Judge   *Scores           `json:"judge,omitempty"`
	// The letter and messages written from this CV, when asked for (Letters).
	Letter      *models.LetterTexts `json:"letter,omitempty"`
	LetterStyle *style.Report       `json:"letter_style,omitempty"`
	LetterJudge *Scores             `json:"letter_judge,omitempty"`
}

// Writer writes a CV; *writer.Writer is one.
type Writer interface {
	Write(ctx context.Context, task *repositories.Task) (models.Written, models.Authorship, error)
}

// Write writes and lints a CV for each case, in order.
func Write(ctx context.Context, w Writer, cases []Case, now time.Time) []Result {
	results := []Result{}
	for _, c := range cases {
		started := time.Now()
		written, by, err := w.Write(ctx, &repositories.Task{ID: uuid.New(), Request: c.Request, Profile: c.Profile, Job: c.Job, CreatedAt: now})
		if err != nil {
			slog.Warn("case not written", "case", c.Name, "error", err)
			continue
		}
		results = append(results, Result{Case: c.Name, Written: written, By: by, Seconds: math.Round(time.Since(started).Seconds()),
			Style: style.Lint(written.Tailored, cvdoc.KnownSkills(c.Profile))})
	}
	return results
}

// SaveRun keeps a run's results in dir, one file per case.
func SaveRun(dir string, results []Result) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, result := range results {
		raw, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, result.Case+".json"), raw, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// LoadRun reads a run's results, by case.
func LoadRun(dir string) ([]Result, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no results in %s", dir)
	}
	results := []Result{}
	for _, file := range files {
		var result Result
		if err := readJSON(file, &result); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		results = append(results, result)
	}
	slices.SortFunc(results, func(a, b Result) int { return strings.Compare(a.Case, b.Case) })
	return results, nil
}

func readJSON(file string, target any) error {
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

// Model asks a model once; *claude.Client is one.
type Model interface {
	Run(ctx context.Context, req claude.Request) (json.RawMessage, error)
}

//go:embed prompts/judge-system.md
var judgeSystem string

//go:embed prompts/judge.schema.json
var judgeSchema []byte

//go:embed prompts/compare-system.md
var compareSystem string

//go:embed prompts/compare.schema.json
var compareSchema []byte

//go:embed prompts/letter-judge-system.md
var letterJudgeSystem string

//go:embed prompts/letter-compare-system.md
var letterCompareSystem string

func compact(raw []byte) string {
	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		panic(err)
	}
	return out.String()
}

// Scores are the judge's marks for one CV's writing (prompts/judge-system.md).
type Scores struct {
	Specific  int      `json:"specific"`
	Human     int      `json:"human"`
	Evidence  int      `json:"evidence"`
	Relevance int      `json:"relevance"`
	Readable  int      `json:"readable"`
	Mean      float64  `json:"mean"`
	Plastic   []string `json:"plastic"`
	Note      string   `json:"note"`
}

type adView struct {
	Title            string   `json:"title"`
	Company          string   `json:"company"`
	About            string   `json:"about"`
	Responsibilities []string `json:"responsibilities"`
	MustHave         []string `json:"must_have"`
	NiceToHave       []string `json:"nice_to_have"`
}

func ad(job models.Job) adView {
	return adView{Title: job.Title, Company: job.Company, About: job.About, Responsibilities: job.Responsibilities,
		MustHave: job.MustHave, NiceToHave: job.NiceToHave}
}

func message(value any) string {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		panic(err)
	}
	return out.String()
}

// ErrUnreadable is a judge's answer that does not fit its schema.
var ErrUnreadable = errors.New("eval: unreadable answer")

// Judge scores one CV's writing for the job.
func Judge(ctx context.Context, m Model, job models.Job, document models.CvDocument) (*Scores, error) {
	return score(ctx, m, judgeSystem, map[string]any{"job": ad(job), "cv": Text(document)})
}

// JudgeLetter scores the writing of a letter and its messages for the job.
func JudgeLetter(ctx context.Context, m Model, job models.Job, texts models.LetterTexts) (*Scores, error) {
	return score(ctx, m, letterJudgeSystem, map[string]any{"job": ad(job), "texts": LetterText(texts)})
}

func score(ctx context.Context, m Model, system string, prompt any) (*Scores, error) {
	raw, err := m.Run(ctx, claude.Request{System: system, Schema: compact(judgeSchema), Prompt: message(prompt)})
	if err != nil {
		return nil, err
	}
	var scores Scores
	if err := json.Unmarshal(raw, &scores); err != nil || scores.Specific == 0 {
		return nil, ErrUnreadable
	}
	scores.Mean = math.Round(float64(scores.Specific+scores.Human+scores.Evidence+scores.Relevance+scores.Readable)/5*100) / 100
	return &scores, nil
}

// Dimensions are what the judge scores, in order.
var Dimensions = []string{"specific", "human", "evidence", "relevance", "readable"}

// Verdict is which of two CVs the judge prefers, overall and on each dimension.
type Verdict struct {
	Winner     string            `json:"winner"`
	Dimensions map[string]string `json:"dimensions"`
	Why        []string          `json:"why"`
}

type comparison struct {
	Specific  string `json:"specific"`
	Human     string `json:"human"`
	Evidence  string `json:"evidence"`
	Relevance string `json:"relevance"`
	Readable  string `json:"readable"`
	Winner    string `json:"winner"`
	Why       string `json:"why"`
}

func (c comparison) pick(dimension string) string {
	return map[string]string{"specific": c.Specific, "human": c.Human, "evidence": c.Evidence, "relevance": c.Relevance, "readable": c.Readable}[dimension]
}

// Compare asks the judge twice, with the CVs in both orders, so the place a CV
// is shown in cannot decide; a pick the two orders disagree on is a tie.
func Compare(ctx context.Context, m Model, job models.Job, a, b models.CvDocument) (Verdict, error) {
	return compare(ctx, m, compareSystem, job, Text(a), Text(b))
}

// CompareLetters is Compare for two versions of a letter and its messages.
func CompareLetters(ctx context.Context, m Model, job models.Job, a, b models.LetterTexts) (Verdict, error) {
	return compare(ctx, m, letterCompareSystem, job, LetterText(a), LetterText(b))
}

func compare(ctx context.Context, m Model, system string, job models.Job, a, b any) (Verdict, error) {
	ask := func(first, second any) (comparison, error) {
		raw, err := m.Run(ctx, claude.Request{System: system, Schema: compact(compareSchema),
			Prompt: message(map[string]any{"job": ad(job), "a": first, "b": second})})
		if err != nil {
			return comparison{}, err
		}
		var answer comparison
		if err := json.Unmarshal(raw, &answer); err != nil || answer.Winner == "" {
			return comparison{}, ErrUnreadable
		}
		return answer, nil
	}
	forward, err := ask(a, b)
	if err != nil {
		return Verdict{}, err
	}
	backward, err := ask(b, a)
	if err != nil {
		return Verdict{}, err
	}
	swap := map[string]string{"a": "b", "b": "a", "tie": "tie"}
	agree := func(first, second string) string {
		if first == swap[second] {
			return first
		}
		return "tie"
	}
	verdict := Verdict{Winner: agree(forward.Winner, backward.Winner), Dimensions: map[string]string{}, Why: []string{forward.Why, backward.Why}}
	for _, dimension := range Dimensions {
		verdict.Dimensions[dimension] = agree(forward.pick(dimension), backward.pick(dimension))
	}
	return verdict, nil
}

// Text is the CV as a reader sees it, without layout.
func Text(d models.CvDocument) string {
	var b strings.Builder
	line := func(text string) {
		if strings.TrimSpace(text) != "" {
			b.WriteString(text + "\n")
		}
	}
	section := func(title string) { b.WriteString("\n" + strings.ToUpper(title) + "\n") }
	line(d.Name)
	line(d.Headline)
	if d.Summary != "" {
		section("Summary")
		line(d.Summary)
	}
	if len(d.Skills) > 0 {
		section("Skills")
		line(strings.Join(d.Skills, ", "))
	}
	if len(d.Experience) > 0 {
		section("Experience")
		for _, role := range d.Experience {
			line(role.Role + " | " + role.Meta)
			for _, bullet := range role.Bullets {
				line("- " + bullet)
			}
		}
	}
	if len(d.Projects) > 0 {
		section("Projects")
		for _, project := range d.Projects {
			line(strings.Trim(project.Name+" | "+project.Meta, " |"))
			for _, bullet := range project.Bullets {
				line("- " + bullet)
			}
		}
	}
	if len(d.Education) > 0 {
		section("Education")
		for _, item := range d.Education {
			line(strings.Trim(item.Degree+" | "+item.Meta, " |"))
		}
	}
	return strings.TrimSpace(b.String())
}

func byName(cases []Case) map[string]Case {
	found := map[string]Case{}
	for _, c := range cases {
		found[c.Name] = c
	}
	return found
}

// JudgeRun has the judge score each result whose case is known; a result the
// judge cannot score keeps no scores.
func JudgeRun(ctx context.Context, m Model, cases []Case, results []Result) []Result {
	known := byName(cases)
	judged := slices.Clone(results)
	for i, result := range judged {
		c, ok := known[result.Case]
		if !ok {
			continue
		}
		if scores, err := Judge(ctx, m, c.Job, result.Written.Tailored); err == nil {
			judged[i].Judge = scores
		} else {
			slog.Warn("case not judged", "case", result.Case, "error", err)
		}
		if result.Letter == nil {
			continue
		}
		if scores, err := JudgeLetter(ctx, m, c.Job, *result.Letter); err == nil {
			judged[i].LetterJudge = scores
		} else {
			slog.Warn("letter not judged", "case", result.Case, "error", err)
		}
	}
	return judged
}

// CompareRuns compares, blind, the CVs two runs wrote for each case both hold;
// with letters, their letters and messages instead.
func CompareRuns(ctx context.Context, m Model, cases []Case, a, b []Result, letters bool) map[string]Verdict {
	known := byName(cases)
	other := map[string]Result{}
	for _, result := range b {
		other[result.Case] = result
	}
	verdicts := map[string]Verdict{}
	for _, first := range a {
		second, ok := other[first.Case]
		c, known := known[first.Case]
		if !ok || !known {
			continue
		}
		var (
			verdict Verdict
			err     error
		)
		switch {
		case !letters:
			verdict, err = Compare(ctx, m, c.Job, first.Written.Tailored, second.Written.Tailored)
		case first.Letter != nil && second.Letter != nil:
			verdict, err = CompareLetters(ctx, m, c.Job, *first.Letter, *second.Letter)
		default:
			continue
		}
		if err != nil {
			slog.Warn("case not compared", "case", first.Case, "error", err)
			continue
		}
		verdicts[first.Case] = verdict
	}
	return verdicts
}

// Slug is a name as a file name: lower case, words joined by "-".
func Slug(parts ...string) string {
	fields := strings.FieldsFunc(strings.ToLower(strings.Join(parts, " ")), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	return strings.Join(fields[:min(6, len(fields))], "-")
}

// SaveCase writes a case to dir as <name>.json, readable only by its owner: a
// case holds a real profile.
func SaveCase(dir string, c Case) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, c.Name+".json"), raw, 0o600)
}

// Analyses keeps each job's analysis for one run, so a case's job is read once.
type Analyses struct {
	mu   sync.Mutex
	kept map[uuid.UUID]json.RawMessage
}

// NewAnalyses is an empty store.
func NewAnalyses() *Analyses { return &Analyses{kept: map[uuid.UUID]json.RawMessage{}} }

func (a *Analyses) Analysis(_ context.Context, jobID uuid.UUID) (json.RawMessage, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.kept[jobID], nil
}

func (a *Analyses) SaveAnalysis(_ context.Context, jobID uuid.UUID, analysis json.RawMessage, _ string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.kept[jobID] = analysis
	return nil
}

// LetterWriter writes the letter and messages for a ready CV; *writer.Writer is one.
type LetterWriter interface {
	Letter(ctx context.Context, task *repositories.Task, cv models.Cv, recruiter string) models.Letter
}

// Letters writes and lints the letter and messages for each result's CV.
func Letters(ctx context.Context, w LetterWriter, cases []Case, results []Result, now time.Time) []Result {
	known := byName(cases)
	written := slices.Clone(results)
	for i, result := range written {
		c, ok := known[result.Case]
		if !ok {
			continue
		}
		tailored := result.Written.Tailored
		cv := models.Cv{ID: uuid.New(), Status: models.StatusReady, Tailored: &tailored, Tailoring: result.Written.Tailoring}
		letter := w.Letter(ctx, &repositories.Task{ID: cv.ID, Request: c.Request, Profile: c.Profile, Job: c.Job, CreatedAt: now}, cv, "")
		if letter.Texts == nil {
			slog.Warn("letter not written", "case", result.Case, "reason", letter.Reason)
			continue
		}
		report := style.LintLetter(*letter.Texts, cvdoc.KnownSkills(c.Profile))
		written[i].Letter, written[i].LetterStyle, written[i].LetterJudge = letter.Texts, &report, nil
	}
	return written
}

// LetterText is a letter and its messages as the reader gets them.
func LetterText(t models.LetterTexts) map[string]string {
	return map[string]string{
		"cover_letter":    t.CoverLetter,
		"connection_note": t.ConnectionNote,
		"message":         strings.TrimSpace("Subject: " + t.Message.Subject + "\n\n" + t.Message.Body),
		"follow_up":       strings.TrimSpace("Subject: " + t.FollowUp.Subject + "\n\n" + t.FollowUp.Body),
	}
}
