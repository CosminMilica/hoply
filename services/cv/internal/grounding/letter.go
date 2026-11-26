package grounding

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"cv/internal/cvdoc"
	"cv/internal/models"
)

// LetterSystem is the system prompt of the cover letter and the messages.
//
//go:embed prompts/letter-system.md
var LetterSystem string

//go:embed prompts/letter.schema.json
var letterSchema []byte

// LetterSchema is the letter's JSON schema on one line.
func LetterSchema() string { return compact(letterSchema) }

// MaxConnectionNote is LinkedIn's limit for the note of a connection request.
const MaxConnectionNote = 300

// cliches are what recruiters skip and what reads as written by a machine.
var cliches = []string{"passionate", "team player", "hard-working", "hardworking", "results-driven", "detail-oriented",
	"perfect fit", "perfect candidate", "dream job", "i am writing to", "i'm writing to", "i believe i would be",
	"to whom it may concern", "just checking in", "look forward to hearing", "please find attached",
	"would love the opportunity", "excited to apply", "exceptional addition", "uniquely positions", "hit the ground running",
	"one more reason i fit", "!"}

// WithJob is the catalogue with what the ad says about the role and the company
// as facts J1, J2…, which the letter's sentences about the company cite.
func (c Catalogue) WithJob(job cvdoc.Job) Catalogue {
	extended := c
	extended.JobTitle = job.Title
	extended.Facts = maps.Clone(c.Facts)
	extended.Order = slices.Clone(c.Order)
	n := 0
	add := func(text string) {
		if text = strings.TrimSpace(text); text == "" {
			return
		}
		n++
		id := fmt.Sprintf("J%d", n)
		extended.Facts[id] = Fact{ID: id, Text: text, Owner: "job"}
		extended.Order = append(extended.Order, id)
	}
	add(strings.Join(keep(job.Title, job.Company), " at "))
	add(job.About)
	add(job.AboutCompany)
	add(job.Product)
	for _, line := range job.Responsibilities[:min(6, len(job.Responsibilities))] {
		add(line)
	}
	return extended
}

// Closes are the CV's close skills as equivalents, so the letter keeps naming
// them the way the CV does.
func Closes(close []models.CloseSkill) []Equivalent {
	equivalents := []Equivalent{}
	for _, skill := range close {
		equivalents = append(equivalents, Equivalent{Skill: skill.Skill, Via: skill.Via, Category: skill.Category})
	}
	return equivalents
}

type letterJob struct {
	Title   string          `json:"title"`
	Company string          `json:"company"`
	Matched []string        `json:"matched"`
	Related []cvdoc.Related `json:"related"`
	Missing []string        `json:"missing"`
}

type letterMessage struct {
	Analysis    *Analysis                 `json:"analysis,omitempty"`
	Job         letterJob                 `json:"job"`
	Positioning string                    `json:"positioning"`
	Priorities  []models.TailoredPriority `json:"priorities"`
	CV          models.CvDocument         `json:"cv"`
	Roles       []Entry                   `json:"roles"`
	Projects    []Entry                   `json:"projects"`
	Facts       []Fact                    `json:"facts"`
}

// LetterPrompt is the letter's message: the job's analysis, the CV as it will
// be sent and how it was tailored, and the numbered facts with the job's.
func (c Catalogue) LetterPrompt(analysis *Analysis, job cvdoc.Job, cv models.CvDocument, tailoring *models.Tailoring) string {
	message := letterMessage{Analysis: analysis, CV: cv, Roles: nonEmpty(c.Roles), Projects: nonEmpty(c.Projects), Facts: c.FactList(),
		Job:        letterJob{Title: job.Title, Company: job.Company, Matched: nonNil(c.Matched), Related: c.Related, Missing: nonNil(c.Missing)},
		Priorities: []models.TailoredPriority{}}
	if message.Job.Related == nil {
		message.Job.Related = []cvdoc.Related{}
	}
	if tailoring != nil {
		message.Positioning = tailoring.Positioning
		message.Priorities = tailoring.Priorities
	}
	raw, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// LetterMessage is a message's subject and sentences, as Claude writes them.
type LetterMessage struct {
	Subject   string     `json:"subject"`
	Sentences []Sentence `json:"sentences"`
}

// LetterAnswer is Claude's answer for the letter (prompts/letter.schema.json).
type LetterAnswer struct {
	Plan struct {
		Hook   string `json:"hook"`
		Proofs []struct {
			Priority string   `json:"priority"`
			Facts    []string `json:"facts"`
		} `json:"proofs"`
		Why string `json:"why"`
	} `json:"plan"`
	Letter []struct {
		Sentences []Sentence `json:"sentences"`
	} `json:"letter"`
	ConnectionNote []Sentence    `json:"connection_note"`
	Message        LetterMessage `json:"message"`
	FollowUp       LetterMessage `json:"follow_up"`
}

// ParseLetter reads Claude's answer for the letter.
func ParseLetter(raw json.RawMessage) (LetterAnswer, error) {
	var answer LetterAnswer
	err := json.Unmarshal(raw, &answer)
	return answer, err
}

// CheckLetter reports why a sentence cannot go in a letter or a message, or ""
// when it can: Check's rules, where the company may be named, and no cliché.
func (c Catalogue) CheckLetter(sentence Sentence) string {
	lower := strings.ToLower(sentence.Text)
	for _, cliche := range cliches {
		if strings.Contains(lower, cliche) {
			return fmt.Sprintf("says %q", cliche)
		}
	}
	// The job's own title may name a skill the person lacks; saying it is not claiming it.
	if title := strings.TrimSpace(c.JobTitle); title != "" {
		if at := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(title)).FindStringIndex(sentence.Text); at != nil {
			sentence.Text = sentence.Text[:at[0]] + "the role" + sentence.Text[at[1]:]
		}
	}
	return c.Check(sentence, "")
}

