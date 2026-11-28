package cvdoc

import "cv/internal/models"

var examples = map[string]string{
	"terraform":              "Wrote Terraform modules for our staging environments, cutting setup from 2 days to 30 minutes",
	"snowflake":              "Managed roles and warehouses in Snowflake for a team of 20 analysts",
	"rbac":                   "Set up RBAC roles for 15 teams and removed shared admin accounts",
	"infrastructure as code": "Moved 40 servers to Infrastructure as Code, so a rebuild now takes minutes",
	"kubernetes":             "Ran a Kubernetes cluster with 30 services and cut deploy time by half",
	"aws":                    "Moved 12 services to AWS and reduced hosting cost by 20%",
	"java":                   "Built a Java service that handles 5,000 requests per minute",
	"algorithms":             "Replaced a slow search with a better algorithm, cutting response time from 4s to 300ms",
	"distributed systems":    "Worked on a queue-based system that processes 1M events a day",
	"testing":                "Added automated tests for the checkout flow and caught 30 bugs before release",
	"windows server":         "Administered 12 Windows Servers and automated monthly patching",
	"active directory":       "Managed Active Directory for 150 users, including groups and policies",
	"networking":             "Set up VLANs and VPN access for 3 offices",
	"backup":                 "Set up nightly backups and tested restores every month",
	"virtualization":         "Ran 25 virtual machines on VMware and moved 8 of them to new hosts",
}

// ExampleFor is the sample answer shown under a question about skill.
func ExampleFor(skill string) string {
	if example, ok := examples[lc(skill)]; ok {
		return example
	}
	return "Used " + skill + " to ..., which improved ... by ..."
}

// Questions is what Prepare your CV asks: the job's keywords the profile does
// not show yet (in its skills, its tools, a highlight or a note that proves one),
// the ones it shows only a close equivalent of, and the job's key responsibilities.
func Questions(job Job, profile models.Profile) models.PrepQuestions {
	match := MatchJob(job, KnownSkills(profile))
	gaps := make([]models.Gap, 0, len(match.Missing))
	for _, skill := range match.Missing {
		gaps = append(gaps, models.Gap{
			Skill:   skill,
			Ask:     "Have you used " + skill + "? Tell us where and what you did. Numbers help: how many, how much, how fast.",
			Example: "e.g. " + ExampleFor(skill),
		})
	}
	related := make([]models.Close, 0, len(match.Related))
	for _, item := range match.Related {
		related = append(related, models.Close{
			Skill:    item.Skill,
			Via:      item.Via,
			Category: item.Category,
			Note: "You use " + JoinList(item.Via) + ", a " + item.Category + " tool like " + item.Skill + ". Your CV shows your " +
				item.Category + " experience, without naming " + item.Skill + ".",
		})
	}
	responsibilities := []models.Responsibility{}
	for _, index := range job.KeyResponsibilities {
		text := job.Responsibilities[index]
		if text == "" {
			continue
		}
		phrase := PastPhrase(text)
		responsibilities = append(responsibilities, models.Responsibility{
			Index: index, Text: text, Question: "Have you " + phrase + "?", Prefill: Capitalize(phrase),
		})
	}
	return models.PrepQuestions{
		JobID:            job.ID,
		BaseScore:        match.Score,
		Covered:          match.Matched,
		Gaps:             gaps,
		Related:          related,
		Responsibilities: responsibilities,
	}
}
