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
	held := map[string]bool{}
	for _, value := range number.FindAllString(strings.Join(cited, " "), -1) {
		held[value] = true
	}
	for _, value := range number.FindAllString(text, -1) {
		if !held[value] {
			return "names " + value + ", which no cited fact holds"
		}
	}
	for _, skill := range c.Missing {
		if cvdoc.HasWord(text, skill) {
			return "claims " + skill + ", which the profile does not show"
		}
	}
	for _, close := range c.Related {
		if !cvdoc.HasWord(text, close.Skill) {
			continue
		}
		named := false
		for _, via := range close.Via {
			named = named || cvdoc.HasWord(text, via)
		}
		if !named {
			return "names " + close.Skill + " without " + cvdoc.JoinList(close.Via) + ", the close equivalent the person has"
		}
	}
	if owner != "" && c.Company != "" && cvdoc.HasWord(text, c.Company) {
		return "names " + c.Company + ", the company being applied to"
	}
	return ""
}

// Assemble builds the tailored CV from Claude's answer: each sentence that
// passes Check, the rules engine's rewrite of the cited fact for each that
// does not, the rules engine's bullets for a role Claude left out, and only the
// projects Claude chose. It returns the document and why each refused sentence
// was refused.
func (c Catalogue) Assemble(answer Answer, job cvdoc.Job, rules models.CvDocument) (models.CvDocument, []string) {
	document := rules
	refusals := []string{}
	refuse := func(where, reason string) { refusals = append(refusals, where+": "+reason) }

	summary := []string{}
	for _, sentence := range answer.Summary {
		if reason := c.Check(sentence, ""); reason != "" {
			refuse("summary", reason)
			continue
		}
		summary = append(summary, strings.TrimSpace(sentence.Text))
	}
	if len(summary) > 0 {
		document.Summary = strings.Join(summary, " ")
	}

	document.Experience = append([]models.RoleSection{}, rules.Experience...)
	for i, entry := range c.Roles {
		limit := 3 + c.PageLimit - 1
		if i == 0 {
			limit = MaxFirstRoleBullets + c.PageLimit - 1
		}
		bullets, failed, ok := c.section(answer.Roles, entry, job, limit)
		for _, reason := range failed {
			refuse(entry.ID, reason)
		}
		if ok && i < len(document.Experience) {
			document.Experience[i].Bullets = bullets
		}
	}
	projects := []models.ProjectSection{}
	for i, entry := range c.Projects {
		if !chosen(answer.Projects, entry.ID) || i >= len(rules.Projects) {
			continue
		}
		project := rules.Projects[i]
		bullets, failed, ok := c.section(answer.Projects, entry, job, MaxProjectBullets)
		for _, reason := range failed {
			refuse(entry.ID, reason)
		}
		if ok {
			project.Bullets = bullets
		}
		project.Bullets = project.Bullets[:min(len(project.Bullets), MaxProjectBullets)]
		projects = append(projects, project)
	}
	document.Projects = projects
	document.Skills = c.skills(answer.Skills, rules.Skills)
	return document, refusals
}

// Limits of the tailored CV for one page; a two-page CV gets one more bullet a role.
const (
	MaxFirstRoleBullets = 5
	MaxProjectBullets   = 2
)

func chosen(sections []Section, id string) bool {
	for _, section := range sections {
		if section.ID == id {
			return true
		}
	}
	return false
}

// section is the bullets of one role or project, ok false when Claude wrote
// none for it, so the rules engine's stay.
func (c Catalogue) section(sections []Section, entry Entry, job cvdoc.Job, limit int) ([]string, []string, bool) {
	var written *Section
	for i := range sections {
		if sections[i].ID == entry.ID {
			written = &sections[i]
			break
		}
	}
	if written == nil || len(written.Bullets) == 0 {
		return nil, nil, false
	}
	bullets, refused := []string{}, []string{}
	for index, sentence := range written.Bullets {
		if len(bullets) == limit {
			break
		}
		reason := c.Check(sentence, entry.ID)
		if reason == "" {
			bullets = append(bullets, strings.TrimSpace(sentence.Text))
			continue
		}
		refused = append(refused, reason)
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

// skills is the skills Claude chose, in its order and the person's spelling,
// then any of the job's requirements the person shows that it left out, at most
// cvdoc.MaxSkills (more only to keep every matched requirement). A skill the
// person does not show is dropped. Without a usable choice the rules' list stays.
func (c Catalogue) skills(ordered, rules []string) []string {
	skills := []string{}
	for _, skill := range ordered {
		for _, have := range c.Known {
			if strings.EqualFold(strings.TrimSpace(skill), strings.TrimSpace(have)) && !contains(skills, have) {
				skills = append(skills, have)
			}
		}
	}
	if len(skills) == 0 {
		return rules
	}
	kept := []string{}
	for _, skill := range skills {
		if len(kept) < cvdoc.MaxSkills || contains(c.Matched, skill) {
			kept = append(kept, skill)
		}
	}
	for _, matched := range c.Matched {
		if !contains(kept, matched) {
			kept = append(kept, matched)
		}
	}
	return kept
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}
