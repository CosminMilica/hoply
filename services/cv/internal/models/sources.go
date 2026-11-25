package models

import "github.com/google/uuid"

// Bullet is a highlight of a role or a project.
type Bullet struct {
	Text   string   `json:"text"`
	Result string   `json:"result"`
	Skills []string `json:"skills"`
}

type Experience struct {
	ID       string   `json:"id"`
	Role     string   `json:"role"`
	Company  string   `json:"company"`
	Location string   `json:"location"`
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Tools    []string `json:"tools"`
	Bullets  []Bullet `json:"bullets"`
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
	Name           string          `json:"name"`
	Headline       string          `json:"headline"`
	About          string          `json:"about"`
	Email          string          `json:"email"`
	Phone          string          `json:"phone"`
	GitHub         string          `json:"github"`
	LinkedIn       string          `json:"linkedin"`
	City           string          `json:"city"`
	Country        string          `json:"country"`
	AdditionalInfo []string        `json:"additional_info"`
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
	About            string    `json:"about"`
	Product          string    `json:"product"`
	MustHave         []string  `json:"must_have"`
	NiceToHave       []string  `json:"nice_to_have"`
	Responsibilities []string  `json:"responsibilities"`
}
