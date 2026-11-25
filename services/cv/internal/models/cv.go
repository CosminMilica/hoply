// Package models holds the CV entities, the request and answer payloads of
// docs/api/cv.md, and the views of the profile and the job the service reads.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Statuses of a CV, in the order a CV goes through them.
const (
	StatusQueued   = "queued"
	StatusWriting  = "writing"
	StatusChecking = "checking"
	StatusReady    = "ready"
	StatusFailed   = "failed"
)

// CvRequest is what the person chose for one CV.
type CvRequest struct {
	JobID     string   `json:"job_id"`
	SkipRoles []string `json:"skip_roles"`
	Focus     []string `json:"focus"`
	Confirmed []int    `json:"confirmed"`
}

type RoleSection struct {
	Role    string   `json:"role"`
	Meta    string   `json:"meta"`
	Bullets []string `json:"bullets"`
}

type ProjectSection struct {
	Name    string   `json:"name"`
	Meta    string   `json:"meta"`
	Bullets []string `json:"bullets"`
}

type EducationLine struct {
	Degree string `json:"degree"`
	Meta   string `json:"meta"`
}

// CvDocument is one version of the CV, as printed.
type CvDocument struct {
	Name           string           `json:"name"`
	Headline       string           `json:"headline"`
	Contact        []string         `json:"contact"`
	Links          []string         `json:"links"`
	Summary        string           `json:"summary"`
	Skills         []string         `json:"skills"`
	Experience     []RoleSection    `json:"experience"`
	Projects       []ProjectSection `json:"projects"`
	Education      []EducationLine  `json:"education"`
	Certifications []string         `json:"certifications"`
	Languages      string           `json:"languages"`
	Additional     []string         `json:"additional"`
}

type AtsCheck struct {
	OK    bool   `json:"ok"`
	Label string `json:"label"`
	Hint  string `json:"hint"`
}

type Ats struct {
	Passed int        `json:"passed"`
	Total  int        `json:"total"`
	Checks []AtsCheck `json:"checks"`
}

type Preview struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Written is what the writer produces for a CV.
type Written struct {
	Highlights []string   `json:"highlights"`
	Profile    CvDocument `json:"profile"`
	Tailored   CvDocument `json:"tailored"`
	Ats        Ats        `json:"ats"`
}

// Cv is a CV as the API returns it. The written parts are nil until the CV is
// ready; Preview is set from the checking step on.
type Cv struct {
	ID         uuid.UUID   `json:"id"`
	JobID      uuid.UUID   `json:"job_id"`
	Status     string      `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
	JobTitle   string      `json:"job_title"`
	Company    string      `json:"company"`
	FileName   string      `json:"file_name"`
	Highlights []string    `json:"highlights"`
	Profile    *CvDocument `json:"profile"`
	Tailored   *CvDocument `json:"tailored"`
	Ats        *Ats        `json:"ats"`
	Preview    *Preview    `json:"preview"`
}

type CvList struct {
	Items []Cv `json:"items"`
}

type Gap struct {
	Skill   string `json:"skill"`
	Ask     string `json:"ask"`
	Example string `json:"example"`
}

type Responsibility struct {
	Index    int    `json:"index"`
	Text     string `json:"text"`
	Question string `json:"question"`
	Prefill  string `json:"prefill"`
}

// PrepQuestions is what Prepare your CV asks for one job.
type PrepQuestions struct {
	JobID            uuid.UUID        `json:"job_id"`
	BaseScore        int              `json:"base_score"`
	Covered          []string         `json:"covered"`
	Gaps             []Gap            `json:"gaps"`
	Responsibilities []Responsibility `json:"responsibilities"`
}
