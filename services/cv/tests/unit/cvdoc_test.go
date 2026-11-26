package unit_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"cv/internal/cvdoc"
	"cv/internal/models"
)

// sampleProfile is tests/testdata/profile.json, the web app's sample profile.
func sampleProfile(t *testing.T) models.Profile {
	t.Helper()
	var profile models.Profile
	readJSON(t, "../testdata/profile.json", &profile)
	return profile
}

// catalogJob is a seed job (0: Snowflake at Capgemini, 1: DevOps at Bitdefender).
func catalogJob(t *testing.T, index int) models.Job {
	t.Helper()
	var jobs []models.Job
	readJSON(t, "../testdata/jobs.json", &jobs)
	return jobs[index]
}

func engineJob(t *testing.T, index int) cvdoc.Job {
	t.Helper()
	return cvdoc.ReadJob(catalogJob(t, index))
}

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatal(err)
	}
}

func equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\n got: %#v\nwant: %#v", got, want)
	}
}

var now = time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)

func request(modify ...func(*models.CvRequest)) models.CvRequest {
	r := models.CvRequest{JobID: "00000000-0000-4000-8000-000000000001", SkipRoles: []string{}, Focus: []string{}, Confirmed: []int{}}
	for _, fn := range modify {
		fn(&r)
	}
	return r
}

func TestTextHelpers(t *testing.T) {
	equal(t, cvdoc.JoinList(nil), "")
	equal(t, cvdoc.JoinList([]string{"SQL"}), "SQL")
	equal(t, cvdoc.JoinList([]string{"SQL", "Git"}), "SQL and Git")
	equal(t, cvdoc.JoinList([]string{"SQL", "Git", "Bash"}), "SQL, Git and Bash")

	equal(t, cvdoc.HasWord("Built CI/CD pipelines", "ci/cd"), true)
	equal(t, cvdoc.HasWord("Ran SQL queries", "SQL"), true)
	equal(t, cvdoc.HasWord("Ran MySQL queries", "SQL"), false)
	for _, skill := range []string{"C++", "C#", ".NET", "Node.js"} {
		equal(t, cvdoc.HasWord("Built services in "+skill+", daily", skill), true)
		equal(t, cvdoc.HasWord(skill+" at scale", skill), true)
	}
	equal(t, cvdoc.HasWord("Wrote C++17 code", "C++"), false)
	equal(t, cvdoc.HasWord("Wrote C++ code", "C"), false)
	equal(t, cvdoc.HasWord("Ran Go_lang", "Go"), false)

	equal(t, cvdoc.PastPhrase("Build and ship features in our React web app."), "built and shipped features in a React web app")
	equal(t, cvdoc.PastPhrase("Monitor your services and fix operational issues quickly."), "monitored the services and fixed operational issues quickly")
	equal(t, cvdoc.PastPhrase("Kubernetes upgrades."), "kubernetes upgrades")
	equal(t, cvdoc.PastOf("Troubleshoot"), "troubleshot")
	equal(t, cvdoc.Capitalize("built it"), "Built it")
	equal(t, cvdoc.Capitalize(""), "")
}

func TestRewriteBulletUsesTheJobsTermsVerbsAndTools(t *testing.T) {
	snowflake := engineJob(t, 0)
	used := map[string]bool{}
	out := cvdoc.RewriteBullet("Built and maintained deployment pipelines for internal services.", 0, cvdoc.RewriteContext{
		Job: snowflake, Matched: []string{"SQL", "Git", "CI/CD", "Monitoring", "Python"},
		Tools: []string{"Linux", "Docker", "CI/CD", "Python", "Git"}, Used: used,
	})
	equal(t, out, "Developed and maintained CI/CD pipelines for internal services, using Git.")
	equal(t, used["git"], true)
}

