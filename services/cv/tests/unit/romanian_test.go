package unit_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"cv/internal/cvdoc"
	"cv/internal/grounding"
	"cv/internal/models"
	"cv/internal/render"
	"cv/internal/style"
)

func TestRomanianCVFromTheRules(t *testing.T) {
	romanian := request(func(r *models.CvRequest) { r.Language = cvdoc.Romanian })
	written := cvdoc.Write(sampleProfile(t), engineJob(t, 1), romanian, time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC))
	equal(t, written.Tailored.Language, cvdoc.Romanian)
	equal(t, written.Profile.Language, cvdoc.Romanian)
	if !strings.Contains(written.Tailored.Summary, " cu ") || !strings.Contains(written.Tailored.Summary, "Îmi doresc să mă alătur echipei ") ||
		strings.Contains(written.Tailored.Summary, "Looking to join") {
		t.Fatalf("a Romanian summary: %q", written.Tailored.Summary)
	}
	english := cvdoc.Write(sampleProfile(t), engineJob(t, 1), request(), time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC))
	equal(t, english.Tailored.Language, cvdoc.English)
	equal(t, strings.Contains(english.Tailored.Summary, "Looking to join"), true)
	equal(t, cvdoc.LanguageOf("fr"), cvdoc.English)
	equal(t, cvdoc.HeadingsFor(cvdoc.Romanian).Experience, "Experiență profesională")
	equal(t, cvdoc.HeadingsFor("").Experience, "Experience")
}

func TestRomanianHeadingsInThePDFAndTheWordDocument(t *testing.T) {
	document := sampleDocument(t)
	document.Language = cvdoc.Romanian
	page, err := render.HTML(document, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<html lang="ro">`, "<h2>Profil</h2>", "<h2>Competențe</h2>", "<h2>Experiență profesională</h2>", "<h2>Educație</h2>"} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
	if strings.Contains(page, "<h2>Experience</h2>") {
		t.Fatal("no English heading on a Romanian CV")
	}
	english, err := render.HTML(sampleDocument(t), "")
	if err != nil || !strings.Contains(english, `<html lang="en">`) || !strings.Contains(english, "<h2>Experience</h2>") {
		t.Fatalf("an English CV keeps its headings: %v", err)
	}
	raw, err := render.DOCX(document, "")
	if err != nil {
		t.Fatal(err)
	}
	if body := docxPart(t, raw, "word/document.xml"); !strings.Contains(body, "Experiență profesională") || strings.Contains(body, ">Experience<") {
		t.Fatal("the Word document has the Romanian headings")
	}
}

func TestRomanianLetterGreetingsAndSignOffs(t *testing.T) {
	c := letterCatalogue(t)
	answer, err := grounding.ParseLetter(letterOutput)
	if err != nil {
		t.Fatal(err)
	}
	texts, _ := c.AssembleLetter(answer, grounding.Addressing{Person: "Ana Pop", Recruiter: "Ioana Ionescu", Company: "BRD", Title: "Contabil",
		Language: cvdoc.Romanian})
	if !strings.HasPrefix(texts.CoverLetter, "Bună ziua, Ioana Ionescu,\n\n") || !strings.HasSuffix(texts.CoverLetter, "\n\nCu stimă,\nAna Pop") {
		t.Fatalf("letter: %q", texts.CoverLetter)
	}
	equal(t, strings.HasPrefix(texts.ConnectionNote, "Bună ziua, Ioana, "), true)
	equal(t, texts.Message.Subject, "Candidatură pentru Contabil")
	equal(t, strings.HasPrefix(texts.Message.Body, "Bună ziua, Ioana,\n\n"), true)
	equal(t, strings.HasSuffix(texts.Message.Body, "Cu stimă,\nAna Pop"), true)
	team, _ := c.AssembleLetter(answer, grounding.Addressing{Company: "BRD", Language: cvdoc.Romanian})
	equal(t, strings.HasPrefix(team.CoverLetter, "Stimată echipă BRD,\n\n"), true)
	anyone, _ := c.AssembleLetter(answer, grounding.Addressing{Language: cvdoc.Romanian})
	equal(t, strings.HasPrefix(anyone.CoverLetter, "Stimată echipă de recrutare,"), true)
	equal(t, strings.HasPrefix(anyone.Message.Body, "Bună ziua,\n\n"), true)
}

func TestRomanianStyleRules(t *testing.T) {
	letter := models.LetterTexts{CoverLetter: "Stimată echipă BRD,\n\nSunt pasionată de contabilitate și am redus închiderea lunii la 3 zile.\n\nCu stimă,\nAna Pop"}
	findings := style.LintLetter(letter, nil).Findings
	if len(findings) == 0 {
		t.Fatal("the cliché is found")
	}
	for _, finding := range findings {
		if finding.Rule != style.Stock || finding.Text != "pasionată" {
			t.Fatalf("the greeting and the sign-off are skipped, only the cliché is found: %+v", findings)
		}
	}
	document := models.CvDocument{Summary: "Contabil cu 5 ani de experiență în închideri lunare și raportare.", Language: cvdoc.Romanian}
	if report := style.Lint(document, nil); !hasKind(report.Findings, style.Formula) {
		t.Fatalf("the Romanian formula is found: %+v", report.Findings)
	}
}

func TestClaudeIsToldTheLanguage(t *testing.T) {
	c, job := catalogue(t, func(r *models.CvRequest) { r.Language = cvdoc.Romanian })
	var message struct {
		Language string `json:"language"`
	}
	if err := json.Unmarshal([]byte(c.Prompt(job, request(func(r *models.CvRequest) { r.Language = cvdoc.Romanian }), nil, nil)), &message); err != nil {
		t.Fatal(err)
	}
	equal(t, message.Language, cvdoc.Romanian)
	if err := json.Unmarshal([]byte(c.LetterPrompt(nil, job, models.CvDocument{Language: cvdoc.Romanian}, nil)), &message); err != nil {
		t.Fatal(err)
	}
	equal(t, message.Language, cvdoc.Romanian)
	if err := json.Unmarshal([]byte(c.Prompt(job, request(), nil, nil)), &message); err != nil {
		t.Fatal(err)
	}
	equal(t, message.Language, cvdoc.English)
	if !strings.Contains(grounding.System, "`language`") || !strings.Contains(grounding.LetterSystem, "`language`") ||
		!strings.Contains(grounding.ReviewSystem, "`language`") {
		t.Fatal("every writing prompt explains the language")
	}
}

func hasKind(findings []style.Finding, kind string) bool {
	for _, finding := range findings {
		if finding.Rule == kind {
			return true
		}
	}
	return false
}

func TestARefusedSentenceOfARomanianCVIsLeftOutNotWrittenInEnglish(t *testing.T) {
	romanian := func(r *models.CvRequest) { r.Language = cvdoc.Romanian }
	c, job := catalogue(t, romanian)
	rules := cvdoc.Write(richProfile(t), job, request(romanian), now).Tailored
	document, refused := c.Assemble(syntheticAnswer(t), job, rules)
	equal(t, len(refused), 2)
	equal(t, document.Experience[0].Bullets, []string{"Developed and maintained CI/CD pipelines for internal services with Git."})
	equal(t, document.Language, cvdoc.Romanian)
}
