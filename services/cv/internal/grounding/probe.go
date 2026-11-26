package grounding

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	"cv/internal/models"
)

// ProbeSystem is the system prompt of the questions for Prepare your CV.
//
//go:embed prompts/probe-system.md
var ProbeSystem string

//go:embed prompts/probe.schema.json
var probeSchema []byte

// ProfileProbeSystem is the system prompt of the questions that make the whole
// profile stronger, for every job.
//
//go:embed prompts/profile-probe-system.md
var ProfileProbeSystem string

// ProbeSchema is the questions' JSON schema on one line.
func ProbeSchema() string { return compact(probeSchema) }

type probeMessage struct {
	Analysis *Analysis `json:"analysis"`
	Roles    []Entry   `json:"roles"`
	Projects []Entry   `json:"projects"`
	Facts    []Fact    `json:"facts"`
}

// ProbePrompt is the message of the questions: the job's analysis and the facts.
func (c Catalogue) ProbePrompt(analysis *Analysis) string {
	raw, err := json.MarshalIndent(probeMessage{Analysis: analysis, Roles: nonEmpty(c.Roles), Projects: nonEmpty(c.Projects), Facts: c.FactList()}, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(raw)
}

type probeAnswer struct {
	Questions []struct {
		Target   string   `json:"target"`
		Theme    string   `json:"theme"`
		Question string   `json:"question"`
		Why      string   `json:"why"`
		Example  string   `json:"example"`
		Choices  []string `json:"choices"`
	} `json:"questions"`
}

// ParseProbe reads the questions and says, for each, what it is about and where
// its answer goes. A question about a fact or an entry the catalogue does not
// hold is asked about the person as a whole.
func (c Catalogue) ParseProbe(raw json.RawMessage) ([]models.ProbeQuestion, error) {
	var answer probeAnswer
	if err := json.Unmarshal(raw, &answer); err != nil {
		return nil, err
	}
	entries := map[string]Entry{}
	for _, entry := range append(append([]Entry{}, c.Roles...), c.Projects...) {
		entries[entry.ID] = entry
	}
	questions := []models.ProbeQuestion{}
	for _, item := range answer.Questions {
		question := strings.TrimSpace(item.Question)
		if question == "" {
			continue
		}
		about, place := "", "other"
		target := strings.TrimSpace(item.Target)
		if fact, ok := c.Facts[target]; ok && fact.Owner != "" {
			about, place = fact.Text, entries[fact.Owner].Ref
		} else if entry, ok := entries[target]; ok {
			about, place = entry.Title, entry.Ref
		}
		questions = append(questions, models.ProbeQuestion{
			Theme: strings.TrimSpace(item.Theme), Question: question, Why: strings.TrimSpace(item.Why),
			Example: strings.TrimSpace(item.Example), About: about, Place: place, Choices: choices(item.Choices),
		})
	}
	return questions, nil
}

// Fingerprint changes whenever the facts do: questions asked for older facts are asked again.
func (c Catalogue) Fingerprint() string {
	raw, _ := json.Marshal(c.FactList())
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// MaxChoice is the longest answer a question offers to pick.
const MaxChoice = 80

// choices are the answers to pick, at most 4, each once.
func choices(offered []string) []string {
	kept := []string{}
	for _, choice := range offered {
		choice = strings.TrimSpace(choice)
		if choice != "" && len(choice) <= MaxChoice && !slices.Contains(kept, choice) && len(kept) < 4 {
			kept = append(kept, choice)
		}
	}
	return kept
}