func TestRewriteBulletAddsAToolOnceAndWithOnOddBullets(t *testing.T) {
	snowflake := engineJob(t, 0)
	out := cvdoc.RewriteBullet("Automated server setup and configuration with scripts.", 1, cvdoc.RewriteContext{
		Job: snowflake, Matched: []string{"Git", "Python"}, Tools: []string{"Git", "Python"}, Used: map[string]bool{"git": true},
	})
	equal(t, out, "Automated server setup and configuration with scripts, using Python.")
	odd := cvdoc.RewriteBullet("Wrote reports for the team.", 1, cvdoc.RewriteContext{
		Job: snowflake, Matched: []string{"SQL"}, Tools: []string{"SQL"}, Used: map[string]bool{},
	})
	equal(t, odd, "Wrote reports for the team with SQL.")
}

func TestRewriteBulletKeepsVerbsOutsideTheGroups(t *testing.T) {
	bitdefender := engineJob(t, 1)
	out := cvdoc.RewriteBullet("Set up backups and checked that they ran.", 0, cvdoc.RewriteContext{Job: bitdefender, Matched: []string{"Linux"}})
	equal(t, out, "Set up backups and checked that they ran.")
}

func TestRewriteBulletTakesOneVerbWhenTheBulletHasTwo(t *testing.T) {
	bitdefender := engineJob(t, 1)
	equal(t, cvdoc.VerbPairs(bitdefender)[0], []string{"designed", "maintained"})
	equal(t, cvdoc.RewriteBullet("Maintained servers and user accounts.", 0, cvdoc.RewriteContext{Job: bitdefender}),
		"Maintained servers and user accounts.")
	equal(t, cvdoc.RewriteBullet("Managed and fixed hosts.", 0, cvdoc.RewriteContext{Job: bitdefender}), "Maintained and fixed hosts.")
}

func TestCleanFact(t *testing.T) {
	equal(t, cvdoc.CleanFact("  wrote   terraform modules...", "Terraform"), "Wrote terraform modules.")
	equal(t, cvdoc.CleanFact("set up roles for 15 teams", "RBAC"), "Set up roles for 15 teams, using RBAC.")
	equal(t, cvdoc.CleanFact("ran the on-call rota", ""), "Ran the on-call rota.")
	equal(t, cvdoc.CleanFact("   ", "SQL"), "")
}

func TestQuestionsAskAboutMissingKeywordsAndKeyResponsibilities(t *testing.T) {
	profile := sampleProfile(t)
	profile.Skills = []string{"Python", "SQL", "Git", "CI/CD", "Monitoring"}
	profile.Experience = nil
	questions := cvdoc.Questions(engineJob(t, 0), profile)
	equal(t, questions.BaseScore, 56)
	equal(t, questions.Covered, []string{"SQL", "Git", "CI/CD", "Monitoring", "Python"})
	skills := []string{}
	for _, gap := range questions.Gaps {
		skills = append(skills, gap.Skill)
	}
	equal(t, skills, []string{"Snowflake", "Terraform", "Infrastructure as Code", "RBAC"})
	equal(t, questions.Gaps[0].Example, "e.g. Managed roles and warehouses in Snowflake for a team of 20 analysts")
	indexes := []int{}
	for _, item := range questions.Responsibilities {
		indexes = append(indexes, item.Index)
	}
	equal(t, indexes, []int{0, 1, 2, 3})
	equal(t, questions.Responsibilities[3].Question, "Have you developed and maintained Terraform modules and reusable deployment templates?")
	equal(t, questions.Responsibilities[3].Prefill, "Developed and maintained Terraform modules and reusable deployment templates")
}

func TestQuestionsDoNotAskAboutSkillsInToolsOrHighlights(t *testing.T) {
	profile := sampleProfile(t)
	profile.Experience[0].Bullets = append(profile.Experience[0].Bullets, models.Bullet{Text: "Managed warehouses", Skills: []string{"Snowflake"}})
	profile.Projects = []models.Project{{ID: "p", Name: "Lab", Tools: []string{"Terraform"}}}
	questions := cvdoc.Questions(engineJob(t, 0), profile)
	skills := []string{}
	for _, gap := range questions.Gaps {
		skills = append(skills, gap.Skill)
	}
	equal(t, skills, []string{"RBAC"})
	equal(t, strings.Contains(strings.Join(questions.Covered, ","), "Snowflake"), true)
	equal(t, questions.BaseScore, 89)
}

