package unit_test

import (
	"strings"
	"testing"

	"cv/internal/cvdoc"
	"cv/internal/grounding"
	"cv/internal/models"
	"cv/internal/style"
)

func TestStyleFindsWhatReadsMachineMade(t *testing.T) {
	document := models.CvDocument{
		Summary: "DevOps Engineer with 2 years of experience running Kubernetes, Terraform, Helm and Grafana. Builds robust platforms.",
		Skills:  []string{"Kubernetes", "FluxCD", "Flux CD", "GitHub Actions"},
		Experience: []models.RoleSection{{Role: "DevOps Engineer", Bullets: []string{
			"Run monitoring with Prometheus, Loki and Grafana, ensuring uptime.",
			"Run clusters on AWS, delivering 50+ microservices.",
			"Run CI on GitHub Actions for every team, including the data team.",
			"Run the on-call rota and write the runbooks for it.",
		}}},
		Projects: []models.ProjectSection{{Name: "Home lab", Bullets: []string{"Built a small cluster at home to learn eBPF."}}},
	}
	report := style.Lint(document, []string{"Prometheus", "Loki", "Grafana", "AWS", "Terraform", "Helm", "GitHub"})
	equal(t, report.Sentences, 7)
	equal(t, report.ByRule, map[string]int{style.Tools: 2, style.IngTail: 2, style.Stock: 2, style.SameStart: 1, style.Uniform: 1,
		style.Formula: 1, style.DoubleSkill: 1})
	found := map[string]string{}
	for _, finding := range report.Findings {
		found[finding.Rule+" "+finding.Where] = finding.Text
	}
	equal(t, found[`tools role "DevOps Engineer" bullet 1`], "names 3 tools: Prometheus, Grafana, Loki")
	equal(t, found[`ing-tail role "DevOps Engineer" bullet 2`], ", delivering")
	equal(t, found["double-skill skills"], "FluxCD and Flux CD")
	equal(t, found["formula summary"], "DevOps Engineer with 2 years of experience running Kubernetes")
	equal(t, strings.HasPrefix(found[`same-start role "DevOps Engineer" bullet 3; role "DevOps Engineer" bullet 4`], `4 bullets start with "run"`), true)
	equal(t, report.Score, 1.43)

	clean := style.Lint(models.CvDocument{Summary: "Platform engineer for a payments company.", Experience: []models.RoleSection{{Role: "x",
		Bullets: []string{"Cut deploys from an hour to ten minutes.", "Moved 40 services to Kubernetes without downtime, one team at a time."}}}}, nil)
	equal(t, clean.Findings, []style.Finding{})
	equal(t, clean.Score, 0.0)
	equal(t, style.Lint(models.CvDocument{}, nil).Score, 0.0)
	equal(t, cvdoc.SameSkill("Argo CD", "ArgoCD"), true)
	equal(t, cvdoc.SameSkill("Helm", "Docker"), false)
}

func TestTheReviewGetsTheStyleFindingsAndSkillsAreNamedOnce(t *testing.T) {
	c, job := catalogue(t)
	prompt := c.ReviewPrompt(job, request(), c.Known, nil, syntheticAnswer(t), []style.Finding{{Where: "summary", Rule: style.Formula, Text: "x"}})
	equal(t, strings.Contains(prompt, `"rule":"formula"`), true)
	equal(t, strings.Contains(c.Prompt(job, request(), c.Known, nil), `"style"`), false)

	dup := grounding.Catalogue{Facts: map[string]grounding.Fact{}, Known: []string{"FluxCD", "Flux CD", "Helm"}, Matched: []string{"Flux CD"}}
	answer := grounding.Answer{Skills: []string{"FluxCD", "Flux CD", "Helm"}}
	document, _ := dup.Assemble(answer, job, models.CvDocument{})
	equal(t, document.Skills, []string{"FluxCD", "Helm"})
}

func TestStyleReadsLettersWithoutTheirGreetings(t *testing.T) {
	texts := models.LetterTexts{
		CoverLetter: "Dear Acme hiring team,\n\nI am excited to apply. I run Kubernetes, Terraform and Helm daily, delivering releases.\n\n" +
			"Your platform team needs faster releases.\n\nKind regards,\nAna Pop",
		ConnectionNote: "Hi Ana, I applied for the role and would love the opportunity to connect.",
		Message:        models.Message{Subject: "Application", Body: "Hi Ana,\n\nI applied today. Could we talk?\n\nBest regards,\nAna Pop"},
		FollowUp:       models.Message{Subject: "Following up", Body: "Hello,\n\nJust checking in on my application.\n\nBest regards,\nAna Pop"},
	}
	report := style.LintLetter(texts, []string{"Kubernetes", "Terraform", "Helm"})
	equal(t, report.Sentences, 7)
	equal(t, report.ByRule, map[string]int{style.Stock: 3, style.Tools: 1, style.IngTail: 1, style.OpensWithI: 1})
	clean := style.LintLetter(models.LetterTexts{CoverLetter: "Dear team,\n\nYour releases take a day. Mine take an hour.\n\nKind regards,\nAna"}, nil)
	equal(t, clean.Findings, []style.Finding{})
	equal(t, clean.Sentences, 2)
}
