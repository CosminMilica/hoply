// Package models holds the job entities and their request payloads.
package models

import (
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID               uuid.UUID `json:"id"`
	Title            string    `json:"title"`
	Company          string    `json:"company"`
	City             string    `json:"city"`
	Country          string    `json:"country"`
	WorkMode         string    `json:"work_mode"`
	Level            string    `json:"level"`
	YearsRequired    *int      `json:"years_required"`
	MustHave         []string  `json:"must_have"`
	NiceToHave       []string  `json:"nice_to_have"`
	Responsibilities []string  `json:"responsibilities"`
	// Qualifications and Bonuses are the ad's own lines for what it asks and what is a plus.
	Qualifications []string `json:"qualifications"`
	Bonuses        []string `json:"bonuses"`
	About          string   `json:"about"`
	AboutCompany   string   `json:"about_company"`
	Product        string   `json:"product"`
	Perks          []string `json:"perks"`
	Areas          []string `json:"areas"`
	URL            string   `json:"url"`
	Source         string   `json:"source"`
	ReadFrom       string   `json:"read_from"`
	// RequirementsFrom is "rules" or "analysis" (Claude's reading of the ad, written by the cv service).
	RequirementsFrom string    `json:"requirements_from"`
	Sample           bool      `json:"sample"`
	AddedByMe        bool      `json:"added_by_me"`
	CreatedAt        time.Time `json:"created_at"`
	// Origin is the job board platform an imported job was read from; empty for the others.
	Origin   string     `json:"origin"`
	PostedAt *time.Time `json:"posted_at"`
	// ClosedAt is when the job's board stopped listing it; closed jobs leave GET /jobs unless asked by id.
	ClosedAt *time.Time `json:"closed_at"`
}

type JobInput struct {
	Title            string   `json:"title"`
	Company          string   `json:"company"`
	City             string   `json:"city"`
	Country          string   `json:"country"`
	WorkMode         string   `json:"work_mode"`
	Level            string   `json:"level"`
	YearsRequired    *int     `json:"years_required"`
	MustHave         []string `json:"must_have"`
	NiceToHave       []string `json:"nice_to_have"`
	Responsibilities []string `json:"responsibilities"`
	Qualifications   []string `json:"qualifications"`
	Bonuses          []string `json:"bonuses"`
	About            string   `json:"about"`
	AboutCompany     string   `json:"about_company"`
	Perks            []string `json:"perks"`
	URL              string   `json:"url"`
	Source           string   `json:"source"`
	ReadFrom         string   `json:"read_from"`
}

// JobDraft is what POST /jobs/read read, for the person to check and edit.
type JobDraft struct {
	JobInput
	Host string `json:"host"`
}

// ReadRequest is a link ({url}) or pasted text ({text, title?, company?, url?}).
type ReadRequest struct {
	URL     string `json:"url"`
	Text    string `json:"text"`
	Title   string `json:"title"`
	Company string `json:"company"`
}

// JobFilter narrows GET /jobs. City, Country and Areas are folded (lower
// case, no diacritics); Cities holds the city's spellings to look for.
type JobFilter struct {
	Q             string
	Cities        []string
	Country       string
	IncludeRemote bool
	Areas         []string
	Level         string
	// IDs asks for these jobs only, closed ones included.
	IDs   []uuid.UUID
	Limit int
}

type JobList struct {
	Items []Job `json:"items"`
	Total int   `json:"total"`
}

// RequirementsInput replaces a job's requirements with Claude's reading of its ad.
type RequirementsInput struct {
	MustHave   []string `json:"must_have"`
	NiceToHave []string `json:"nice_to_have"`
}

// ImportedJob is a job read from a company's board, as the importer stores it.
type ImportedJob struct {
	JobInput
	Areas      []string
	Origin     string
	ExternalID string
	Board      string
	PostedAt   *time.Time
}

// Board is a company's job board the importer reads. FollowedAt is when a
// person followed it; nil for the boards Hoply lists itself.
type Board struct {
	Company    string     `json:"company"`
	Platform   string     `json:"platform"`
	Board      string     `json:"board"`
	FollowedAt *time.Time `json:"followed_at"`
}

// BoardOf is the board a job link belongs to, and whether its jobs are
// imported already.
type BoardOf struct {
	Platform string `json:"platform"`
	Board    string `json:"board"`
	Company  string `json:"company"`
	Imported bool   `json:"imported"`
}

// FollowInput follows the board of a job link.
type FollowInput struct {
	URL     string `json:"url"`
	Company string `json:"company"`
}
