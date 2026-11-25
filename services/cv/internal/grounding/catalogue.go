// Package grounding keeps invented facts out of a CV that Claude writes
// (docs/plans/2026-11-25-cv-service.md, §4). It numbers every fact of the
// profile, asks Claude to cite the facts behind each sentence, checks each
// sentence against what it cites, and puts the rules engine's sentence in place
// of any that fails.
package grounding

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"cv/internal/cvdoc"
	"cv/internal/models"
)

// Fact is one numbered fact of the profile.
type Fact struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// Owner is the role (E1) or project (P1) the fact belongs to; empty for the
	// facts about the person as a whole.
	Owner string `json:"-"`
	// Tools are the tools of the fact's role or project.
	Tools []string `json:"-"`
}

// Entry is a role or project of the CV, with the ids of its facts.
type Entry struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Tools []string `json:"tools"`
	Facts []string `json:"facts"`
}

// Catalogue is what Claude may write from.
type Catalogue struct {
	Facts    map[string]Fact
	Order    []string
	Roles    []Entry
	Projects []Entry
	// Known are the skills the person shows (profile skills, tools, highlight skills).
	Known []string
	// Missing are the job's requirements the person does not show: never to be claimed.
	Missing []string
	// Matched are the job's requirements the person shows.
	Matched []string
	Company string
}

// Build numbers the facts of the profile for one CV: the roles the request
// keeps, every project, the additional information, certifications and
// education, the headline, the about text and the years of experience.
func Build(profile models.Profile, job cvdoc.Job, request models.CvRequest, now time.Time) Catalogue {
	known := cvdoc.KnownSkills(profile)
	match := cvdoc.MatchJob(job, known)
	c := Catalogue{Facts: map[string]Fact{}, Known: known, Missing: match.Missing, Matched: match.Matched, Company: job.Company}
	add := func(id, text, owner string, tools []string) {
		if text = strings.TrimSpace(text); text == "" {
			return
		}
		c.Facts[id] = Fact{ID: id, Text: text, Owner: owner, Tools: tools}
		c.Order = append(c.Order, id)
	}
	add("S1", profile.Headline, "", nil)
	add("S2", profile.About, "", nil)
	if years := cvdoc.YearsOfExperience(profile, now); years > 0 {
		add("Y", fmt.Sprintf("%d years of experience", years), "", nil)
	}
	n := 0
	for _, role := range profile.Experience {
		if slices.Contains(request.SkipRoles, role.ID) {
			continue
		}
		n++
		id := fmt.Sprintf("E%d", n)
		entry := Entry{ID: id, Title: strings.Join(keep(role.Role, role.Company), " at "), Tools: nonNil(role.Tools), Facts: []string{}}
		for j, bullet := range role.Bullets {
			factID := fmt.Sprintf("%s.B%d", id, j+1)
			add(factID, cvdoc.BulletLine(bullet), id, role.Tools)
			if _, ok := c.Facts[factID]; ok {
				entry.Facts = append(entry.Facts, factID)
			}
		}
		c.Roles = append(c.Roles, entry)
	}
	for k, project := range profile.Projects {
		id := fmt.Sprintf("P%d", k+1)
		entry := Entry{ID: id, Title: project.Name, Tools: nonNil(project.Tools), Facts: []string{}}
		for j, bullet := range project.Bullets {
			factID := fmt.Sprintf("%s.B%d", id, j+1)
			add(factID, cvdoc.BulletLine(bullet), id, project.Tools)
			if _, ok := c.Facts[factID]; ok {
				entry.Facts = append(entry.Facts, factID)
			}
		}
		c.Projects = append(c.Projects, entry)
	}
	for i, line := range profile.AdditionalInfo {
		add(fmt.Sprintf("A%d", i+1), line, "", nil)
	}
	for i, item := range profile.Certifications {
		add(fmt.Sprintf("C%d", i+1), strings.Join(keep(item.Name, item.Issuer), ", "), "", nil)
	}
	for i, item := range profile.Education {
		add(fmt.Sprintf("D%d", i+1), strings.Join(keep(item.Degree, item.Level, item.Field, item.School), ", "), "", nil)
	}
	return c
}

func keep(values ...string) []string {
	kept := []string{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			kept = append(kept, value)
		}
	}
	return kept
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// FactList is the catalogue's facts in order, as the prompt lists them.
func (c Catalogue) FactList() []Fact {
	facts := make([]Fact, 0, len(c.Order))
	for _, id := range c.Order {
		facts = append(facts, c.Facts[id])
	}
	return facts
}
