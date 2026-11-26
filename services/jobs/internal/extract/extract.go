// Package extract reads a job ad with rules and a skills dictionary: sections
// by their headings, skills, years, city, work mode, level, title, company and
// the about text. It never calls a model; the same text always reads the same.
package extract

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Limits that keep a draft valid for POST /jobs.
const (
	MaxTitle            = 160
	MaxCompany          = 120
	MaxSkills           = 40
	MaxResponsibilities = 30
	MaxResponsibility   = 400
	MaxAbout            = 4000
	maxHeading          = 60
	minAboutLine        = 40
	maxYears            = 40
)

// Hints are values the caller already knows; they win over what the rules read.
type Hints struct {
	Title   string
	Company string
}

// Result is what the rules read from an ad. YearsRequired is nil when the ad
// names no years.
type Result struct {
	Title            string
	Company          string
	City             string
	Country          string
	WorkMode         string
	Level            string
	YearsRequired    *int
	MustHave         []string
	NiceToHave       []string
	Responsibilities []string
	About            string
	// Qualifications and Bonuses are the lines of the requirements and nice-to-have sections, as the ad words them.
	Qualifications []string
	Bonuses        []string
	// AboutCompany and Perks are what the ad says about the company and what it offers.
	AboutCompany string
	Perks        []string
}

type section int

const (
	body section = iota
	requirements
	niceToHave
	responsibilities
	aboutCompany
	perks
	ignored
)

// headings are folded heading starts; "about {company}" is added per ad.
var headings = []struct {
	section  section
	prefixes []string
}{
	{requirements, []string{"requirements", "minimum qualifications", "qualifications", "what we look for",
		"what we're looking for", "what you bring", "what you'll bring", "what you'll need", "what you need", "what we require",
		"what makes you a great fit", "who you are",
		"about you", "you have", "skills and experience", "must have", "must-have", "cerinte", "ce cautam",
		"profilul candidatului", "ce aduci"}},
	{niceToHave, []string{"nice to have", "nice-to-have", "bonus", "preferred", "it's a plus", "extra points",
		"constituie un avantaj", "ar fi un plus", "reprezinta un avantaj"}},
	{responsibilities, []string{"responsibilities", "key responsibilities", "what you will do", "what you'll do",
		"what you'll be doing", "what you will be doing", "your role", "your responsibilities", "in this role", "the role",
		"responsabilit", "ce vei face", "atributii"}},
	{aboutCompany, []string{"about us", "about the company", "company description", "who we are", "despre noi", "despre companie"}},
	{perks, []string{"benefits", "what we offer", "perks", "why you'll", "why join", "ce oferim", "beneficii"}},
	{ignored, []string{"our culture", "equal opportunity", "apply for this job", "how to apply"}},
}

// normalised turns curly quotes and apostrophes into straight ones and
// non-breaking spaces into spaces.
var normalised = strings.NewReplacer(
	"\u2018", "'", "\u2019", "'", "\u201a", "'", "\u201b", "'", "\u02bc", "'", "\u2032", "'",
	"\u201c", `"`, "\u201d", `"`, "\u201e", `"`, "\u201f", `"`,
	"\u00a0", " ", "\u202f", " ", "\u2007", " ",
)

var (
	listMarker   = regexp.MustCompile(`^(?:[-•·–*]|\d{1,2}[.)])\s*`)
	bulletLine   = regexp.MustCompile(`^[-•·–]\s`)
	yearsPattern = regexp.MustCompile(`(?:^|[^\d])(\d{1,2})\s*\+?\s*(?:years?|yrs|ani)\b`)
	companyLine  = regexp.MustCompile(`(?i)\b(?:company|firma|angajator)\s*:\s*([^,\n·(]+)`)
	seniorTitle  = regexp.MustCompile(`(?i)\b(?:senior|sr)\b`)
	juniorTitle  = regexp.MustCompile(`(?i)\b(?:junior|jr|entry)\b`)
	leadTitle    = regexp.MustCompile(`(?i)\b(?:lead|principal|staff|head)\b`)
)

