package grounding

import (
	"encoding/json"
	"regexp"
	"strings"

	"cv/internal/cvdoc"
	"cv/internal/models"
)

// MaxSentence is the longest sentence a CV takes, as the profile's highlights.
const MaxSentence = 300

// Sentence is one sentence Claude wrote, with the facts it rests on.
type Sentence struct {
	Text  string   `json:"text"`
	Facts []string `json:"facts"`
}

// Section is what Claude wrote for one role or project.
type Section struct {
	ID      string     `json:"id"`
	Bullets []Sentence `json:"bullets"`
}

// Answer is Claude's answer, as prompts/cv-output.schema.json describes it.
type Answer struct {
	Summary  []Sentence `json:"summary"`
	Roles    []Section  `json:"roles"`
	Projects []Section  `json:"projects"`
	Skills   []string   `json:"skills"`
}

// ParseAnswer reads Claude's structured output.
func ParseAnswer(raw json.RawMessage) (Answer, error) {
	var answer Answer
	err := json.Unmarshal(raw, &answer)
	return answer, err
}

var number = regexp.MustCompile(`\b\d+(?:[.,]\d+)*`)

// Check reports why a sentence cannot go in the CV, or "" when it can. owner is
// the role or project the sentence describes, "" for the summary.
func (c Catalogue) Check(sentence Sentence, owner string) string {
	text := strings.TrimSpace(sentence.Text)
	switch {
	case text == "":
		return "empty"
	case len(text) > MaxSentence:
		return "too long"
	case len(sentence.Facts) == 0:
		return "cites no fact"
	}
	cited := []string{}
	for _, id := range sentence.Facts {
		fact, ok := c.Facts[id]
		if !ok {
			return "cites an unknown fact " + id
		}
		if owner != "" && fact.Owner != owner {
			return "cites " + id + ", which is not about " + owner
		}
		cited = append(cited, fact.Text)
	}
	source := strings.Join(cited, " ")
	for _, value := range number.FindAllString(text, -1) {
		if !strings.Contains(source, value) {
			return "names " + value + ", which no cited fact holds"
		}
	}
	for _, skill := range c.Missing {
		if cvdoc.HasWord(text, skill) {
			return "claims " + skill + ", which the profile does not show"
		}
	}
	if owner != "" && c.Company != "" && cvdoc.HasWord(text, c.Company) {
		return "names " + c.Company + ", the company being applied to"
	}
	return ""
}

// Assemble builds the tailored CV from Claude's answer: each sentence that
// passes Check, the rules engine's rewrite of the cited fact for each that
// does not, and the rules engine's version of anything Claude left out. It
// returns the document and how many sentences it refused.
func (c Catalogue) Assemble(answer Answer, job cvdoc.Job, rules models.CvDocument) (models.CvDocument, int) {
	document := rules
	refused := 0

	summary := []string{}
	for _, sentence := range answer.Summary {
		if c.Check(sentence, "") != "" {
			refused++
			continue
		}
		summary = append(summary, strings.TrimSpace(sentence.Text))
	}
	if len(summary) > 0 {
		document.Summary = strings.Join(summary, " ")
	}

	document.Experience = append([]models.RoleSection{}, rules.Experience...)
	for i, entry := range c.Roles {
		limit := 3
		if i == 0 {
			limit = 7
		}
		bullets, failed, ok := c.section(answer.Roles, entry, job, limit)
		refused += failed
		if ok && i < len(document.Experience) {
			document.Experience[i].Bullets = bullets
		}
	}
	document.Projects = append([]models.ProjectSection{}, rules.Projects...)
	for i, entry := range c.Projects {
		bullets, failed, ok := c.section(answer.Projects, entry, job, 3)
		refused += failed
		if ok && i < len(document.Projects) {
			document.Projects[i].Bullets = bullets
		}
	}
	document.Skills = c.skills(answer.Skills, rules.Skills)
	return document, refused
}

// section is the bullets of one role or project, ok false when Claude wrote
// none for it, so the rules engine's stay.
func (c Catalogue) section(sections []Section, entry Entry, job cvdoc.Job, limit int) ([]string, int, bool) {
	var written *Section
	for i := range sections {
		if sections[i].ID == entry.ID {
			written = &sections[i]
			break
		}
	}
	if written == nil || len(written.Bullets) == 0 {
		return nil, 0, false
	}
	bullets, refused := []string{}, 0
	for index, sentence := range written.Bullets {
		if len(bullets) == limit {
			break
		}
		if c.Check(sentence, entry.ID) == "" {
			bullets = append(bullets, strings.TrimSpace(sentence.Text))
			continue
		}
		refused++
		if fallback := c.rewrite(sentence, entry.ID, job, index); fallback != "" && !contains(bullets, fallback) {
			bullets = append(bullets, fallback)
		}
	}
	if len(bullets) == 0 {
		return nil, refused, false
	}
	return bullets, refused, true
}

// rewrite is the rules engine's version of the first fact a refused sentence
// cites, when that fact belongs to the entry.
func (c Catalogue) rewrite(sentence Sentence, owner string, job cvdoc.Job, index int) string {
	for _, id := range sentence.Facts {
		if fact, ok := c.Facts[id]; ok && fact.Owner == owner {
			return cvdoc.RewriteBullet(cvdoc.CleanFact(fact.Text, ""), index, cvdoc.RewriteContext{Job: job, Matched: c.Matched, Tools: fact.Tools})
		}
	}
	return ""
}

// skills is Claude's order of the person's skills, then the rest in the rules
// engine's order; a skill the person does not show is dropped.
func (c Catalogue) skills(ordered, rules []string) []string {
	skills := []string{}
	for _, skill := range ordered {
		for _, have := range rules {
			if strings.EqualFold(strings.TrimSpace(skill), have) && !contains(skills, have) {
				skills = append(skills, have)
			}
		}
	}
	for _, have := range rules {
		if !contains(skills, have) {
			skills = append(skills, have)
		}
	}
	return skills
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}
