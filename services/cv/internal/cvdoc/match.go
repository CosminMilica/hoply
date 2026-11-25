package cvdoc

import (
	_ "embed"
	"encoding/json"
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"cv/internal/models"
)

// Job is a job as the engine reads it.
type Job struct {
	models.Job
	// Keywords are the requirements to match and write around: must-haves,
	// then nice-to-haves.
	Keywords []string
	// Requirements are the requirements as sentences, for the job's own wording.
	Requirements []string
	// KeyResponsibilities are the indexes Prepare your CV asks about: the first four.
	KeyResponsibilities []int
}

// ReadJob prepares a job for the engine; a job without a product works on
// "products".
func ReadJob(job models.Job) Job {
	if job.Product == "" {
		job.Product = "products"
	}
	keywords := append(append([]string{}, job.MustHave...), job.NiceToHave...)
	requirements := make([]string, 0, len(keywords))
	for _, item := range job.MustHave {
		requirements = append(requirements, "Experience with "+item+".")
	}
	for _, item := range job.NiceToHave {
		requirements = append(requirements, item+" is a plus.")
	}
	key := []int{}
	for index := range job.Responsibilities {
		if index < 4 {
			key = append(key, index)
		}
	}
	return Job{Job: job, Keywords: keywords, Requirements: requirements, KeyResponsibilities: key}
}

// Related is a requirement the person has not used but knows a close
// equivalent of (Argo CD for someone who uses FluxCD).
type Related struct {
	Skill string   `json:"skill"`
	Via   []string `json:"via"`
	// Category is what the two have in common ("GitOps" for Argo CD and FluxCD):
	// the CV names it beside the person's tool, and never the requirement itself.
	Category string `json:"category"`
}

// Match is how the job's requirements line up with a set of skills.
type Match struct {
	Matched []string
	Related []Related
	Missing []string
	Score   int
}

// MatchJob compares the requirements with skills (skillmap.json): a
// requirement is matched when the person has it under any of its names or has a
// tool that proves it (GitHub Actions proves CI/CD), related when they have a
// close equivalent, missing otherwise. A must-have counts twice a nice-to-have;
// a related one counts half.
func MatchJob(job Job, skills []string) Match {
	have := map[string]string{}
	for _, skill := range skills {
		if key := skillKey(skill); key != "" {
			if _, ok := have[key]; !ok {
				have[key] = skill
			}
		}
	}
	nice := map[string]bool{}
	for _, item := range job.NiceToHave {
		nice[lc(item)] = true
	}
	weight := func(item string) int {
		if nice[lc(item)] {
			return 2
		}
		return 4
	}
	match := Match{Matched: []string{}, Related: []Related{}, Missing: []string{}}
	total, got := 0, 0
	for _, item := range job.Keywords {
		total += weight(item)
		switch via, category := closeTo(item, have); {
		case covered(item, have):
			match.Matched = append(match.Matched, item)
			got += weight(item)
		case len(via) > 0:
			match.Related = append(match.Related, Related{Skill: item, Via: via, Category: category})
			got += weight(item) / 2
		default:
			match.Missing = append(match.Missing, item)
		}
	}
	if total > 0 {
		match.Score = int(math.Round(float64(got) / float64(total) * 100))
	}
	return match
}

// KnownSkills is every skill the profile shows: its skills, the tools of each
// role and project, and the skills its highlights and notes prove, once each.
func KnownSkills(profile models.Profile) []string {
	all := append([]string{}, profile.Skills...)
	add := func(tools []string, bullets []models.Bullet, notes []models.Note) {
		all = append(all, tools...)
		for _, bullet := range bullets {
			all = append(all, bullet.Skills...)
		}
		for _, note := range notes {
			all = append(all, note.Skills...)
		}
	}
	for _, role := range profile.Experience {
		add(role.Tools, role.Bullets, role.Notes)
	}
	for _, project := range profile.Projects {
		add(project.Tools, project.Bullets, project.Notes)
	}
	add(nil, nil, profile.Notes)
	seen := map[string]bool{}
	known := []string{}
	for _, skill := range all {
		key := lc(strings.TrimSpace(skill))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		known = append(known, skill)
	}
	return known
}