func TestQuestionsSkipEmptyResponsibilitiesAndUseAGenericExample(t *testing.T) {
	job := catalogJob(t, 0)
	job.Responsibilities = []string{"", "Build things."}
	job.MustHave = []string{"Rust"}
	questions := cvdoc.Questions(cvdoc.ReadJob(job), sampleProfile(t))
	equal(t, len(questions.Responsibilities), 1)
	equal(t, questions.Gaps[0].Example, "e.g. Used Rust to ..., which improved ... by ...")
	equal(t, cvdoc.ExampleFor("Rust"), "Used Rust to ..., which improved ... by ...")
}

func TestMatchWeighsMustHavesTwiceNiceToHaves(t *testing.T) {
	job := catalogJob(t, 0)
	job.MustHave = []string{"Go"}
	job.NiceToHave = []string{"Rust"}
	equal(t, cvdoc.MatchJob(cvdoc.ReadJob(job), []string{"rust"}).Score, 33)
	equal(t, cvdoc.MatchJob(cvdoc.ReadJob(models.Job{}), []string{"rust"}).Score, 0)
	equal(t, cvdoc.ReadJob(job).Requirements, []string{"Experience with Go.", "Rust is a plus."})
}

func TestWriteKeepsTheHeadlineAndPutsTheTitleInTheSummary(t *testing.T) {
	cv := cvdoc.Write(sampleProfile(t), engineJob(t, 0), request(), now)
	equal(t, cv.Tailored.Headline, "DevOps Engineer")
	equal(t, cv.Profile.Headline, "DevOps Engineer")
	equal(t, cv.Tailored.Summary, "DevOps Engineer with 6 years of experience, strongest in SQL, Git and CI/CD. "+
		"I like reliable systems, clear documentation and calm on-call weeks. "+
		"Looking to join Capgemini Romania as Snowflake Administrator DevOps and work on its enterprise Knowledge Platform on Snowflake.")
	equal(t, cvdoc.FileName("Andrei Popescu", "Capgemini Romania"), "Andrei_Popescu_CV_Capgemini_Romania.pdf")
	equal(t, cv.Tailored.Name, "Andrei Popescu")
	equal(t, cv.Tailored.Contact, []string{"andrei@example.com", "+40 700 000 000", "Iași, Romania"})
	equal(t, cv.Tailored.Skills[:5], []string{"SQL", "Git", "CI/CD", "Monitoring", "Python"})
	equal(t, cv.Profile.Skills, sampleProfile(t).Skills)
	equal(t, cv.Tailored.Links, []string{"github.com/andrei-popescu", "linkedin.com/in/andrei-popescu"})
	equal(t, cv.Tailored.Languages, "Romanian (Native), English (C1)")
}

func TestWritePrintsMonthDatesResultsAndTheEducationTitle(t *testing.T) {
	cv := cvdoc.Write(sampleProfile(t), engineJob(t, 0), request(), now)
	equal(t, []string{cv.Profile.Experience[0].Meta, cv.Profile.Experience[1].Meta},
		[]string{"Verde Cloud | Iași | 03/2022 – Present", "Bonbon Labs | Iași | 09/2020 – 02/2022"})
	equal(t, cv.Profile.Experience[1].Bullets[1], "Set up backups and checked that they ran every night, restored every test backup within an hour.")
	equal(t, cv.Profile.Education, []models.EducationLine{{Degree: "BSc Computer Science", Meta: "University of Iași | 2016 – 2020"}})

	profile := sampleProfile(t)
	profile.Education = []models.Education{{Level: "master", Field: "Data", StartYear: "2021"}}
	byLevel := cvdoc.Write(profile, engineJob(t, 0), request(), now)
	equal(t, byLevel.Profile.Education, []models.EducationLine{{Degree: "Master’s degree in Data", Meta: "2021 – Present"}})
	profile.Education = []models.Education{{Level: "", EndYear: "2020"}}
	equal(t, cvdoc.Write(profile, engineJob(t, 0), request(), now).Profile.Education,
		[]models.EducationLine{{Degree: "Education", Meta: "? – 2020"}})
}

