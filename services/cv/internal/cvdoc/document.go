package cvdoc

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"cv/internal/models"
)

var monthPattern = regexp.MustCompile(`^\d{4}-\d{2}$`)

// monthLabel prints "2022-03" as "03/2022".
func monthLabel(value string) string {
	parts := strings.SplitN(value, "-", 3)
	if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
		return parts[1] + "/" + parts[0]
	}
	return value
}

// CvDates is "03/2022 – Present"; nothing without a start.
func CvDates(start, end string) string {
	if start == "" {
		return ""
	}
	to := "Present"
	if end != "" {
		to = monthLabel(end)
	}
	return monthLabel(start) + " – " + to
}

func yearRange(start, end string) string {
	if start == "" && end == "" {
		return ""
	}
	if start == "" {
		start = "?"
	}
	if end == "" {
		end = "now"
	}
	return start + " – " + end
}

var educationLevels = map[string]string{
	"high_school": "High school",
	"associate":   "Associate degree",
	"bachelor":    "Bachelor’s degree",
	"master":      "Master’s degree",
	"doctorate":   "Doctorate",
	"other":       "Other",
}

// educationTitle is the degree's own title, else its level and field.
func educationTitle(item models.Education) string {
	if strings.TrimSpace(item.Degree) != "" {
		return item.Degree
	}
	if title := strings.Join(keepNonEmpty(educationLevels[item.Level], strings.TrimSpace(item.Field)), " in "); title != "" {
		return title
	}
	return "Education"
}

func monthIndex(value string) int {
	var year, month int
	fmt.Sscanf(value, "%d-%d", &year, &month)
	return year*12 + month - 1
}

// YearsOfExperience counts whole years from the earliest role's start month to now.
func YearsOfExperience(profile models.Profile, now time.Time) int {
	earliest := -1
	for _, role := range profile.Experience {
		if monthPattern.MatchString(role.Start) {
			if at := monthIndex(role.Start); earliest < 0 || at < earliest {
				earliest = at
			}
		}
	}
	if earliest < 0 {
		return 0
	}
	return max(0, (now.Year()*12+int(now.Month())-1-earliest)/12)
}

type built struct {
	document models.CvDocument
	// matched are the keywords the tailored CV is written around.
	matched []string
}

type rankedBullet struct {
	text  string
	hits  int
	index int
}