// Read applies the rules to text.
func Read(text string, hints Hints) Result {
	lines := splitLines(normalised.Replace(text))
	companyText := ""
	if m := companyLine.FindStringSubmatch(strings.Join(lines, "\n")); m != nil {
		companyText = strings.TrimSpace(m[1])
	}
	company := FirstOf(strings.TrimSpace(hints.Company), companyText)

	parts := map[section][]string{}
	var mustLines, kept []string
	sectioned := false
	current := body
	for _, line := range lines {
		if s, ok := heading(line, company); ok {
			current, sectioned = s, true
			if s < aboutCompany {
				kept = append(kept, line)
			}
			continue
		}
		if current == ignored {
			continue
		}
		// What the ad says about the company and its perks feeds no other field: a city or a skill named there is
		// not the job's.
		if current == aboutCompany || current == perks {
			parts[current] = append(parts[current], listMarker.ReplaceAllString(line, ""))
			continue
		}
		kept = append(kept, line)
		line = listMarker.ReplaceAllString(line, "")
		parts[current] = append(parts[current], line)
		if current == requirements || current == responsibilities {
			mustLines = append(mustLines, line)
		}
	}

	all := strings.Join(kept, "\n")
	result := Result{WorkMode: WorkMode(all)}
	result.City, result.Country = Place(all)

	result.Title = strings.TrimSpace(hints.Title)
	titleFromText := result.Title == ""
	if titleFromText && len(lines) > 0 {
		result.Title = lines[0]
	}
	result.Title = Clip(result.Title, MaxTitle)

	result.Company = Clip(company, MaxCompany)

	if m := yearsPattern.FindStringSubmatch(strings.ToLower(all)); m != nil {
		if years, err := strconv.Atoi(m[1]); err == nil && years <= maxYears {
			result.YearsRequired = &years
		}
	}
	result.Level = level(result.Title, result.YearsRequired)

	dictionary := Skills()
	nice := dictionary.Find(strings.Join(parts[niceToHave], "\n"))
	var must []string
	if sectioned {
		must = dictionary.Find(strings.Join(mustLines, "\n"))
	} else {
		must = dictionary.Find(all)
	}
	must = slices.DeleteFunc(must, func(name string) bool { return slices.Contains(nice, name) })
	result.MustHave = limit(must, MaxSkills)
	result.NiceToHave = limit(nice, MaxSkills)

	result.Responsibilities = listOf(parts[responsibilities])
	result.Qualifications = listOf(parts[requirements])
	result.Bonuses = listOf(parts[niceToHave])
	result.Perks = listOf(parts[perks])
	result.AboutCompany = Clip(strings.Join(listOf(parts[aboutCompany]), " "), MaxAbout)

	var about []string
	for i, line := range parts[body] {
		isTitle := i == 0 && len(lines) > 0 && line == lines[0] && (titleFromText || Fold(line) == Fold(result.Title))
		isCompany := companyText != "" && companyLine.MatchString(line)
		if !isTitle && !isCompany && utf8.RuneCountInString(line) > minAboutLine {
			about = append(about, line)
		}
	}
	result.About = Clip(strings.Join(about, " "), MaxAbout)
	return result
}

func splitLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// listOf keeps a section's lines that say something, at most MaxResponsibilities, each clipped.
func listOf(lines []string) []string {
	kept := []string{}
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" && len(kept) < MaxResponsibilities {
			kept = append(kept, Clip(line, MaxResponsibility))
		}
	}
	return kept
}

// heading reports the section a heading line opens. A heading is shorter than
// 60 characters, may start with "#" or "*" and end with ":", and starts with a
// known heading (so "Cerințele postului" opens the requirements), or with
// "About {company}"; list items are never headings.
func heading(line, company string) (section, bool) {
	if bulletLine.MatchString(line) {
		return body, false
	}
	bare := strings.TrimRight(strings.TrimLeft(line, "#* \t"), ":* \t")
	if utf8.RuneCountInString(bare) >= maxHeading {
		return body, false
	}
	folded := Fold(bare)
	if company != "" && strings.HasPrefix(folded, "about "+Fold(company)) {
		return aboutCompany, true
	}
	for _, h := range headings {
		for _, prefix := range h.prefixes {
			if strings.HasPrefix(folded, prefix) {
				return h.section, true
			}
		}
	}
	return body, false
}