// Addressing is who a letter is from and to.
type Addressing struct {
	Person    string
	Recruiter string
	Company   string
	Title     string
}

func (a Addressing) firstName() string {
	if fields := strings.Fields(a.Recruiter); len(fields) > 0 {
		return fields[0]
	}
	return ""
}

func (a Addressing) hello() string {
	if name := a.firstName(); name != "" {
		return "Hi " + name + ","
	}
	return "Hello,"
}

func (a Addressing) signed(closing string) string {
	if strings.TrimSpace(a.Person) == "" {
		return closing
	}
	return closing + "\n" + strings.TrimSpace(a.Person)
}

// AssembleLetter builds the texts from Claude's answer: the sentences that pass
// CheckLetter, the greetings and sign-offs, a subject of the code's own for one
// that fails, and a connection note within LinkedIn's limit. It returns the texts
// and why each refused sentence was refused.
func (c Catalogue) AssembleLetter(answer LetterAnswer, to Addressing) (models.LetterTexts, []string) {
	refusals := []string{}
	pass := func(where string, sentences []Sentence) ([]string, []string) {
		kept, facts := []string{}, []string{}
		for i, sentence := range sentences {
			sentence.Text = plain(sentence.Text)
			if reason := c.CheckLetter(sentence); reason != "" {
				refusals = append(refusals, fmt.Sprintf("%s %d: %s", where, i+1, reason))
				continue
			}
			kept = append(kept, sentence.Text)
			facts = append(facts, sentence.Facts...)
		}
		return kept, facts
	}

	paragraphs := []string{}
	for i, paragraph := range answer.Letter {
		if kept, _ := pass(fmt.Sprintf("letter paragraph %d sentence", i+1), paragraph.Sentences); len(kept) > 0 {
			paragraphs = append(paragraphs, strings.Join(kept, " "))
		}
	}
	texts := models.LetterTexts{}
	if len(paragraphs) > 0 {
		greeting := "Dear hiring team,"
		switch {
		case strings.TrimSpace(to.Recruiter) != "":
			greeting = "Dear " + strings.TrimSpace(to.Recruiter) + ","
		case strings.TrimSpace(to.Company) != "":
			greeting = "Dear " + strings.TrimSpace(to.Company) + " hiring team,"
		}
		texts.CoverLetter = strings.Join(append(append([]string{greeting}, paragraphs...), to.signed("Kind regards,")), "\n\n")
	}

	note, _ := pass("connection note sentence", answer.ConnectionNote)
	prefix := ""
	if name := to.firstName(); name != "" {
		prefix = "Hi " + name + ", "
	}
	for len(note) > 0 && len(prefix+strings.Join(note, " ")) > MaxConnectionNote {
		refusals = append(refusals, "connection note: over LinkedIn's limit")
		note = note[:len(note)-1]
	}
	if len(note) > 0 {
		texts.ConnectionNote = prefix + strings.Join(note, " ")
	}

	message := func(where string, written LetterMessage, fallback string) models.Message {
		body, facts := pass(where+" sentence", written.Sentences)
		if len(body) == 0 {
			return models.Message{}
		}
		subject := plain(written.Subject)
		if reason := c.CheckLetter(Sentence{Text: subject, Facts: facts}); reason != "" || len(subject) > 120 {
			refusals = append(refusals, where+" subject: "+reason)
			subject = fallback
		}
		return models.Message{Subject: subject, Body: to.hello() + "\n\n" + strings.Join(body, " ") + "\n\n" + to.signed("Best regards,")}
	}
	title := strings.TrimSpace(to.Title)
	texts.Message = message("message", answer.Message, strings.TrimSpace("Application for "+title))
	texts.FollowUp = message("follow-up", answer.FollowUp, strings.TrimSpace("Following up on my application for "+title))
	return texts, refusals
}

// plain trims a sentence and writes an em dash as a comma, the way a person types.
func plain(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, " — ", ", ")
	text = strings.ReplaceAll(text, " – ", ", ")
	return strings.ReplaceAll(text, "—", ", ")
}
