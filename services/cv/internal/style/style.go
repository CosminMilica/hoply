// Package style finds what makes a written CV read as machine-made: bullets that
// list tools, trailing "-ing" clauses, stock words, bullets that all start or
// run alike, a summary formula and a skills list with the same skill twice.
// docs/cv-writing.md lists the rules.
package style

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"cv/internal/cvdoc"
	"cv/internal/models"
)

// Rules, as a finding names them.
const (
	Tools       = "tools"        // a sentence names more than MaxTools skills
	IngTail     = "ing-tail"     // a sentence ends on ", delivering …"
	Stock       = "stock-word"   // a word recruiters read as filler
	SameStart   = "same-start"   // too many bullets start with the same word
	Uniform     = "uniform"      // a role's bullets are all about as long
	Formula     = "formula"      // "<title> with N years of experience …"
	DoubleSkill = "double-skill" // the skills name one skill twice
)

// MaxTools is how many skills one sentence may name before it reads as a list.
const MaxTools = 2

// stockWords are matched as whole words, case-insensitively.
var stockWords = []string{"leverage", "leveraged", "leveraging", "spearheaded", "spearheading", "orchestrated", "robust",
	"seamless", "seamlessly", "cutting-edge", "state-of-the-art", "end-to-end", "holistic", "synergy", "synergies",
	"utilize", "utilized", "utilizing", "facilitated", "empowered", "empowering", "streamlined", "streamlining",
	"ensure", "ensured", "ensuring", "passionate", "dynamic", "innovative", "best-in-class", "world-class",
	"results-driven", "proven track record", "comprehensive", "successfully", "effectively", "efficiently",
	"pivotal", "instrumental", "foster", "fostered", "elevate", "elevated", "showcase", "showcased", "various"}

var (
	ingTail = regexp.MustCompile(`,\s+(?:and\s+)?([a-z]+ing)\b`)
	formula = regexp.MustCompile(`(?i)^[\w /&+.-]{2,60}\bwith\s+\d+\+?\s+years?\s+of\s+experience\b`)
)

// notTails are "-ing" words that start a plain clause, not a tacked-on claim.
var notTails = map[string]bool{"including": true, "during": true, "according": true, "something": true, "nothing": true, "anything": true}

// Finding is one thing that reads as machine-made, where it is.
type Finding struct {
	Where string `json:"where"`
	Rule  string `json:"rule"`
	Text  string `json:"text"`
}

// Report is what Lint found. Score is findings per sentence: 0 is clean.
type Report struct {
	Sentences int            `json:"sentences"`
	Findings  []Finding      `json:"findings"`
	ByRule    map[string]int `json:"by_rule"`
	Score     float64        `json:"score"`
}

// Lint reads the CV's summary and bullets; known are the skills it counts as
// tools (the person's skills and the CV's).
func Lint(document models.CvDocument, known []string) Report {
	report := Report{Findings: []Finding{}, ByRule: map[string]int{}}
	add := func(where, rule, text string) {
		report.Findings = append(report.Findings, Finding{Where: where, Rule: rule, Text: text})
		report.ByRule[rule]++
	}
	skills := append(append([]string{}, known...), document.Skills...)
	sentence := func(where, text string) {
		report.Sentences++
		if named := tools(text, skills); len(named) > MaxTools {
			add(where, Tools, fmt.Sprintf("names %d tools: %s", len(named), strings.Join(named, ", ")))
		}
		for _, match := range ingTail.FindAllStringSubmatch(text, -1) {
			if !notTails[strings.ToLower(match[1])] {
				add(where, IngTail, strings.TrimSpace(match[0]))
			}
		}
		lower := strings.ToLower(text)
		for _, word := range stockWords {
			if cvdoc.HasWord(lower, word) {
				add(where, Stock, word)
			}
		}
	}

	for i, line := range sentences(document.Summary) {
		sentence(fmt.Sprintf("summary %d", i+1), line)
	}
	if formula.MatchString(strings.TrimSpace(document.Summary)) {
		add("summary", Formula, strings.SplitN(document.Summary, ",", 2)[0])
	}
	starts := map[string][]string{}
	bullets := func(kind, name string, items []string) {
		lengths := []float64{}
		for i, bullet := range items {
			where := fmt.Sprintf("%s %q bullet %d", kind, name, i+1)
			sentence(where, bullet)
			if fields := strings.Fields(bullet); len(fields) > 0 {
				first := strings.ToLower(strings.Trim(fields[0], ",.;:"))
				starts[first] = append(starts[first], where)
			}
			lengths = append(lengths, float64(len(bullet)))
		}
		if len(lengths) >= 4 && variation(lengths) < 0.15 {
			add(fmt.Sprintf("%s %q", kind, name), Uniform, "every bullet is about as long")
		}
	}
	for _, role := range document.Experience {
		bullets("role", role.Role, role.Bullets)
	}
	for _, project := range document.Projects {
		bullets("project", project.Name, project.Bullets)
	}
	words := make([]string, 0, len(starts))
	for word := range starts {
		words = append(words, word)
	}
	slices.Sort(words)
	for _, word := range words {
		if places := starts[word]; len(places) > 2 {
			add(strings.Join(places[2:], "; "), SameStart, fmt.Sprintf("%d bullets start with %q", len(places), word))
		}
	}
	for i, skill := range document.Skills {
		for _, earlier := range document.Skills[:i] {
			if cvdoc.SameSkill(skill, earlier) {
				add("skills", DoubleSkill, earlier+" and "+skill)
				break
			}
		}
	}
	if report.Sentences > 0 {
		report.Score = math.Round(float64(len(report.Findings))/float64(report.Sentences)*100) / 100
	}
	return report
}

// tools are the skills a sentence names, each once, the longest name first so
// "GitHub Actions" is not also counted as "GitHub".
func tools(text string, skills []string) []string {
	ordered := slices.Clone(skills)
	slices.SortStableFunc(ordered, func(a, b string) int { return len(b) - len(a) })
	named := []string{}
	rest := " " + text + " "
	for _, skill := range ordered {
		skill = strings.TrimSpace(skill)
		if skill == "" || !cvdoc.HasWord(rest, skill) || slices.ContainsFunc(named, func(seen string) bool { return cvdoc.SameSkill(seen, skill) }) {
			continue
		}
		named = append(named, skill)
		rest = strings.ReplaceAll(rest, skill, " ")
	}
	return named
}

var sentenceEnd = regexp.MustCompile(`[.!?]\s+`)

func sentences(text string) []string {
	out := []string{}
	for _, part := range sentenceEnd.Split(strings.TrimSpace(text), -1) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// variation is the coefficient of variation: the spread of the values against their mean.
func variation(values []float64) float64 {
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	spread := 0.0
	for _, value := range values {
		spread += (value - mean) * (value - mean)
	}
	return math.Sqrt(spread/float64(len(values))) / mean
}
