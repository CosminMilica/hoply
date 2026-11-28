// Package models holds the profile entities and their request payloads.
package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID                      uuid.UUID `json:"id"`
	Name                    string    `json:"name"`
	Headline                string    `json:"headline"`
	About                   string    `json:"about"`
	Email                   string    `json:"email"`
	Phone                   string    `json:"phone"`
	GitHub                  string    `json:"github"`
	LinkedIn                string    `json:"linkedin"`
	City                    string    `json:"city"`
	Country                 string    `json:"country"`
	WorkModes               []string  `json:"work_modes"`
	OpenToRelocation        bool      `json:"open_to_relocation"`
	WorkAuthorization       string    `json:"work_authorization"`
	WorkAuthorizationRegion string    `json:"work_authorization_region"`
	NoticePeriodWeeks       *int      `json:"notice_period_weeks"`
	AvailableFrom           string    `json:"available_from"`
	TargetRoles             []string  `json:"target_roles"`
	TargetSeniority         []string  `json:"target_seniority"`
	EmploymentTypes         []string  `json:"employment_types"`
	SalaryMin               *int      `json:"salary_min"`
	SalaryCurrency          string    `json:"salary_currency"`
	SalaryPeriod            string    `json:"salary_period"`
	// AppLanguage is the language the person uses the app in, "en" or "ro"; empty until they pick one.
	AppLanguage string `json:"app_language"`
	// Domains are the fields of work whose jobs Find jobs shows; none until the person is through onboarding.
	Domains        []string        `json:"domains"`
	AdditionalInfo []string        `json:"additional_info"`
	Notes          []Note          `json:"notes"`
	Skills         []string        `json:"skills"`
	Experience     []Experience    `json:"experience"`
	Projects       []Project       `json:"projects"`
	Education      []Education     `json:"education"`
	Certifications []Certification `json:"certifications"`
	Languages      []Language      `json:"languages"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// ProfilePatch changes only the fields that are present. A nil slice is an
// absent array; an empty one clears it.
type ProfilePatch struct {
	Name                    *string  `json:"name"`
	Headline                *string  `json:"headline"`
	About                   *string  `json:"about"`
	Email                   *string  `json:"email"`
	Phone                   *string  `json:"phone"`
	GitHub                  *string  `json:"github"`
	LinkedIn                *string  `json:"linkedin"`
	City                    *string  `json:"city"`
	Country                 *string  `json:"country"`
	WorkModes               []string `json:"work_modes"`
	OpenToRelocation        *bool    `json:"open_to_relocation"`
	WorkAuthorization       *string  `json:"work_authorization"`
	WorkAuthorizationRegion *string  `json:"work_authorization_region"`
	NoticePeriodWeeks       Nullable `json:"notice_period_weeks"`
	AvailableFrom           *string  `json:"available_from"`
	TargetRoles             []string `json:"target_roles"`
	TargetSeniority         []string `json:"target_seniority"`
	EmploymentTypes         []string `json:"employment_types"`
	SalaryMin               Nullable `json:"salary_min"`
	SalaryCurrency          *string  `json:"salary_currency"`
	SalaryPeriod            *string  `json:"salary_period"`
	AppLanguage             *string  `json:"app_language"`
	Domains                 []string `json:"domains"`
	AdditionalInfo          []string `json:"additional_info"`
	Notes                   []Note   `json:"notes"`
}

// Empty reports whether the patch changes nothing.
func (p ProfilePatch) Empty() bool {
	for _, value := range []*string{p.Name, p.Headline, p.About, p.Email, p.Phone, p.GitHub, p.LinkedIn, p.City,
		p.Country, p.WorkAuthorization, p.WorkAuthorizationRegion, p.AvailableFrom, p.SalaryCurrency, p.SalaryPeriod, p.AppLanguage} {
		if value != nil {
			return false
		}
	}
	for _, list := range [][]string{p.WorkModes, p.TargetRoles, p.TargetSeniority, p.EmploymentTypes, p.AdditionalInfo, p.Domains} {
		if list != nil {
			return false
		}
	}
	return p.Notes == nil && p.OpenToRelocation == nil && !p.NoticePeriodWeeks.Set && !p.SalaryMin.Set
}

// Nullable is a patch field where an absent key, null and a value differ.
// Raw keeps a sent value for the service to validate; Value holds it once
// cleaned.
type Nullable struct {
	Set   bool
	Raw   json.RawMessage
	Value *int
}

func (n *Nullable) UnmarshalJSON(data []byte) error {
	n.Set = true
	n.Value = nil
	n.Raw = nil
	if string(data) != "null" {
		n.Raw = append(json.RawMessage(nil), data...)
	}
	return nil
}

type SkillsInput struct {
	Skills []string `json:"skills"`
}

type Bullet struct {
	Text   string   `json:"text"`
	Result string   `json:"result"`
	Skills []string `json:"skills"`
}

// Note is something the person said about a role, a project or themselves, in
// their own words, with the skills it shows. CVs are written from notes, never
// printed from them.
type Note struct {
	Text   string   `json:"text"`
	Skills []string `json:"skills"`
}

type ExperienceInput struct {
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

type Experience struct {
	ID uuid.UUID `json:"id"`
	ExperienceInput
}

type ProjectInput struct {
	Name    string   `json:"name"`
	Role    string   `json:"role"`
	URL     string   `json:"url"`
	Start   string   `json:"start"`
	End     string   `json:"end"`
	Tools   []string `json:"tools"`
	Bullets []Bullet `json:"bullets"`
	Notes   []Note   `json:"notes"`
}

type Project struct {
	ID uuid.UUID `json:"id"`
	ProjectInput
}

type EducationInput struct {
	Level     string `json:"level"`
	Field     string `json:"field"`
	Degree    string `json:"degree"`
	School    string `json:"school"`
	StartYear string `json:"start_year"`
	EndYear   string `json:"end_year"`
}

type Education struct {
	ID uuid.UUID `json:"id"`
	EducationInput
}

type CertificationInput struct {
	Name         string `json:"name"`
	Issuer       string `json:"issuer"`
	Issued       string `json:"issued"`
	Expires      string `json:"expires"`
	CredentialID string `json:"credential_id"`
	URL          string `json:"url"`
}

type Certification struct {
	ID uuid.UUID `json:"id"`
	CertificationInput
}

type LanguageInput struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

type Language struct {
	ID uuid.UUID `json:"id"`
	LanguageInput
}
