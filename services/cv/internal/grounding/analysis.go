package grounding

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strings"

	"cv/internal/cvdoc"
	"cv/internal/models"
)

// AnalyzeSystem is the system prompt of the job analysis.
//
//go:embed prompts/analyze-system.md
var AnalyzeSystem string

//go:embed prompts/analyze.schema.json
var analyzeSchema []byte

// ReviewSystem is the system prompt of the review.
//
//go:embed prompts/review-system.md
var ReviewSystem string

//go:embed prompts/review-output.schema.json
var reviewSchema []byte

func compact(raw []byte) string {
	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		panic(err)
	}
	return out.String()
}

// AnalyzeSchema is the analysis's JSON schema on one line.
func AnalyzeSchema() string { return compact(analyzeSchema) }

// ReviewSchema is the review's JSON schema on one line.
func ReviewSchema() string { return compact(reviewSchema) }

// Priority is a theme the hire will be judged on.
type Priority struct {
	Theme   string   `json:"theme"`
	AdWords []string `json:"ad_words"`
	Why     string   `json:"why"`
}

// Duty is a real responsibility of the job: an index into its lines.
type Duty struct {
	Index int    `json:"index"`
	Theme string `json:"theme"`
}

// Requirement is a skill the ad asks for, as Claude reads it.
type Requirement struct {
	Name       string `json:"name"`
	Importance string `json:"importance"`
}

// Analysis is what a job ad says the company needs (prompts/analyze-system.md).
type Analysis struct {
	Requirements     []Requirement `json:"requirements"`
	Role             string        `json:"role"`
	Context          string        `json:"context"`
	Priorities       []Priority    `json:"priorities"`
	Responsibilities []Duty        `json:"responsibilities"`
	Traits           []string      `json:"traits"`
	Terms            []string      `json:"terms"`
	Seniority        string        `json:"seniority"`
}

type analysisLine struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type analysisMessage struct {
	Title         string         `json:"title"`
	Company       string         `json:"company"`
	Level         string         `json:"level,omitempty"`
	YearsRequired *int           `json:"years_required,omitempty"`
	About         string         `json:"about"`
	AboutCompany  string         `json:"about_company,omitempty"`
	Product       string         `json:"product,omitempty"`
	Lines         []analysisLine `json:"lines"`
	// Qualifications and NiceToHave are the ad's own lines; Requirements are the skills read from it.
	Qualifications []string `json:"qualifications,omitempty"`
	NiceToHave     []string `json:"nice_to_have,omitempty"`
	Requirements   []string `json:"requirements"`
}

// AnalysisPrompt is the job ad as the analysis reads it.
func AnalysisPrompt(job cvdoc.Job) string {
	lines := make([]analysisLine, 0, len(job.Responsibilities))
	for i, line := range job.Responsibilities {
		lines = append(lines, analysisLine{Index: i, Text: line})
	}
	raw, err := json.MarshalIndent(analysisMessage{
		Title: job.Title, Company: job.Company, Level: job.Level, YearsRequired: job.YearsRequired, About: job.About,
		AboutCompany: job.AboutCompany, Product: job.Product, Lines: lines, Qualifications: job.Qualifications,
		NiceToHave: job.Bonuses, Requirements: nonNil(job.Keywords),
	}, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// ParseAnalysis reads the analysis and drops responsibilities that point past
// the job's lines.
func ParseAnalysis(raw json.RawMessage, job cvdoc.Job) (Analysis, error) {
	var analysis Analysis
	if err := json.Unmarshal(raw, &analysis); err != nil {
		return analysis, err
	}
	duties := []Duty{}
	seen := map[int]bool{}
	for _, duty := range analysis.Responsibilities {
		if duty.Index >= 0 && duty.Index < len(job.Responsibilities) && !seen[duty.Index] {
			seen[duty.Index] = true
			duties = append(duties, duty)
		}
	}
	analysis.Responsibilities = duties
	return analysis, nil
}

// Indexes are the job's real responsibilities, most important first.
func (a Analysis) Indexes() []int {
	indexes := make([]int, 0, len(a.Responsibilities))
	for _, duty := range a.Responsibilities {
		indexes = append(indexes, duty.Index)
	}
	return indexes
}

// Split is the analysis's requirements as must-haves and nice-to-haves, each
// named once; ok is false when the analysis read none.
func (a Analysis) Split() (must, nice []string, ok bool) {
	must, nice = []string{}, []string{}
	seen := map[string]bool{}
	for _, requirement := range a.Requirements {
		name := strings.TrimSpace(requirement.Name)
		key := strings.ToLower(name)
		if name == "" || seen[key] {
			continue
		}
		seen[key] = true
		if requirement.Importance == "nice" {
			nice = append(nice, name)
		} else {
			must = append(must, name)
		}
	}
	return must, nice, len(must)+len(nice) > 0
}

// Apply is the job with the analysis's requirements in place of the rules' ones.
func (a *Analysis) Apply(job models.Job) models.Job {
	if a == nil {
		return job
	}
	if must, nice, ok := a.Split(); ok {
		job.MustHave, job.NiceToHave = must, nice
	}
	return job
}
