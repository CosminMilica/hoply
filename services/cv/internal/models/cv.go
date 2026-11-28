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
	// Language is the CV's language: "en" (the default) or "ro".
	Language string `json:"language,omitempty"`
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
	// Language is the language the document is written in, "en" or "ro"; its headings follow it.
	Language string `json:"language"`
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
	Tailoring  *Tailoring `json:"tailoring,omitempty"`
}

// TailoredPriority is a priority of the job and how strongly the CV proves it.
type TailoredPriority struct {
	Theme    string `json:"theme"`
	Strength string `json:"strength"`
}

// Tailoring is how Claude tailored a CV: the angle it took, the job's
// priorities and how strongly the person's facts prove each, and what the review
// changed.
type Tailoring struct {
	Positioning string             `json:"positioning"`
	Priorities  []TailoredPriority `json:"priorities"`
	// Close are the requirements the person knows an equivalent of; the CV shows their category instead.
	Close   []CloseSkill `json:"close"`
	Changes []string     `json:"changes"`
}

// CloseSkill is a requirement the person knows an equivalent of.
type CloseSkill struct {
	Skill    string   `json:"skill"`
	Via      []string `json:"via"`
	Category string   `json:"category"`
}

// Writers of a CV.
const (
	WrittenByClaude = "claude"
	WrittenByRules  = "rules"
)

// Why the rules wrote a CV instead of Claude.
const (
	RulesOff     = "off"           // the service is set to write with the rules
	NoToken      = "no_token"      // no Claude token is installed
	LoginNeeded  = "login_needed"  // the token is expired or revoked
	UsageLimit   = "usage_limit"   // the subscription's usage limit was reached
	ClaudeFailed = "claude_failed" // Claude failed twice or answered something unusable
)

// WriterStatus is how the service writes CVs now (GET /api/v1/writer).
type WriterStatus struct {
	// Writer is "claude" when the service is set to write with Claude, else "rules".
	Writer string `json:"writer"`
	Model  string `json:"model"`
	// State is "ok", "off", "no_token", "login_needed" or "usage_limit".
	State      string     `json:"state"`
	LimitUntil *time.Time `json:"limit_until"`
}

// Authorship is who wrote a CV and how it went.
type Authorship struct {
	WrittenBy   string     `json:"written_by"`
	Model       string     `json:"model"`
	Refused     int        `json:"refused"`
	RulesReason string     `json:"rules_reason"`
	LimitUntil  *time.Time `json:"limit_until"`
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
	// Tailoring is set when Claude wrote the CV.
	Tailoring *Tailoring `json:"tailoring"`
	// WrittenBy is "claude" or "rules" once the CV is ready, nil before.
	WrittenBy   *string    `json:"written_by"`
	Model       string     `json:"model"`
	Refused     int        `json:"refused"`
	RulesReason string     `json:"rules_reason"`
	LimitUntil  *time.Time `json:"limit_until"`
}

type CvList struct {
	Items []Cv `json:"items"`
}

type Gap struct {
	Skill   string `json:"skill"`
	Ask     string `json:"ask"`
	Example string `json:"example"`
}

// ProbeQuestion is a question Claude asks to make a CV stronger for one job:
// about one of the person's facts, a role or project, or the person.
type ProbeQuestion struct {
	Theme    string `json:"theme"`
	Question string `json:"question"`
	Why      string `json:"why"`
	Example  string `json:"example"`
	// About is the fact or the role the question is about, as the person wrote it; empty for the person as a whole.
	About string `json:"about"`
	// Place is where the answer goes: "experience:<id>", "projects:<id>" or "other" (the person's own notes).
	Place string `json:"place"`
	// Choices are answers the person can pick instead of typing: ranges for a count, a size or a frequency.
	Choices []string `json:"choices"`
}

// Probe states of a job's questions.
const (
	ProbeReady       = "ready"
	ProbePending     = "pending"
	ProbeUnavailable = "unavailable"
)

// Probe is the questions for one job: ready, still being asked for, or
// unavailable (Claude cannot answer now; Prepare your CV goes on without them).
type Probe struct {
	Status    string          `json:"status"`
	Questions []ProbeQuestion `json:"questions"`
}

// Close is a requirement the person has not used but knows a close equivalent of.
type Close struct {
	Skill string   `json:"skill"`
	Via   []string `json:"via"`
	Note  string   `json:"note"`
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
	Related          []Close          `json:"related"`
	Responsibilities []Responsibility `json:"responsibilities"`
}

// Letter states of a CV's cover letter and messages; LetterNone is a CV nobody asked one for.
const (
	LetterNone    = "none"
	LetterWriting = "writing"
	LetterReady   = "ready"
	LetterFailed  = "failed"
)

// Message is a message to the recruiter: a subject line and its body.
type Message struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// LetterTexts are the texts written for one CV, each ready to send.
type LetterTexts struct {
	CoverLetter    string  `json:"cover_letter"`
	ConnectionNote string  `json:"connection_note"`
	Message        Message `json:"message"`
	FollowUp       Message `json:"follow_up"`
}

// Letter is a CV's cover letter and messages to the recruiter (docs/api/cv.md).
type Letter struct {
	Status    string `json:"status"`
	Recruiter string `json:"recruiter"`
	// Texts are nil until the letter is ready.
	Texts *LetterTexts `json:"texts"`
	Model string       `json:"model"`
	// Refused is how many sentences failed the checks and were left out.
	Refused int `json:"refused"`
	// Reason is why a letter failed: no_token, login_needed, usage_limit or claude_failed.
	Reason     string     `json:"reason"`
	LimitUntil *time.Time `json:"limit_until"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// LetterRequest asks for a CV's cover letter and messages.
type LetterRequest struct {
	// Recruiter is the name to address, optional.
	Recruiter string `json:"recruiter"`
}

// LetterFile is a cover letter, as the person edited it, to export.
type LetterFile struct {
	Text string `json:"text"`
}
