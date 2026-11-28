package cvdoc

import (
	"slices"
	"strings"

	"cv/internal/models"
)

// ReadyPercent is how complete a profile must be (Readiness) before Claude is asked anything for it: questions, an
// analysis, a CV or a letter written from too little only spend tokens.
const ReadyPercent = 60

// Readiness is how complete a profile is, in percent: the share of the web app's checks it passes
// (apps/frontend/src/features/profile/readiness.ts, which keeps the same checks).
func Readiness(p models.Profile) int {
	filled := func(value string) bool { return strings.TrimSpace(value) != "" }
	bullets := []models.Bullet{}
	for _, role := range p.Experience {
		bullets = append(bullets, role.Bullets...)
	}
	for _, project := range p.Projects {
		bullets = append(bullets, project.Bullets...)
	}
	checks := []bool{
		filled(p.Name),
		filled(p.Headline),
		filled(p.About),
		filled(p.Email),
		filled(p.Phone),
		filled(p.GitHub) || filled(p.LinkedIn),
		filled(p.City) && filled(p.Country),
		filled(p.WorkAuthorization),
		len(p.Skills) >= 5,
		slices.ContainsFunc(p.Experience, richRole),
		slices.ContainsFunc(bullets, func(b models.Bullet) bool { return filled(b.Result) }),
		slices.ContainsFunc(p.Education, func(e models.Education) bool { return filled(e.Level) }),
		len(p.Languages) >= 1,
	}
	passed := 0
	for _, ok := range checks {
		if ok {
			passed++
		}
	}
	return (passed*200 + len(checks)) / (2 * len(checks))
}

// richRole is a role the tailoring can work with: a start month, at least three tools and two highlights.
func richRole(role models.Experience) bool {
	return len(role.Tools) >= 3 && len(role.Bullets) >= 2 && strings.TrimSpace(role.Start) != ""
}

// HasWork reports whether the profile tells what the person did: a role or a project with a highlight.
func HasWork(p models.Profile) bool {
	said := func(bullets []models.Bullet) bool {
		return slices.ContainsFunc(bullets, func(b models.Bullet) bool { return strings.TrimSpace(b.Text) != "" })
	}
	return slices.ContainsFunc(p.Experience, func(r models.Experience) bool { return said(r.Bullets) }) ||
		slices.ContainsFunc(p.Projects, func(pr models.Project) bool { return said(pr.Bullets) })
}

// ReadyForClaude reports whether the profile gives Claude enough to work with: ReadyPercent complete and some work.
func ReadyForClaude(p models.Profile) bool {
	return Readiness(p) >= ReadyPercent && HasWork(p)
}
