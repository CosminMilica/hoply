// Package services holds the business rules. It depends on the repository
// and page-reader interfaces, so any implementation (including in-memory test
// doubles) fits.
package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"jobs/internal/apperrors"
	"jobs/internal/data/repositories"
	"jobs/internal/extract"
	"jobs/internal/fetch"
	"jobs/internal/identity"
	"jobs/internal/models"
)

// Accepted enum values, in the order the contract lists them.
var (
	WorkModes = []string{"remote", "hybrid", "onsite"}
	Levels    = []string{"junior", "mid", "senior", "lead"}
	Sources   = []string{"link", "text"}
	ReadFroms = []string{"ats_api", "json_ld", "page_text", "text"}
)

const (
	maxURL      = 2000
	minText     = 80
	maxText     = 50000
	maxYears    = 40
	maxSkill    = 60
	maxLocation = 120
)

// PageReader fetches and reads a job link; *fetch.Reader implements it.
type PageReader interface {
	Read(ctx context.Context, rawURL string) (*fetch.Page, error)
}

type JobService struct {
	repo   repositories.JobRepository
	reader PageReader
}

func NewJobService(repo repositories.JobRepository, reader PageReader) *JobService {
	return &JobService{repo: repo, reader: reader}
}

// JobQuery is GET /jobs's query string as sent.
type JobQuery struct {
	Q             string
	City          string
	Country       string
	IncludeRemote string
	Areas         []string
	Level         string
}

func (s *JobService) List(ctx context.Context, ident identity.IdentityContext, query JobQuery) (*models.JobList, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	filter, err := CleanQuery(query)
	if err != nil {
		return nil, err
	}
	jobs, err := s.repo.List(ctx, ident, filter)
	if err != nil {
		return nil, err
	}
	return &models.JobList{Items: jobs, Total: len(jobs)}, nil
}

func (s *JobService) Get(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (*models.Job, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	job, err := s.repo.Get(ctx, ident, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, apperrors.NewNotFound("Job", id.String())
	}
	return job, nil
}

func (s *JobService) Create(ctx context.Context, ident identity.IdentityContext, data models.JobInput) (*models.Job, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	cleaned, err := CleanJob(data)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, ident, cleaned)
}

// Delete removes the caller's own job; a catalog job or anyone else's is 404.
func (s *JobService) Delete(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) error {
	if err := requireSubject(ident); err != nil {
		return err
	}
	removed, err := s.repo.Delete(ctx, ident, id)
	if err != nil {
		return err
	}
	if !removed {
		return apperrors.NewNotFound("Job", id.String())
	}
	return nil
}

// Read reads pasted text or a link into a draft; it saves nothing.
func (s *JobService) Read(ctx context.Context, ident identity.IdentityContext, req models.ReadRequest) (*models.JobDraft, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	req.URL, req.Text = strings.TrimSpace(req.URL), strings.TrimSpace(req.Text)
	req.Title, req.Company = strings.TrimSpace(req.Title), strings.TrimSpace(req.Company)
	if req.URL != "" && !validURL(req.URL) {
		return nil, invalid("url", fmt.Sprintf("'url' must be an http(s) link of at most %d characters", maxURL))
	}
	if req.Text != "" {
		if n := utf8.RuneCountInString(req.Text); n < minText || n > maxText {
			return nil, invalid("text", fmt.Sprintf("'text' must be %d to %d characters: paste the whole ad", minText, maxText))
		}
		result := extract.Read(req.Text, extract.Hints{Title: req.Title, Company: req.Company})
		return draft(result, "text", "text", req.URL), nil
	}
	if req.URL == "" {
		return nil, invalid("url", "'url' or 'text' is required")
	}
	page, err := s.reader.Read(ctx, req.URL)
	if err != nil {
		var unreadable *fetch.Error
		if errors.As(err, &unreadable) {
			return nil, pageUnreadable(unreadable.Reason)
		}
		return nil, err
	}
	hints := extract.Hints{Title: firstOf(req.Title, page.Title), Company: firstOf(req.Company, page.Company)}
	result := extract.Read(page.Text, hints)
	if page.City != "" {
		result.City = clip(firstOf(extract.CityName(page.City), page.City), maxLocation)
		if _, country := extract.Place(result.City); country != "" {
			result.Country = country
		}
	} else if city, country := extract.Place(page.Location); country != "" {
		result.City, result.Country = city, country
	}
	if country := countryName(page.Country); country != "" {
		result.Country = clip(country, maxLocation)
	}
	result.WorkMode = firstOf(page.WorkMode, extract.WorkMode(page.Location), result.WorkMode)
	return draft(result, "link", page.ReadFrom, req.URL), nil
}

func draft(r extract.Result, source, readFrom, link string) *models.JobDraft {
	host := ""
	if parsed, err := url.Parse(link); err == nil && link != "" {
		host = parsed.Hostname()
	}
	return &models.JobDraft{Host: host, JobInput: models.JobInput{
		Title: r.Title, Company: r.Company, City: r.City, Country: r.Country, WorkMode: r.WorkMode, Level: r.Level,
		YearsRequired: r.YearsRequired, MustHave: r.MustHave, NiceToHave: r.NiceToHave,
		Responsibilities: r.Responsibilities, About: r.About, URL: link, Source: source, ReadFrom: readFrom,
	}}
}

// countryName turns a JobPosting or board country into a name: RO, ROU and
// România become Romania, other two-letter codes are upper-cased, anything
// else is kept.
func countryName(value string) string {
	value = strings.TrimSpace(value)
	switch folded := extract.Fold(value); {
	case folded == "":
		return ""
	case folded == "ro" || folded == "rou" || folded == "romania":
		return "Romania"
	case len(value) == 2:
		return strings.ToUpper(value)
	}
	return value
}