func TestWriteUsesTheFactsRememberedInTheProfile(t *testing.T) {
	profile := sampleProfile(t)
	profile.AdditionalInfo = []string{"HashiCorp Terraform Associate, 2025", "  "}
	profile.Projects = []models.Project{{
		ID: "p1", Name: "Home lab", Role: "Maintainer", URL: "github.com/a/lab", Start: "2023-06", Tools: []string{"Docker", "Terraform"},
		Bullets: []models.Bullet{{Text: "Provisioned 12 containers with Terraform", Skills: []string{"Terraform"}}},
	}}
	profile.Certifications = []models.Certification{{Name: "CKA", Issuer: "CNCF", Issued: "2024-05", Expires: "2027-05", CredentialID: "X1"}}
	profile.Experience[1].Bullets = append(profile.Experience[1].Bullets, models.Bullet{Text: "managed warehouses for 20 analysts", Skills: []string{"Snowflake"}})
	cv := cvdoc.Write(profile, engineJob(t, 0), request(func(r *models.CvRequest) { r.Confirmed = []int{1, 3, 99} }), now)
	highlights := strings.Join(cv.Highlights, ",")
	equal(t, strings.Contains(highlights, "Snowflake") && strings.Contains(highlights, "Terraform"), true)
	equal(t, cv.Tailored.Experience[1].Bullets[0], "Administered warehouses for 20 analysts.")
	equal(t, cv.Tailored.Projects, []models.ProjectSection{{Name: "Home lab", Meta: "Maintainer | github.com/a/lab | 06/2023 – Present", Bullets: []string{"Provisioned 12 containers with Terraform."}}})
	equal(t, cv.Tailored.Certifications, []string{"CKA | CNCF | Issued 05/2024 | Expires 05/2027 | ID X1"})
	equal(t, cv.Tailored.Additional, []string{"HashiCorp Terraform Associate, 2025."})
	equal(t, cv.Profile.Additional, []string{"HashiCorp Terraform Associate, 2025."})
	equal(t, checkStartingWith(t, cv.Ats.Checks, "Job responsibilities"), models.AtsCheck{OK: true, Label: "Job responsibilities mirrored: 2 of 4", Hint: "Describe them in Prepare your CV."})
	equal(t, checkStartingWith(t, cv.Ats.Checks, "Keywords from the job").Label, "Keywords from the job: 8 of 9")
}

func TestWriteLeavesOutSkippedRolesAndFollowsTheFocus(t *testing.T) {
	cv := cvdoc.Write(sampleProfile(t), engineJob(t, 0), request(func(r *models.CvRequest) {
		r.SkipRoles = []string{"r2"}
		r.Focus = []string{"Python", "Snowflake"}
	}), now)
	equal(t, len(cv.Tailored.Experience), 1)
	equal(t, len(cv.Profile.Experience), 2)
	equal(t, cv.Tailored.Headline, "DevOps Engineer")
	equal(t, strings.Contains(cv.Tailored.Summary, "strongest in Python."), true)
}

func TestWriteKeepsSevenBulletsOnTheFirstRoleAndThreeOnTheOthers(t *testing.T) {
	profile := sampleProfile(t)
	for i := 0; i < 6; i++ {
		profile.Experience[0].Bullets = append(profile.Experience[0].Bullets, models.Bullet{Text: "Kept the lights on."})
		profile.Experience[1].Bullets = append(profile.Experience[1].Bullets, models.Bullet{Text: "Kept the lights on."})
	}
	cv := cvdoc.Write(profile, engineJob(t, 0), request(), now)
	equal(t, len(cv.Tailored.Experience[0].Bullets), 7)
	equal(t, len(cv.Tailored.Experience[1].Bullets), 3)
	equal(t, len(cv.Profile.Experience[0].Bullets), 10)
}

