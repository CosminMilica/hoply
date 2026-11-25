package cvdoc

import (
	"math"
	"regexp"
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

// Match is how the job's requirements line up with a set of skills.
type Match struct {
	Matched []string
	Missing []string
	Score   int
}

// MatchJob compares the requirements with skills, ignoring case; a must-have
// counts twice a nice-to-have.
func MatchJob(job Job, skills []string) Match {
	have := map[string]bool{}
	for _, skill := range skills {
		have[lc(skill)] = true
	}
	nice := map[string]bool{}
	for _, item := range job.NiceToHave {
		nice[lc(item)] = true
	}
	weight := func(item string) int {
		if nice[lc(item)] {
			return 1
		}
		return 2
	}
	match := Match{Matched: []string{}, Missing: []string{}}
	total, got := 0, 0
	for _, item := range job.Keywords {
		total += weight(item)
		if have[lc(item)] {
			match.Matched = append(match.Matched, item)
			got += weight(item)
		} else {
			match.Missing = append(match.Missing, item)
		}
	}
	if total > 0 {
		match.Score = int(math.Round(float64(got) / float64(total) * 100))
	}
	return match
}

// KnownSkills is every skill the profile shows: its skills, the tools of each
// role and project, and the skills its highlights prove, once each.
func KnownSkills(profile models.Profile) []string {
	all := append([]string{}, profile.Skills...)
	add := func(tools []string, bullets []models.Bullet) {
		all = append(all, tools...)
		for _, bullet := range bullets {
			all = append(all, bullet.Skills...)
		}
	}
	for _, role := range profile.Experience {
		add(role.Tools, role.Bullets)
	}
	for _, project := range profile.Projects {
		add(project.Tools, project.Bullets)
	}
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

// bulletLine is a highlight as one line: its text, then its result.
func bulletLine(bullet models.Bullet) string {
	result := strings.TrimSpace(bullet.Result)
	if result == "" {
		return bullet.Text
	}
	text := trailingDots.ReplaceAllString(strings.TrimSpace(bullet.Text), "")
	_, size := utf8.DecodeRuneInString(result)
	return text + ", " + lowerFirst(result[:size]) + trailingDots.ReplaceAllString(result[size:], "") + "."
}