func pageUnreadable(reason fetch.Reason) error {
	err := apperrors.New(apperrors.PageUnreadable, "the page could not be read: "+string(reason))
	err.Context = map[string]any{"reason": string(reason)}
	return err
}

func requireSubject(ident identity.IdentityContext) error {
	if ident.Subject == "" {
		return apperrors.New(apperrors.AuthRequired, "authentication required")
	}
	return nil
}

// CleanQuery validates GET /jobs's query and folds it for matching.
func CleanQuery(q JobQuery) (models.JobFilter, error) {
	filter := models.JobFilter{Q: strings.TrimSpace(q.Q), Country: strings.TrimSpace(extract.Fold(q.Country))}
	if city := strings.TrimSpace(extract.Fold(q.City)); city != "" {
		filter.Cities = []string{city}
		if name := extract.CityName(city); name != "" {
			filter.Cities = append(filter.Cities, extract.Fold(name))
		}
	}
	if raw := strings.TrimSpace(q.IncludeRemote); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return filter, invalid("include_remote", "'include_remote' must be true or false")
		}
		filter.IncludeRemote = value
	}
	for _, area := range q.Areas {
		if area = strings.TrimSpace(extract.Fold(area)); area != "" && !slices.Contains(filter.Areas, area) {
			filter.Areas = append(filter.Areas, area)
		}
	}
	filter.Level = strings.TrimSpace(q.Level)
	if filter.Level != "" && !slices.Contains(Levels, filter.Level) {
		return filter, invalid("level", "'level' must be one of "+strings.Join(Levels, ", "))
	}
	return filter, nil
}

// cleaner keeps the first validation error, so CleanJob reads as a list of
// rules.
type cleaner struct{ err error }

func (c *cleaner) text(field string, value *string, maxLength int, required bool) {
	if c.err != nil {
		return
	}
	*value = strings.TrimSpace(*value)
	switch {
	case required && *value == "":
		c.err = invalid(field, fmt.Sprintf("'%s' is required", field))
	case utf8.RuneCountInString(*value) > maxLength:
		c.err = invalid(field, fmt.Sprintf("'%s' must be at most %d characters", field, maxLength))
	}
}

func (c *cleaner) enum(field string, value *string, allowed []string, allowEmpty bool) {
	if c.err != nil {
		return
	}
	*value = strings.TrimSpace(*value)
	if (*value != "" || !allowEmpty) && !slices.Contains(allowed, *value) {
		prefix := "one of "
		if allowEmpty {
			prefix = "empty or one of "
		}
		c.err = invalid(field, fmt.Sprintf("'%s' must be %s%s", field, prefix, strings.Join(allowed, ", ")))
	}
}

// list trims and drops empty entries; with seen, it also drops entries already
// in seen (case-insensitively) and records the kept ones.
func (c *cleaner) list(field string, values *[]string, maxItems, maxLength int, seen map[string]bool) {
	if c.err != nil {
		return
	}
	cleaned := make([]string, 0, len(*values))
	for _, value := range *values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if seen != nil {
			key := strings.ToLower(value)
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		if utf8.RuneCountInString(value) > maxLength {
			c.err = invalid(field, fmt.Sprintf("each of '%s' must be at most %d characters", field, maxLength))
			return
		}
		cleaned = append(cleaned, value)
	}
	if len(cleaned) > maxItems {
		c.err = invalid(field, fmt.Sprintf("'%s' must have at most %d entries", field, maxItems))
		return
	}
	*values = cleaned
}

// CleanJob trims and validates a job a person adds.
func CleanJob(data models.JobInput) (models.JobInput, error) {
	var c cleaner
	c.text("title", &data.Title, extract.MaxTitle, true)
	c.text("company", &data.Company, maxLocation, false)
	c.text("city", &data.City, maxLocation, false)
	c.text("country", &data.Country, maxLocation, false)
	c.enum("work_mode", &data.WorkMode, WorkModes, true)
	c.enum("level", &data.Level, Levels, true)
	if c.err == nil && data.YearsRequired != nil && (*data.YearsRequired < 0 || *data.YearsRequired > maxYears) {
		c.err = invalid("years_required", fmt.Sprintf("'years_required' must be null or from 0 to %d", maxYears))
	}
	seen := map[string]bool{}
	c.list("must_have", &data.MustHave, extract.MaxSkills, maxSkill, seen)
	c.list("nice_to_have", &data.NiceToHave, extract.MaxSkills, maxSkill, seen)
	c.list("responsibilities", &data.Responsibilities, extract.MaxResponsibilities, extract.MaxResponsibility, nil)
	c.text("about", &data.About, extract.MaxAbout, false)
	if c.err == nil {
		data.URL = strings.TrimSpace(data.URL)
		if data.URL != "" && !validURL(data.URL) {
			c.err = invalid("url", fmt.Sprintf("'url' must be empty or an http(s) link of at most %d characters", maxURL))
		}
	}
	c.enum("source", &data.Source, Sources, false)
	c.enum("read_from", &data.ReadFrom, ReadFroms, false)
	return data, c.err
}

func validURL(value string) bool {
	if utf8.RuneCountInString(value) > maxURL || strings.ContainsAny(value, " \t\n") {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return (scheme == "http" || scheme == "https") && parsed.Hostname() != ""
}

func firstOf(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func clip(value string, n int) string {
	if utf8.RuneCountInString(value) <= n {
		return value
	}
	return strings.TrimSpace(string([]rune(value)[:n]))
}

func invalid(field, message string) error {
	err := apperrors.NewValidation(message)
	err.Context = map[string]any{"field": field}
	return err
}