func TestWriteCopesWithAThinProfile(t *testing.T) {
	profile := sampleProfile(t)
	profile.Name, profile.Headline, profile.About = "", "", "Short."
	profile.Skills, profile.Experience = nil, nil
	profile.Email, profile.Phone, profile.City, profile.Country = "", "", "", ""
	cv := cvdoc.Write(profile, engineJob(t, 0), request(), now)
	equal(t, strings.HasPrefix(cv.Tailored.Summary, "Engineer with hands-on experience."), true)
	equal(t, cv.Tailored.Skills, []string{})
	equal(t, cvdoc.RewritePreview(profile, engineJob(t, 0), request()) == nil, true)
	equal(t, cvdoc.FileName(profile.Name, "Capgemini Romania"), "CV_Capgemini_Romania.pdf")
	failed := []string{}
	for _, check := range cv.Ats.Checks {
		if !check.OK {
			failed = append(failed, check.Label)
		}
	}
	for _, label := range []string{"Email and phone in the body, not in a header", "Location in the header", "Dates with month and year", "Every bullet starts with an action verb"} {
		if !strings.Contains(strings.Join(failed, "|"), label) {
			t.Fatalf("%q should fail: %v", label, failed)
		}
	}
}

func TestDates(t *testing.T) {
	equal(t, cvdoc.CvDates("2022-03", ""), "03/2022 – Present")
	equal(t, cvdoc.CvDates("2022-03", "2023-11"), "03/2022 – 11/2023")
	equal(t, cvdoc.CvDates("", "2023-11"), "")
	equal(t, cvdoc.CvDates("2022", ""), "2022 – Present")
	equal(t, cvdoc.YearsOfExperience(sampleProfile(t), time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)), 5)
	equal(t, cvdoc.YearsOfExperience(sampleProfile(t), time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)), 6)
	equal(t, cvdoc.YearsOfExperience(models.Profile{}, now), 0)
}

func TestAtsChecksPassACompleteCVAndFailUndatedRoles(t *testing.T) {
	job := engineJob(t, 0)
	document := models.CvDocument{
		Headline: "x", Summary: "Snowflake Administrator DevOps using " + strings.Join(job.Keywords, ", ") + ".",
		Skills: job.Keywords, Experience: []models.RoleSection{{Role: "r", Bullets: []string{"Built 3 things."}}},
	}
	confirmed := request(func(r *models.CvRequest) { r.Confirmed = []int{1} })
	for _, check := range cvdoc.AtsChecks(document, job, sampleProfile(t), confirmed, now) {
		if !check.OK {
			t.Fatalf("failed: %s", check.Label)
		}
	}
	undated := sampleProfile(t)
	undated.Experience = undated.Experience[:1]
	undated.Experience[0].Start = ""
	for _, check := range cvdoc.AtsChecks(document, job, undated, confirmed, now) {
		if !check.OK {
			equal(t, check.Label, "Dates with month and year")
			return
		}
	}
	t.Fatal("an undated role fails")
}

func TestRewritePreviewRewritesTheFirstHighlightOfTheFirstRoleKept(t *testing.T) {
	equal(t, cvdoc.RewritePreview(sampleProfile(t), engineJob(t, 0), request()), &models.Preview{
		From: "Built and maintained deployment pipelines for internal services.",
		To:   "Developed and maintained CI/CD pipelines for internal services, using Git.",
	})
	skipped := cvdoc.RewritePreview(sampleProfile(t), engineJob(t, 0), request(func(r *models.CvRequest) { r.SkipRoles = []string{"r1"} }))
	equal(t, skipped.From, "Maintained servers and user accounts for a small office.")
}

func TestFileName(t *testing.T) {
	equal(t, cvdoc.FileName("Andrei Popescu", "Capgemini Romania"), "Andrei_Popescu_CV_Capgemini_Romania.pdf")
	equal(t, cvdoc.FileName(" Ana-Maria  Ștefan ", "A&B Co."), "Ana_Maria_tefan_CV_A_B_Co_.pdf")
	equal(t, cvdoc.FileName("  ", "Capgemini Romania"), "CV_Capgemini_Romania.pdf")
}

