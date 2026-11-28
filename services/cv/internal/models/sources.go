package models

import "github.com/google/uuid"

// Note is something the person said about a role, a project or themselves, in
// their own words: material for Claude, never printed as it is.
type Note struct {
	Text   string   `json:"text"`
	Skills []string `json:"skills"`
}

// Bullet is a highlight of a role or a project.
type Bullet struct {
	Text   string   `json:"text"`
	Result string   `json:"result"`
	Skills []string `json:"skills"`
}

type Experience struct {
	ID             string   `json:"id"`
	Role           string   `json:"role"`
	Company        string   `json:"company"`
	Location       string   `json:"location"`
	EmploymentType string   `json:"employment_type"`
	Industry       string   `json:"industry"`
	TeamSize       *int     `json:"team_size"`
	PeopleManaged  *int     `json:"people_managed"`
	Start          string   `json:"start"`
	End            string   `json:"end"`
	Tools          []string `json:"tools"`
	Bullets        []Bullet `json:"bullets"`
	Notes          []Note   `json:"notes"`
}

type Project struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Role    string   `json:"role"`
	URL     string   `json:"url"`
	Start   string   `json:"start"`
	End     string   `json:"end"`
	Tools   []string `json:"tools"`
	Bullets []Bullet `json:"bullets"`
	Notes   []Note   `json:"notes"`
}

type Education struct {
	Level     string `json:"level"`
	Field     string `json:"field"`
	Degree    string `json:"degree"`
	School    string `json:"school"`
	StartYear string `json:"start_year"`
	EndYear   string `json:"end_year"`
}

type Certification struct {
	Name         string `json:"name"`
	Issuer       string `json:"issuer"`
	Issued       string `json:"issued"`
	Expires      string `json:"expires"`
	CredentialID string `json:"credential_id"`
	URL          string `json:"url"`
}

type Language struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

// Profile is the part of the profile service's profile (docs/api/profile.md)
// a CV is made from.
type Profile struct {
	Name     string `json:"name"`
	Headline string `json:"headline"`
	About    string `json:"about"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	GitHub   string `json:"github"`
	LinkedIn string `json:"linkedin"`
	City     string `json:"city"`
	Country  string `json:"country"`
	// WorkAuthorization counts toward how complete the profile is (cvdoc.Readiness).
	WorkAuthorization string `json:"work_authorization"`
	// AppLanguage is the language the person uses the app in, "en" or "ro" (empty: English); what Claude asks them is
	// written in it.
	AppLanguage    string          `json:"app_language"`
	AdditionalInfo []string        `json:"additional_info"`
	Notes          []Note          `json:"notes"`
	Skills         []string        `json:"skills"`
	Experience     []Experience    `json:"experience"`
	Projects       []Project       `json:"projects"`
	Education      []Education     `json:"education"`
	Certifications []Certification `json:"certifications"`
	Languages      []Language      `json:"languages"`
}

// Job is the part of the jobs service's job (docs/api/jobs.md) a CV is written
// around.
type Job struct {
	ID               uuid.UUID `json:"id"`
	Title            string    `json:"title"`
	Company          string    `json:"company"`
	Level            string    `json:"level"`
	YearsRequired    *int      `json:"years_required"`
	About            string    `json:"about"`
	AboutCompany     string    `json:"about_company"`
	Product          string    `json:"product"`
	MustHave         []string  `json:"must_have"`
	NiceToHave       []string  `json:"nice_to_have"`
	Responsibilities []string  `json:"responsibilities"`
	// Qualifications and Bonuses are the ad's own lines for what it asks and what is a plus.
	Qualifications []string `json:"qualifications"`
	Bonuses        []string `json:"bonuses"`
}