var trailingDots = regexp.MustCompile(`[.]+$`)

// BulletLine is a highlight as one line: its text, then its result.
func BulletLine(bullet models.Bullet) string {
	result := strings.TrimSpace(bullet.Result)
	if result == "" {
		return bullet.Text
	}
	text := trailingDots.ReplaceAllString(strings.TrimSpace(bullet.Text), "")
	_, size := utf8.DecodeRuneInString(result)
	return text + ", " + lowerFirst(result[:size]) + trailingDots.ReplaceAllString(result[size:], "") + "."
}

//go:embed skillmap.json
var skillMapJSON []byte

// skillMap is skillmap.json: the names of one skill, the tools that prove an
// umbrella requirement, and families of interchangeable tools. The web app
// holds a copy (apps/frontend/src/features/jobs/skillmap.json), kept identical
// by a test.
var skillMap = func() (m struct {
	Same   [][]string          `json:"same"`
	Covers map[string][]string `json:"covers"`
	Close  []struct {
		Category string   `json:"category"`
		Tools    []string `json:"tools"`
	} `json:"close"`
}) {
	if err := json.Unmarshal(skillMapJSON, &m); err != nil {
		panic(err)
	}
	return m
}()

var (
	sameKey  = map[string]string{}
	coverBy  = map[string][]string{}
	families = []family{}
)

// family is tools of one kind and the name of that kind.
type family struct {
	category string
	keys     []string
}

func init() {
	for _, group := range skillMap.Same {
		for _, name := range group {
			sameKey[fold(name)] = fold(group[0])
		}
	}
	for umbrella, tools := range skillMap.Covers {
		keys := make([]string, len(tools))
		for i, tool := range tools {
			keys[i] = skillKey(tool)
		}
		coverBy[skillKey(umbrella)] = keys
	}
	for _, group := range skillMap.Close {
		keys := make([]string, len(group.Tools))
		for i, tool := range group.Tools {
			keys[i] = skillKey(tool)
		}
		families = append(families, family{category: group.Category, keys: keys})
	}
}

var notAlnum = regexp.MustCompile(`[^\pL\pN+#]+`)

// fold is a name without case, spaces or punctuation: "Argo CD" and "argocd" fold alike.
func fold(name string) string {
	return notAlnum.ReplaceAllString(lc(strings.TrimSpace(name)), "")
}

// skillKey identifies a skill whatever name it goes by.
func skillKey(name string) string {
	key := fold(name)
	if same, ok := sameKey[key]; ok {
		return same
	}
	return key
}

// covered reports whether the person has the requirement or a tool that proves it.
func covered(requirement string, have map[string]string) bool {
	key := skillKey(requirement)
	if _, ok := have[key]; ok {
		return true
	}
	for _, tool := range coverBy[key] {
		if _, ok := have[tool]; ok {
			return true
		}
	}
	return false
}

// closeTo is the person's skills, in their spelling, that are interchangeable
// with the requirement, and the kind of tool they share.
func closeTo(requirement string, have map[string]string) ([]string, string) {
	key := skillKey(requirement)
	via, category := []string{}, ""
	for _, group := range families {
		if !slices.Contains(group.keys, key) {
			continue
		}
		for _, member := range group.keys {
			if skill, ok := have[member]; ok && member != key && !slices.Contains(via, skill) {
				via = append(via, skill)
				if category == "" {
					category = group.category
				}
			}
		}
	}
	return via, category
}

// SameSkill reports whether two names are one skill ("Flux CD" and "FluxCD").
func SameSkill(a, b string) bool { return skillKey(a) == skillKey(b) }