func TestReadJobDefaultsTheProductAndKeepsFourKeyResponsibilities(t *testing.T) {
	job := catalogJob(t, 0)
	job.Product = ""
	read := cvdoc.ReadJob(job)
	equal(t, read.Product, "products")
	equal(t, read.KeyResponsibilities, []int{0, 1, 2, 3})
}

func checkStartingWith(t *testing.T, checks []models.AtsCheck, prefix string) models.AtsCheck {
	t.Helper()
	for _, check := range checks {
		if strings.HasPrefix(check.Label, prefix) {
			return check
		}
	}
	t.Fatalf("no check starts with %q", prefix)
	return models.AtsCheck{}
}

func TestAtsChecksAcceptPresentTenseVerbsAndPunctuatedTitles(t *testing.T) {
	job := catalogJob(t, 0)
	job.Title = "DevOps Engineer- Senior Associate- Public Sector"
	job.Responsibilities = nil
	read := cvdoc.ReadJob(job)
	document := models.CvDocument{
		Summary:    "Applying as DevOps Engineer (Senior Associate, Public Sector) with " + strings.Join(read.Keywords, ", ") + ".",
		Skills:     read.Keywords,
		Experience: []models.RoleSection{{Role: "r", Bullets: []string{"Run 3 clusters.", "Keeps 2 backups.", "Operate the VPN.", "Wrote docs.", "Mitigates attacks."}}},
	}
	for _, check := range cvdoc.AtsChecks(document, read, sampleProfile(t), request(), now) {
		if !check.OK {
			t.Errorf("failed: %s", check.Label)
		}
	}
	document.Experience[0].Bullets = append(document.Experience[0].Bullets, "The VPN was set up.")
	if check := checkStartingWith(t, cvdoc.AtsChecks(document, read, sampleProfile(t), request(), now), "Every bullet"); check.OK {
		t.Fatal("a bullet without a verb fails")
	}
	equal(t, checkStartingWith(t, cvdoc.AtsChecks(document, read, sampleProfile(t), request(), now), "Job responsibilities").Label, "Job responsibilities mirrored: 0 of 0")
}

func TestOnePageIsEstimatedInLines(t *testing.T) {
	short := models.CvDocument{Summary: "Short.", Skills: []string{"Go"}, Experience: []models.RoleSection{{Role: "r", Meta: "m", Bullets: []string{"Built it."}}},
		Projects: []models.ProjectSection{{Name: "p", Bullets: []string{"Ran it."}}}, Education: []models.EducationLine{{Degree: "BSc"}},
		Certifications: []string{"CKA"}, Languages: "English (C1)", Additional: []string{"Speaks."}}
	equal(t, cvdoc.EstimatedLines(short), 2+2+2+4+3+2+2+2+2)
	long := short
	long.Experience = []models.RoleSection{{Role: "r", Meta: "m", Bullets: []string{strings.Repeat("x", 4000)}}}
	equal(t, cvdoc.EstimatedLines(long) > cvdoc.PageLines, true)
	if check := checkStartingWith(t, cvdoc.AtsChecks(long, engineJob(t, 0), sampleProfile(t), request(), now), "Fits on one page"); check.OK {
		t.Fatal("a long CV does not fit")
	}
}

func TestExperiencedPeopleGetTwoPages(t *testing.T) {
	profile := sampleProfile(t)
	equal(t, cvdoc.PageLimit(profile, now), 1)
	profile.Experience[1].Start = "2016-01"
	equal(t, cvdoc.PageLimit(profile, now), 2)
	long := models.CvDocument{Summary: "x", Experience: []models.RoleSection{{Role: "r", Bullets: make([]string, 70)}}}
	for i := range long.Experience[0].Bullets {
		long.Experience[0].Bullets[i] = "Built it."
	}
	check := checkStartingWith(t, cvdoc.AtsChecks(long, engineJob(t, 0), profile, request(), now), "Fits on")
	equal(t, check.Label, "Fits on two pages")
	equal(t, check.OK, true)
}

