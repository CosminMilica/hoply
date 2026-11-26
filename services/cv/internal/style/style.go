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

type linter struct {
	report Report
	skills []string
}

// newLinter checks sentences against the known skills, the longest name first (see tools).
func newLinter(known []string) *linter {
	ordered := slices.Clone(known)
	slices.SortStableFunc(ordered, func(a, b string) int { return len(b) - len(a) })
	return &linter{report: Report{Findings: []Finding{}, ByRule: map[string]int{}}, skills: ordered}
}

func (l *linter) add(where, rule, text string) {
	l.report.Findings = append(l.report.Findings, Finding{Where: where, Rule: rule, Text: text})
	l.report.ByRule[rule]++
}

// sentence applies the rules every sentence follows, plus the extra stock words.
func (l *linter) sentence(where, text string, extra ...string) {
	l.report.Sentences++
	if named := tools(text, l.skills); len(named) > MaxTools {
		l.add(where, Tools, fmt.Sprintf("names %d tools: %s", len(named), strings.Join(named, ", ")))
	}
	for _, match := range ingTail.FindAllStringSubmatch(text, -1) {
		if !notTails[strings.ToLower(match[1])] {
			l.add(where, IngTail, strings.TrimSpace(match[0]))
		}
	}
	lower := strings.ToLower(text)
	for _, word := range append(slices.Clone(cvdoc.StockWords), extra...) {
		if cvdoc.HasWord(lower, word) {
			l.add(where, Stock, word)
		}
	}
}

func (l *linter) done() Report {
	if l.report.Sentences > 0 {
		l.report.Score = math.Round(float64(len(l.report.Findings))/float64(l.report.Sentences)*100) / 100
	}
	return l.report
}

// Lint reads the CV's summary and bullets; known are the skills it counts as
// tools (the person's skills and the CV's).
func Lint(document models.CvDocument, known []string) Report {
	l := newLinter(append(append([]string{}, known...), document.Skills...))
	for i, line := range sentences(document.Summary) {
		l.sentence(fmt.Sprintf("summary %d", i+1), line)
	}
	if formula.MatchString(strings.TrimSpace(document.Summary)) {
		l.add("summary", Formula, strings.SplitN(document.Summary, ",", 2)[0])
	}
	starts := map[string][]string{}
	bullets := func(kind, name string, items []string) {
		lengths := []float64{}
		for i, bullet := range items {
			where := fmt.Sprintf("%s %q bullet %d", kind, name, i+1)
			l.sentence(where, bullet)
			if fields := strings.Fields(bullet); len(fields) > 0 {
				first := strings.ToLower(strings.Trim(fields[0], ",.;:"))
				starts[first] = append(starts[first], where)
			}
			lengths = append(lengths, float64(len(bullet)))
		}
		if len(lengths) >= 4 && variation(lengths) < 0.15 {
			l.add(fmt.Sprintf("%s %q", kind, name), Uniform, "every bullet is about as long")
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
			l.add(strings.Join(places[2:], "; "), SameStart, fmt.Sprintf("%d bullets start with %q", len(places), word))
		}
	}
	for i, skill := range document.Skills {
		for _, earlier := range document.Skills[:i] {
			if cvdoc.SameSkill(skill, earlier) {
				l.add("skills", DoubleSkill, earlier+" and "+skill)
				break
			}
		}
	}
	return l.done()
}

// OpensWithI is a cover letter that starts with the person, not the company.
const OpensWithI = "opens-with-i"

var (
	salutation = regexp.MustCompile(`^(?:Dear|Hi|Hello)\b.*,$`)
	signOff    = regexp.MustCompile(`^(?:Kind regards|Best regards|Best|Regards|Thank you|Thanks),?$`)
)

// LintLetter reads a cover letter and the messages, without their greetings
// and sign-offs; known are the skills it counts as tools.
func LintLetter(texts models.LetterTexts, known []string) Report {
	l := newLinter(known)
	read := func(where, text string) []string {
		lines := []string{}
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || salutation.MatchString(line) || signOff.MatchString(line) || !strings.ContainsAny(line, ".?") && len(line) < 40 {
				continue
			}
			lines = append(lines, sentences(line)...)
		}
		for i, line := range lines {
			l.sentence(fmt.Sprintf("%s %d", where, i+1), line, cvdoc.LetterCliches...)
		}
		return lines
	}
	if body := read("letter", texts.CoverLetter); len(body) > 0 && strings.HasPrefix(body[0], "I ") {
		l.add("letter 1", OpensWithI, body[0])
	}
	read("note", texts.ConnectionNote)
	read("message", texts.Message.Body)
	read("follow-up", texts.FollowUp.Body)
	return l.done()
}

// tools are the skills a sentence names, each once; skills come longest first so
// "GitHub Actions" is not also counted as "GitHub".
func tools(text string, skills []string) []string {
	named := []string{}
	rest := " " + text + " "
	for _, skill := range skills {
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