// buildDocument makes one CV from a profile snapshot: as the profile reads, or
// tailored: rewritten in the job's words around the skills it asks for.
func buildDocument(profile models.Profile, job Job, request models.CvRequest, tailored bool, now time.Time) built {
	matched := MatchJob(job, KnownSkills(profile)).Matched
	var keywords []string
	if tailored {
		keywords = matched
	}
	rewrite := func(bullets []models.Bullet, tools []string) []rankedBullet {
		used := map[string]bool{}
		ranked := make([]rankedBullet, 0, len(bullets))
		for index, bullet := range bullets {
			text := CleanFact(BulletLine(bullet), "")
			if tailored {
				text = RewriteBullet(text, index, RewriteContext{Job: job, Matched: matched, Tools: tools, Used: used})
			}
			ranked = append(ranked, rankedBullet{text: text, hits: countKeywordHits(text, keywords), index: index})
		}
		return ranked
	}

	top := matched
	if len(request.Focus) > 0 {
		top = []string{}
		for _, skill := range request.Focus {
			if slices.Contains(matched, skill) {
				top = append(top, skill)
			}
		}
	} else if len(top) > 3 {
		top = top[:3]
	}
	summary := profile.About
	headline := profile.Headline
	skills := append([]string{}, profile.Skills...)
	if tailored {
		years := YearsOfExperience(profile, now)
		experience := "hands-on experience"
		if years > 0 {
			experience = fmt.Sprintf("%d years of experience", years)
		}
		strongest := ""
		if len(top) > 0 {
			strongest = ", strongest in " + JoinList(top)
		}
		title := profile.Headline
		if title == "" {
			title = "Engineer"
		}
		sentences := []string{}
		for _, sentence := range strings.Split(profile.About, ".") {
			if sentence = strings.TrimSpace(sentence); sentence != "" {
				sentences = append(sentences, sentence)
			}
		}
		second := ""
		if len(sentences) > 1 {
			second = sentences[1] + "."
		}
		opening, joining := title+" with "+experience+strongest+".", "Looking to join "+job.Company+" as "+job.Title+" and work on its "+job.Product+"."
		if LanguageOf(request.Language) == Romanian {
			opening, joining = romanianSummary(profile.Headline, years, top, job)
		}
		summary = strings.Join(keepNonEmpty(opening, second, joining), " ")
		skills = append([]string{}, matched...)
		for _, skill := range profile.Skills {
			if !containsFold(matched, skill) {
				skills = append(skills, skill)
			}
		}
		skills = skills[:min(len(skills), max(MaxSkills, len(matched)))]
	}

	experience := []models.RoleSection{}
	kept := 0
	for _, role := range profile.Experience {
		if tailored && slices.Contains(request.SkipRoles, role.ID) {
			continue
		}
		bullets := rewrite(role.Bullets, role.Tools)
		if tailored {
			digits := func(text string) int {
				if strings.ContainsAny(text, "0123456789") {
					return 1
				}
				return 0
			}
			sort.SliceStable(bullets, func(i, j int) bool {
				a, b := bullets[i], bullets[j]
				if a.hits != b.hits {
					return a.hits > b.hits
				}
				if digits(a.text) != digits(b.text) {
					return digits(a.text) > digits(b.text)
				}
				return a.index < b.index
			})
			limit := 3
			if kept == 0 {
				limit = 7
			}
			bullets = bullets[:min(limit, len(bullets))]
		}
		kept++
		experience = append(experience, models.RoleSection{
			Role:    role.Role,
			Meta:    strings.Join(keepNonEmpty(role.Company, role.Location, CvDates(role.Start, role.End)), " | "),
			Bullets: texts(bullets),
		})
	}
	projects := []models.ProjectSection{}
	for _, project := range profile.Projects {
		projects = append(projects, models.ProjectSection{
			Name:    project.Name,
			Meta:    strings.Join(keepNonEmpty(project.Role, project.URL, CvDates(project.Start, project.End)), " | "),
			Bullets: texts(rewrite(project.Bullets, project.Tools)),
		})
	}
	certifications := []string{}
	for _, item := range profile.Certifications {
		issued, expires, id := "", "", ""
		if item.Issued != "" {
			issued = "Issued " + monthLabel(item.Issued)
		}
		if item.Expires != "" {
			expires = "Expires " + monthLabel(item.Expires)
		}
		if item.CredentialID != "" {
			id = "ID " + item.CredentialID
		}
		certifications = append(certifications, strings.Join(keepNonEmpty(item.Name, item.Issuer, issued, expires, id, item.URL), " | "))
	}
	education := []models.EducationLine{}
	for _, item := range profile.Education {
		years := strings.Replace(yearRange(item.StartYear, item.EndYear), "now", "Present", 1)
		education = append(education, models.EducationLine{
			Degree: educationTitle(item),
			Meta:   strings.Join(keepNonEmpty(item.School, years), " | "),
		})
	}
	languages := make([]string, 0, len(profile.Languages))
	for _, language := range profile.Languages {
		languages = append(languages, language.Name+" ("+language.Level+")")
	}
	additional := []string{}
	for _, line := range profile.AdditionalInfo {
		if fact := CleanFact(line, ""); fact != "" {
			additional = append(additional, fact)
		}
	}
	return built{
		matched: matched,
		document: models.CvDocument{
			Name:           strings.TrimSpace(profile.Name),
			Headline:       headline,
			Contact:        keepNonBlank(profile.Email, profile.Phone, strings.Join(keepNonBlank(profile.City, profile.Country), ", ")),
			Links:          keepNonBlank(profile.GitHub, profile.LinkedIn),
			Summary:        summary,
			Skills:         nonNil(skills),
			Experience:     experience,
			Projects:       projects,
			Education:      education,
			Certifications: certifications,
			Languages:      strings.Join(languages, ", "),
			Additional:     additional,
			Language:       LanguageOf(request.Language),
		},
	}
}

