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
	About            string    `json:"about"`
	AboutCompany     string    `json:"about_company"`
	Product          string    `json:"product"`
	Perks            []string  `json:"perks"`
	Areas            []string  `json:"areas"`
	URL              string    `json:"url"`
	Source           string    `json:"source"`
	ReadFrom         string    `json:"read_from"`
	Sample           bool      `json:"sample"`
	AddedByMe        bool      `json:"added_by_me"`
	CreatedAt        time.Time `json:"created_at"`
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
	About            string   `json:"about"`
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
}

type JobList struct {
	Items []Job `json:"items"`
	Total int   `json:"total"`
}
