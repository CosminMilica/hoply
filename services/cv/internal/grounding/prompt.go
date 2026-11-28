package grounding

import (
	_ "embed"
	"encoding/json"
	"strings"

	"cv/internal/cvdoc"
	"cv/internal/models"
	"cv/internal/style"
)

// System is the system prompt.
//
//go:embed prompts/cv-system.md
var System string

func init() {
	System = strings.Replace(System, "{{stock_words}}", cvdoc.Quoted(cvdoc.StockWords), 1)
}

//go:embed prompts/cv-output.schema.json
var schema []byte

// Schema is the answer's JSON schema on one line, as the CLI takes it.
func Schema() string { return compact(schema) }

type promptJob struct {
	Title            string          `json:"title"`
	Company          string          `json:"company"`
	Level            string          `json:"level,omitempty"`
	YearsRequired    *int            `json:"years_required,omitempty"`
	Product          string          `json:"product"`
	About            string          `json:"about"`
	AboutCompany     string          `json:"about_company,omitempty"`
	Requirements     []string        `json:"requirements"`
	Responsibilities []string        `json:"responsibilities"`
	Matched          []string        `json:"matched"`
	Related          []cvdoc.Related `json:"related"`
	Missing          []string        `json:"missing"`
	Confirmed        []string        `json:"confirmed_by_the_person"`
}

type promptMessage struct {
	Language  string          `json:"language"`
	PageLimit int             `json:"page_limit"`
	Analysis  *Analysis       `json:"analysis,omitempty"`
	Job       promptJob       `json:"job"`
	Focus     []string        `json:"focus"`
	Skills    []string        `json:"skills"`
	Roles     []Entry         `json:"roles"`
	Projects  []Entry         `json:"projects"`
	Facts     []Fact          `json:"facts"`
	Draft     *Answer         `json:"draft,omitempty"`
	Style     []style.Finding `json:"style,omitempty"`
}

// Prompt is the message: the analysis of the job (nil without one), the job (as
// data), the person's choices and the numbered facts.
func (c Catalogue) Prompt(job cvdoc.Job, request models.CvRequest, skills []string, analysis *Analysis) string {
	return c.message(job, request, skills, analysis, nil, nil)
}

// ReviewPrompt is the review's message: Prompt's, with the draft to improve and
// what reads as machine-made in it.
func (c Catalogue) ReviewPrompt(job cvdoc.Job, request models.CvRequest, skills []string, analysis *Analysis, draft Answer, findings []style.Finding) string {
	return c.message(job, request, skills, analysis, &draft, findings)
}

func (c Catalogue) message(job cvdoc.Job, request models.CvRequest, skills []string, analysis *Analysis, draft *Answer, findings []style.Finding) string {
	confirmed := []string{}
	for _, index := range request.Confirmed {
		if index >= 0 && index < len(job.Responsibilities) {
			confirmed = append(confirmed, job.Responsibilities[index])
		}
	}
	focus := request.Focus
	if len(focus) == 0 {
		focus = c.Matched[:min(3, len(c.Matched))]
	}
	message := promptMessage{
		Language:  cvdoc.LanguageOf(request.Language),
		PageLimit: c.PageLimit,
		Analysis:  analysis,
		Draft:     draft,
		Style:     findings,
		Job: promptJob{
			Title: job.Title, Company: job.Company, Level: job.Level, YearsRequired: job.YearsRequired, Product: job.Product,
			About: job.About, AboutCompany: job.AboutCompany,
			Requirements: nonNil(job.Keywords), Responsibilities: nonNil(job.Responsibilities),
			Matched: nonNil(c.Matched), Related: relatedOrEmpty(c.Related), Missing: nonNil(c.Missing), Confirmed: confirmed,
		},
		Focus:    nonNil(focus),
		Skills:   nonNil(skills),
		Roles:    nonEmpty(c.Roles),
		Projects: nonEmpty(c.Projects),
		Facts:    c.FactList(),
	}
	raw, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func nonEmpty(entries []Entry) []Entry {
	if entries == nil {
		return []Entry{}
	}
	return entries
}

func relatedOrEmpty(related []cvdoc.Related) []cvdoc.Related {
	if related == nil {
		return []cvdoc.Related{}
	}
	return related
}