func texts(bullets []rankedBullet) []string {
	out := make([]string, len(bullets))
	for i, bullet := range bullets {
		out[i] = bullet.text
	}
	return out
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

var (
	actionVerbs = []string{"built", "wrote", "made", "ran", "led", "kept", "took", "won", "grew", "found", "drove", "rebuilt", "rewrote", "taught", "oversaw", "undertook", "began", "brought", "sold", "spoke", "accelerate", "achieve", "adapt", "add", "administer", "advise", "align", "analyze", "apply", "architect", "assemble", "assess", "audit", "automate", "balance", "benchmark", "build", "cache", "calculate", "centralize", "champion", "clarify", "close", "coach", "collaborate", "compile", "configure", "consolidate", "construct", "containerize", "contribute", "convert", "coordinate", "create", "cut", "debug", "decouple", "define", "delegate", "deliver", "deploy", "design", "detect", "develop", "diagnose", "direct", "document", "draft", "drive", "earn", "edit", "eliminate", "enable", "engineer", "enhance", "establish", "evaluate", "examine", "execute", "expand", "expedite", "extend", "facilitate", "find", "fix", "forecast", "formulate", "found", "generate", "grow", "guide", "handle", "harden", "head", "identify", "implement", "improve", "increase", "influence", "initiate", "inspect", "install", "instrument", "integrate", "introduce", "investigate", "isolate", "keep", "launch", "lead", "maintain", "make", "manage", "map", "measure", "mentor", "merge", "migrate", "minimize", "mitigate", "model", "moderate", "modernize", "monitor", "motivate", "negotiate", "onboard", "open", "operate", "optimize", "orchestrate", "organize", "oversee", "own", "parallelize", "partner", "perform", "pilot", "plan", "prepare", "present", "prevent", "prioritize", "process", "produce", "profile", "program", "prototype", "provide", "provision", "publish", "raise", "rebuild", "recommend", "redesign", "reduce", "refactor", "remove", "reorganize", "replace", "report", "research", "resolve", "restore", "restructure", "review", "revise", "rewrite", "run", "save", "scale", "schedule", "script", "secure", "select", "set", "ship", "simplify", "solve", "spearhead", "standardize", "streamline", "strengthen", "structure", "supervise", "support", "take", "teach", "test", "track", "train", "transform", "translate", "troubleshoot", "tune", "unify", "update", "upgrade", "validate", "verify", "win", "write"}
	leadWord    = regexp.MustCompile(`^(\w+)`)
	nonAlnum    = regexp.MustCompile(`[^\pL\pN]+`)
)

// MaxSkills is how many skills the tailored CV lists: the job's first, then the person's own.
const MaxSkills = 20

// PageLines is about how many lines of the exported CV fit on one A4 page
// (internal/render), measured on rendered CVs.
const PageLines = 50

// notVerbs are first words that show a bullet does not start with an action.
var notVerbs = []string{"a", "an", "the", "i", "my", "me", "we", "our", "he", "she", "they", "it", "this", "that", "these",
	"responsible", "duties", "duty", "tasks", "task", "member", "part", "involved", "various", "experience", "knowledge",
	"strong", "excellent", "good", "great", "successful", "successfully", "worked", "work", "helped", "help", "assisted",
	"participated", "was", "were", "is", "are", "been", "being", "had", "have", "has", "team", "role", "in", "on",
	"for", "with", "as", "to", "of", "from", "by", "and", "or"}

// isActionVerb reports whether a bullet's first word is an action verb: any
// word but the ones a bullet should not start with ("The", "Responsible",
// "Helped") and -ing forms ("Monitoring").
func isActionVerb(word string) bool {
	word = lc(word)
	if word == "" || slices.Contains(notVerbs, word) {
		return false
	}
	if strings.HasSuffix(word, "ing") && !slices.Contains(actionVerbs, word) {
		return false
	}
	return true
}

// words is text as lower-case words, punctuation dropped: "DevOps Engineer- Senior" is "devops engineer senior".
func words(text string) string {
	return strings.TrimSpace(nonAlnum.ReplaceAllString(lc(text), " "))
}

// charsPerLine is about how many characters a line of the exported CV holds
// (internal/render: A4, 14 mm margins, 10 pt), measured on rendered CVs.
const charsPerLine = 110

// levelWords are the parts of a job title that say its level, not its role.
var levelWords = []string{"senior", "sr", "staff", "lead", "principal", "junior", "jr", "mid", "middle", "associate",
	"intern", "trainee", "head", "chief", "level", "i", "ii", "iii", "iv"}

// coreTitle is the role a job title names, in lower-case words, without its
// level or what follows a separator: "Senior / Staff Fullstack Engineer" is
// "fullstack engineer", "DevOps Engineer- Senior Associate- Public Sector" is
// "devops engineer".
var titleSeparator = regexp.MustCompile(`\s*(?:\s-|-\s|\(|,|\|)`)

func coreTitle(title string) string {
	if at := titleSeparator.FindStringIndex(title); at != nil {
		title = title[:at[0]]
	}
	kept := []string{}
	for _, word := range strings.Fields(words(title)) {
		if !slices.Contains(levelWords, word) {
			kept = append(kept, word)
		}
	}
	return strings.Join(kept, " ")
}

// lines is about how many lines text takes in the exported CV, width
// characters to a line.
func lines(text string, width int) int {
	if text == "" {
		return 0
	}
	return (utf16Length(text) + width - 1) / width
}

// EstimatedLines is about how many lines the document takes when exported.
func EstimatedLines(d models.CvDocument) int {
	total := 2 + min(1, len(d.Contact)) + min(1, len(d.Links))
	total += 1 + lines(d.Summary, charsPerLine)
	if len(d.Skills) > 0 {
		total += 1 + lines(strings.Join(d.Skills, ", "), charsPerLine)
	}
	entry := func(meta string, bullets []string) int {
		n := 1 + lines(meta, charsPerLine)
		for _, bullet := range bullets {
			n += max(1, lines(bullet, charsPerLine-4))
		}
		return n
	}
	if len(d.Experience) > 0 {
		total++
		for _, role := range d.Experience {
			total += entry(role.Meta, role.Bullets)
		}
	}
	if len(d.Projects) > 0 {
		total++
		for _, project := range d.Projects {
			total += entry(project.Meta, project.Bullets)
		}
	}
	if len(d.Education) > 0 {
		total += 1 + len(d.Education)
	}
	if len(d.Certifications) > 0 {
		total += 1 + len(d.Certifications)
	}
	if d.Languages != "" {
		total += 1 + lines(d.Languages, charsPerLine)
	}
	if len(d.Additional) > 0 {
		total += 1 + len(d.Additional)
	}
	return total
}

// PageLimit is how many pages a CV gets: one under eight years of experience,
// two from then on, the common advice for experienced people.
func PageLimit(profile models.Profile, now time.Time) int {
	if YearsOfExperience(profile, now) >= 8 {
		return 2
	}
	return 1
}

// AtsChecks is what an applicant tracking system looks for, checked on the
// final text and the profile it came from.
func AtsChecks(document models.CvDocument, job Job, profile models.Profile, request models.CvRequest, now time.Time) []models.AtsCheck {
	bullets := []string{}
	for _, entry := range document.Experience {
		bullets = append(bullets, entry.Bullets...)
	}
	for _, entry := range document.Projects {
		bullets = append(bullets, entry.Bullets...)
	}
	skillsLine := strings.Join(document.Skills, ", ")
	body := strings.Join(append([]string{document.Summary, skillsLine}, bullets...), " ")
	inSentences := strings.Join(append([]string{document.Summary}, bullets...), " ")
	found, inContext := 0, 0
	for _, keyword := range job.Keywords {
		if HasWord(body, keyword) {
			found++
		}
		if HasWord(inSentences, keyword) {
			inContext++
		}
	}
	verbs := len(bullets) > 0
	withNumbers := 0
	for _, bullet := range bullets {
		word := ""
		if lead := leadWord.FindStringSubmatch(bullet); lead != nil {
			word = lc(lead[1])
		}
		if !isActionVerb(word) {
			verbs = false
		}
		if strings.ContainsAny(bullet, "0123456789") {
			withNumbers++
		}
	}
	dated, roles := true, 0
	for _, role := range profile.Experience {
		if slices.Contains(request.SkipRoles, role.ID) {
			continue
		}
		roles++
		if !monthPattern.MatchString(role.Start) || (role.End != "" && !monthPattern.MatchString(role.End)) {
			dated = false
		}
	}
	dated = dated && roles > 0
	mirrored := map[int]bool{}
	for _, index := range request.Confirmed {
		if slices.Contains(job.KeyResponsibilities, index) {
			mirrored[index] = true
		}
	}
	check := func(id string, ok bool, label, hint string) models.AtsCheck {
		return models.AtsCheck{ID: id, OK: ok, Label: label, Hint: hint}
	}
	counted := func(id string, ok bool, n, of int, label, hint string) models.AtsCheck {
		return models.AtsCheck{ID: id, OK: ok, Label: fmt.Sprintf(label, n, of), Hint: hint, N: n, Of: of}
	}
	title := check("title", strings.Contains(" "+words(document.Headline+" "+document.Summary)+" ", " "+coreTitle(job.Title)+" "),
		"Target job title “"+job.Title+"” in the summary", "")
	title.Title = job.Title
	return []models.AtsCheck{
		check("layout", true, "Single column, plain text, no tables or images", ""),
		check("headings", true, "Standard headings: Summary, Skills, Experience, Education", ""),
		check("contact", strings.TrimSpace(profile.Email) != "" && strings.TrimSpace(profile.Phone) != "",
			"Email and phone in the body, not in a header", "Add them in Profile."),
		check("location", strings.TrimSpace(profile.City) != "" || strings.TrimSpace(profile.Country) != "",
			"Location in the header", "Add your city and country in Profile."),
		check("dates", dated, "Dates with month and year", "Add the start month of each role in Profile."),
		title,
		counted("keywords", found == len(job.Keywords), found, len(job.Keywords), "Keywords from the job: %d of %d",
			"The missing ones are not in your profile yet. Tell us about them in Prepare your CV."),
		counted("keywords_in_context", inContext >= (found*6+9)/10, inContext, found, "Keywords used in sentences, not only listed: %d of %d",
			"List the tools you used for each role in Profile."),
		counted("responsibilities", len(mirrored) > 0 || len(job.KeyResponsibilities) == 0, len(mirrored), len(job.KeyResponsibilities),
			"Job responsibilities mirrored: %d of %d", "Describe them in Prepare your CV."),
		check("verbs", verbs, "Every bullet starts with an action verb", ""),
		counted("numbers", withNumbers > 0, withNumbers, len(bullets), "Results with numbers: %d of %d",
			"Add a result to your highlights in Prepare your CV."),
		fits(document, PageLimit(profile, now)),
	}
}

func fits(document models.CvDocument, pages int) models.AtsCheck {
	label := "Fits on one page"
	if pages == 2 {
		label = "Fits on two pages"
	}
	return models.AtsCheck{ID: "pages", OK: EstimatedLines(document) <= pages*PageLines, Label: label, N: pages,
		Hint: "Leave out a role or a project in Prepare your CV."}
}

// utf16Length counts text the way the web app does, in UTF-16 code units.
func utf16Length(text string) int {
	n := 0
	for _, r := range text {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// RewritePreview is the sample shown while the CV is made: the first highlight
// of the first role kept, rewritten for the job; nil when no role has one.
func RewritePreview(profile models.Profile, job Job, request models.CvRequest) *models.Preview {
	matched := MatchJob(job, KnownSkills(profile)).Matched
	for _, role := range profile.Experience {
		if slices.Contains(request.SkipRoles, role.ID) || len(role.Bullets) == 0 {
			continue
		}
		from := CleanFact(BulletLine(role.Bullets[0]), "")
		to := RewriteBullet(from, 0, RewriteContext{Job: job, Matched: matched, Tools: role.Tools, Used: map[string]bool{}})
		return &models.Preview{From: from, To: to}
	}
	return nil
}

var notFileSafe = regexp.MustCompile(`[^A-Za-z0-9]+`)

// FileName is "<Name>_CV_<Company>.pdf", or "CV_<Company>.pdf" without a name.
func FileName(name, company string) string {
	safe := func(value string) string { return notFileSafe.ReplaceAllString(strings.TrimSpace(value), "_") }
	prefix := ""
	if strings.TrimSpace(name) != "" {
		prefix = safe(name) + "_"
	}
	return prefix + "CV_" + safe(company) + ".pdf"
}

// Write makes both versions of the CV from one snapshot of the profile and
// checks the tailored one.
func Write(profile models.Profile, job Job, request models.CvRequest, now time.Time) models.Written {
	plain := buildDocument(profile, job, request, false, now)
	tailored := buildDocument(profile, job, request, true, now)
	return models.Written{
		Highlights: tailored.matched,
		Profile:    plain.document,
		Tailored:   tailored.document,
		Ats:        Score(AtsChecks(tailored.document, job, profile, request, now)),
	}
}

// Score counts the checks that pass.
func Score(checks []models.AtsCheck) models.Ats {
	passed := 0
	for _, check := range checks {
		if check.OK {
			passed++
		}
	}
	return models.Ats{Passed: passed, Total: len(checks), Checks: checks}
}