// cities lists Romanian cities with the spellings that name them, folded.
var cities = []struct {
	name     string
	spelling []string
}{
	{"Iași", []string{"iasi"}},
	{"Bucharest", []string{"bucharest", "bucuresti"}},
	{"Cluj-Napoca", []string{"cluj-napoca", "cluj napoca", "cluj"}},
	{"Timișoara", []string{"timisoara"}},
	{"Brașov", []string{"brasov"}},
	{"Constanța", []string{"constanta"}},
	{"Craiova", []string{"craiova"}},
	{"Galați", []string{"galati"}},
	{"Oradea", []string{"oradea"}},
	{"Sibiu", []string{"sibiu"}},
	{"Arad", []string{"arad"}},
	{"Ploiești", []string{"ploiesti"}},
	{"Târgu Mureș", []string{"targu mures", "targu-mures", "tirgu mures", "tirgu-mures", "tg. mures", "tg mures"}},
}

var cityPatterns = func() []*regexp.Regexp {
	patterns := make([]*regexp.Regexp, len(cities))
	for i, c := range cities {
		quoted := make([]string, len(c.spelling))
		for j, s := range c.spelling {
			quoted[j] = regexp.QuoteMeta(s)
		}
		patterns[i] = regexp.MustCompile(`(?:^|[^\p{L}\p{N}])(?:` + strings.Join(quoted, "|") + `)(?:$|[^\p{L}\p{N}])`)
	}
	return patterns
}()

var romaniaWord = regexp.MustCompile(`(?:^|[^\p{L}])romania(?:$|[^\p{L}])`)

// Place returns the Romanian city mentioned first in text and the country:
// Romania when a Romanian city or the word Romania appears.
func Place(text string) (city, country string) {
	folded := Fold(text)
	first := -1
	for i, pattern := range cityPatterns {
		if m := pattern.FindStringIndex(folded); m != nil && (first < 0 || m[0] < first) {
			first, city = m[0], cities[i].name
		}
	}
	if city != "" || romaniaWord.MatchString(folded) {
		country = "Romania"
	}
	return city, country
}

// CityName returns the canonical name of a Romanian city spelling (folded
// comparison), or "" for any other value.
func CityName(value string) string {
	folded := strings.TrimSpace(Fold(value))
	for _, c := range cities {
		if slices.Contains(c.spelling, folded) {
			return c.name
		}
	}
	return ""
}

var (
	hybridMode = regexp.MustCompile(`(?:^|[^\p{L}])(?:hybrid|hibrid)`)
	onsiteMode = regexp.MustCompile(`(?:^|[^\p{L}])(?:on-?site|on site|la birou|office-based)`)
	remoteMode = regexp.MustCompile(`(?:^|[^\p{L}])(?:remote|de acasa)`)
)

// WorkMode reads the work mode a text names: hybrid, else onsite, else
// remote, else "".
func WorkMode(text string) string {
	folded := Fold(text)
	switch {
	case hybridMode.MatchString(folded):
		return "hybrid"
	case onsiteMode.MatchString(folded):
		return "onsite"
	case remoteMode.MatchString(folded):
		return "remote"
	}
	return ""
}

func level(title string, years *int) string {
	switch {
	case seniorTitle.MatchString(title):
		return "senior"
	case juniorTitle.MatchString(title):
		return "junior"
	case leadTitle.MatchString(title):
		return "lead"
	case years == nil:
		return ""
	case *years >= 5:
		return "senior"
	case *years <= 1:
		return "junior"
	}
	return "mid"
}

// FirstOf returns the first value that is not empty.
func FirstOf(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// IsRomania reports whether a country, as a name or an ISO code, is Romania: RO, ROU, Romania or România.
func IsRomania(country string) bool {
	folded := Fold(strings.TrimSpace(country))
	return folded == "ro" || folded == "rou" || folded == "romania"
}

func limit(values []string, n int) []string {
	return values[:min(len(values), n)]
}

// Clip cuts value to at most n characters.
func Clip(value string, n int) string {
	if utf8.RuneCountInString(value) <= n {
		return value
	}
	return strings.TrimSpace(string([]rune(value)[:n]))
}