func TestTailoredSkillsStopAtTwentyButKeepEveryMatch(t *testing.T) {
	profile := sampleProfile(t)
	for i := range 30 {
		profile.Skills = append(profile.Skills, fmt.Sprintf("Skill%d", i))
	}
	cv := cvdoc.Write(profile, engineJob(t, 0), request(), now)
	equal(t, len(cv.Tailored.Skills), cvdoc.MaxSkills)
	equal(t, cv.Tailored.Skills[:5], []string{"SQL", "Git", "CI/CD", "Monitoring", "Python"})
	equal(t, len(cv.Profile.Skills), len(profile.Skills))
}

func TestMatchKnowsOtherNamesUmbrellasAndCloseTools(t *testing.T) {
	job := catalogJob(t, 0)
	job.MustHave = []string{"CI/CD", "Containers", "Git", "ArgoCD", "GCP", "GraphQL", "k8s"}
	job.NiceToHave = []string{"Redis"}
	match := cvdoc.MatchJob(cvdoc.ReadJob(job), []string{"GitHub Actions", "Docker", "FluxCD", "AWS", "Kubernetes", "Valkey"})
	equal(t, match.Matched, []string{"CI/CD", "Containers", "Git", "k8s"})
	equal(t, match.Related, []cvdoc.Related{{Skill: "ArgoCD", Via: []string{"FluxCD"}, Category: "GitOps"},
		{Skill: "GCP", Via: []string{"AWS"}, Category: "cloud infrastructure"}, {Skill: "Redis", Via: []string{"Valkey"}, Category: "in-memory caching"}})
	equal(t, match.Missing, []string{"GraphQL"})
	// 4 matched must-haves (16) + 2 related must-haves (2 + 2) + a related nice-to-have (1), of 30.
	equal(t, match.Score, 70)
	equal(t, cvdoc.MatchJob(cvdoc.ReadJob(job), []string{"Argo CD"}).Matched, []string{"CI/CD", "ArgoCD"})
}

func TestSkillMapCopiesAreIdentical(t *testing.T) {
	ours, err := os.ReadFile("../../internal/cvdoc/skillmap.json")
	if err != nil {
		t.Fatal(err)
	}
	theirs, err := os.ReadFile("../../../../apps/frontend/src/features/jobs/skillmap.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(ours) != string(theirs) {
		t.Fatal("apps/frontend/src/features/jobs/skillmap.json must be a copy of services/cv/internal/cvdoc/skillmap.json")
	}
}

func TestTitleCheckLooksForTheRoleWithoutItsLevel(t *testing.T) {
	read := func(title string) cvdoc.Job {
		job := catalogJob(t, 0)
		job.Title = title
		return cvdoc.ReadJob(job)
	}
	title := func(document models.CvDocument, job cvdoc.Job) bool {
		return checkStartingWith(t, cvdoc.AtsChecks(document, job, sampleProfile(t), request(), now), "Target job title").OK
	}
	devops := models.CvDocument{Headline: "DevOps Engineer", Summary: "Runs EKS."}
	equal(t, title(devops, read("DevOps Engineer- Senior Associate- Public Sector")), true)
	equal(t, title(devops, read("Senior DevOps Engineer (Platform)")), true)
	equal(t, title(devops, read("Senior / Staff Fullstack Engineer")), false)
	equal(t, title(models.CvDocument{Summary: "A fullstack engineer who ships."}, read("Senior / Staff Fullstack Engineer")), true)
}

func TestVerbCheckFailsOnlyOnStartsThatAreNotActions(t *testing.T) {
	verbs := func(bullets ...string) bool {
		document := models.CvDocument{Experience: []models.RoleSection{{Role: "r", Bullets: bullets}}}
		return checkStartingWith(t, cvdoc.AtsChecks(document, engineJob(t, 0), sampleProfile(t), request(), now), "Every bullet").OK
	}
	equal(t, verbs("Store secrets in Vault.", "Harden hosts.", "Built it.", "Ran it.", "Spearheaded it."), true)
	for _, bad := range []string{"Responsible for servers.", "Monitoring with Grafana.", "The VPN was set up.", "Helped the team.", "I fixed it."} {
		if verbs(bad) {
			t.Errorf("%q does not start with an action", bad)
		}
	}
}
