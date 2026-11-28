package cvdoc

import (
	"fmt"
	"strings"
)

// The languages a CV is written in: English unless the request asks for Romanian.
const (
	English  = "en"
	Romanian = "ro"
)

// LanguageOf is the language a request's value names; anything but Romanian is English.
func LanguageOf(value string) string {
	if value == Romanian {
		return Romanian
	}
	return English
}

// Headings are a CV's section titles in its language.
type Headings struct {
	Summary, Skills, Experience, Projects, Education, Certifications, Languages, Additional string
}

// HeadingsFor gives the section titles of a language, the ones recruiters and ATS parsers expect in it.
func HeadingsFor(language string) Headings {
	if LanguageOf(language) == Romanian {
		return Headings{Summary: "Profil", Skills: "Competențe", Experience: "Experiență profesională", Projects: "Proiecte",
			Education: "Educație", Certifications: "Certificări", Languages: "Limbi străine", Additional: "Informații suplimentare"}
	}
	return Headings{Summary: "Summary", Skills: "Skills", Experience: "Experience", Projects: "Projects",
		Education: "Education", Certifications: "Certifications", Languages: "Languages", Additional: "Additional information"}
}

// romanianSummary is the rules' opening and closing sentences of a tailored summary in Romanian.
func romanianSummary(headline string, years int, strongest []string, job Job) (opening, joining string) {
	title := headline
	if title == "" {
		title = "Profesionist"
	}
	experience := "experiență practică"
	switch {
	case years == 1:
		experience = "1 an de experiență"
	case years > 1 && years < 20:
		experience = fmt.Sprintf("%d ani de experiență", years)
	case years >= 20:
		experience = fmt.Sprintf("%d de ani de experiență", years)
	}
	opening = title + " cu " + experience
	if len(strongest) > 0 {
		opening += ", mai ales în " + joinRomanian(strongest)
	}
	joining = "Îmi doresc să mă alătur echipei " + job.Company + " ca " + job.Title
	if strings.TrimSpace(job.Product) != "" {
		joining += " și să lucrez la " + job.Product
	}
	return opening + ".", joining + "."
}

// joinRomanian lists items the Romanian way: "A, B și C".
func joinRomanian(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " și " + items[len(items)-1]
}
