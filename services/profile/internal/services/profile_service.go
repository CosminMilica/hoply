// Package services holds the business rules. It depends on the repository
// interface, so any implementation (including in-memory test doubles) fits.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"profile/internal/apperrors"
	"profile/internal/data/repositories"
	"profile/internal/identity"
	"profile/internal/models"
)

// Accepted enum values, in the order the contract lists them.
var (
	Levels             = []string{"A1", "A2", "B1", "B2", "C1", "C2", "Native"}
	EducationLevels    = []string{"high_school", "associate", "bachelor", "master", "doctorate", "other"}
	WorkModes          = []string{"remote", "hybrid", "onsite"}
	WorkAuthorizations = []string{"citizen", "permanent_resident", "work_permit", "needs_sponsorship"}
	Seniorities        = []string{"intern", "junior", "mid", "senior", "lead", "principal"}
	EmploymentTypes    = []string{"full_time", "part_time", "contract", "freelance", "internship"}
	SalaryPeriods      = []string{"year", "month", "hour"}
	AppLanguages       = []string{"en", "ro"}
	// Domains are the jobs service's (extract.Domains).
	Domains = []string{"it", "banking", "accounting", "finance", "insurance", "hr"}
)

const (
	minYear = 1950
	maxYear = 2100
)

type ProfileService struct {
	repo repositories.ProfileRepository
}

func NewProfileService(repo repositories.ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

// Get returns the caller's profile, creating it empty on first use.
func (s *ProfileService) Get(ctx context.Context, ident identity.IdentityContext) (*models.Profile, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	return s.repo.Ensure(ctx, ident)
}

func (s *ProfileService) Patch(ctx context.Context, ident identity.IdentityContext, patch models.ProfilePatch) (*models.Profile, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	cleaned, err := CleanPatch(patch)
	if err != nil {
		return nil, err
	}
	if cleaned.Empty() {
		return s.repo.Ensure(ctx, ident)
	}
	return s.repo.Patch(ctx, ident, cleaned)
}

func (s *ProfileService) SetSkills(ctx context.Context, ident identity.IdentityContext, data models.SkillsInput) (*models.Profile, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	skills, err := CleanSkills(data.Skills)
	if err != nil {
		return nil, err
	}
	return s.repo.SetSkills(ctx, ident, skills)
}

func (s *ProfileService) AddExperience(ctx context.Context, ident identity.IdentityContext, data models.ExperienceInput) (*models.Experience, error) {
	return create(ctx, ident, data, CleanExperience, s.repo.AddExperience)
}

func (s *ProfileService) UpdateExperience(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.ExperienceInput) (*models.Experience, error) {
	return update(ctx, ident, "Experience", id, data, CleanExperience, s.repo.UpdateExperience)
}

func (s *ProfileService) DeleteExperience(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) error {
	return remove(ctx, ident, "Experience", id, s.repo.DeleteExperience)
}

func (s *ProfileService) AddProject(ctx context.Context, ident identity.IdentityContext, data models.ProjectInput) (*models.Project, error) {
	return create(ctx, ident, data, CleanProject, s.repo.AddProject)
}

func (s *ProfileService) UpdateProject(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.ProjectInput) (*models.Project, error) {
	return update(ctx, ident, "Project", id, data, CleanProject, s.repo.UpdateProject)
}

func (s *ProfileService) DeleteProject(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) error {
	return remove(ctx, ident, "Project", id, s.repo.DeleteProject)
}

func (s *ProfileService) AddEducation(ctx context.Context, ident identity.IdentityContext, data models.EducationInput) (*models.Education, error) {
	return create(ctx, ident, data, CleanEducation, s.repo.AddEducation)
}

func (s *ProfileService) UpdateEducation(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.EducationInput) (*models.Education, error) {
	return update(ctx, ident, "Education", id, data, CleanEducation, s.repo.UpdateEducation)
}

func (s *ProfileService) DeleteEducation(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) error {
	return remove(ctx, ident, "Education", id, s.repo.DeleteEducation)
}

func (s *ProfileService) AddCertification(ctx context.Context, ident identity.IdentityContext, data models.CertificationInput) (*models.Certification, error) {
	return create(ctx, ident, data, CleanCertification, s.repo.AddCertification)
}

func (s *ProfileService) UpdateCertification(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.CertificationInput) (*models.Certification, error) {
	return update(ctx, ident, "Certification", id, data, CleanCertification, s.repo.UpdateCertification)
}

func (s *ProfileService) DeleteCertification(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) error {
	return remove(ctx, ident, "Certification", id, s.repo.DeleteCertification)
}

func (s *ProfileService) AddLanguage(ctx context.Context, ident identity.IdentityContext, data models.LanguageInput) (*models.Language, error) {
	return create(ctx, ident, data, CleanLanguage, s.repo.AddLanguage)
}

func (s *ProfileService) DeleteLanguage(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) error {
	return remove(ctx, ident, "Language", id, s.repo.DeleteLanguage)
}

func create[In, Out any](ctx context.Context, ident identity.IdentityContext, data In,
	clean func(In) (In, error), add func(context.Context, identity.IdentityContext, In) (*Out, error)) (*Out, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	cleaned, err := clean(data)
	if err != nil {
		return nil, err
	}
	return add(ctx, ident, cleaned)
}

// update cleans data and replaces the caller's entity id; a missing one is 404.
func update[In, Out any](ctx context.Context, ident identity.IdentityContext, entity string, id uuid.UUID, data In,
	clean func(In) (In, error), replace func(context.Context, identity.IdentityContext, uuid.UUID, In) (*Out, error)) (*Out, error) {
	if err := requireSubject(ident); err != nil {
		return nil, err
	}
	cleaned, err := clean(data)
	if err != nil {
		return nil, err
	}
	updated, err := replace(ctx, ident, id, cleaned)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, apperrors.NewNotFound(entity, id.String())
	}
	return updated, nil
}

func remove(ctx context.Context, ident identity.IdentityContext, entity string, id uuid.UUID,
	del func(context.Context, identity.IdentityContext, uuid.UUID) (bool, error)) error {
	if err := requireSubject(ident); err != nil {
		return err
	}
	found, err := del(ctx, ident, id)
	if err != nil {
		return err
	}
	if !found {
		return apperrors.NewNotFound(entity, id.String())
	}
	return nil
}

func requireSubject(ident identity.IdentityContext) error {
	if ident.Subject == "" {
		return apperrors.New(apperrors.AuthRequired, "authentication required")
	}
	return nil
}

// cleaner accumulates the first validation error, so a cleaning function reads
// as a list of rules.
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

func (c *cleaner) optionalText(field string, value *string, maxLength int) {
	if value != nil {
		c.text(field, value, maxLength, false)
	}
}

func (c *cleaner) enum(field string, value *string, allowed []string, required bool) {
	if c.err != nil || value == nil {
		return
	}
	*value = strings.TrimSpace(*value)
	if (*value != "" || required) && !slices.Contains(allowed, *value) {
		prefix := "empty or one of "
		if required {
			prefix = "one of "
		}
		c.err = invalid(field, fmt.Sprintf("'%s' must be %s%s", field, prefix, strings.Join(allowed, ", ")))
	}
}

func (c *cleaner) enumList(field string, values *[]string, allowed []string) {
	if c.err != nil || *values == nil {
		return
	}
	cleaned := make([]string, 0, len(*values))
	for _, value := range *values {
		value = strings.TrimSpace(value)
		if !slices.Contains(allowed, value) {
			c.err = invalid(field, fmt.Sprintf("each of '%s' must be one of %s", field, strings.Join(allowed, ", ")))
			return
		}
		if !slices.Contains(cleaned, value) {
			cleaned = append(cleaned, value)
		}
	}
	*values = cleaned
}

// list trims, drops empty entries and, with dedupe, case-insensitive
// duplicates (keeping the first spelling), then enforces the limits.
func (c *cleaner) list(field string, values *[]string, maxItems, maxLength int, dedupe bool) {
	if c.err != nil {
		return
	}
	cleaned := make([]string, 0, len(*values))
	seen := map[string]bool{}
	for _, value := range *values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if dedupe {
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

func (c *cleaner) optionalList(field string, values *[]string, maxItems, maxLength int, dedupe bool) {
	if *values != nil {
		c.list(field, values, maxItems, maxLength, dedupe)
	}
}

// month accepts "" or YYYY-MM with the year in range.
func (c *cleaner) month(field string, value *string) {
	if c.err != nil || value == nil {
		return
	}
	*value = strings.TrimSpace(*value)
	if *value == "" {
		return
	}
	year, month, ok := strings.Cut(*value, "-")
	m, err := strconv.Atoi(month)
	if !ok || !validYear(year) || len(month) != 2 || !digits(month) || err != nil || m < 1 || m > 12 {
		c.err = invalid(field, fmt.Sprintf("'%s' must be empty or a month YYYY-MM between %d and %d", field, minYear, maxYear))
	}
}

func (c *cleaner) year(field string, value *string) {
	if c.err != nil {
		return
	}
	*value = strings.TrimSpace(*value)
	if *value != "" && !validYear(*value) {
		c.err = invalid(field, fmt.Sprintf("'%s' must be empty or a year between %d and %d", field, minYear, maxYear))
	}
}

// order rejects a non-empty last that sorts before a non-empty first; months
// and years compare correctly as strings.
func (c *cleaner) order(first, last string, field, after string) {
	if c.err == nil && first != "" && last != "" && last < first {
		c.err = invalid(field, fmt.Sprintf("'%s' must not be before '%s'", field, after))
	}
}

func (c *cleaner) count(field string, value *int, maxValue int) {
	if c.err == nil && value != nil && (*value < 0 || *value > maxValue) {
		c.err = invalid(field, fmt.Sprintf("'%s' must be null or an integer from 0 to %d", field, maxValue))
	}
}

func (c *cleaner) nullable(field string, value *models.Nullable, maxValue int) {
	if c.err != nil || !value.Set || value.Raw == nil {
		return
	}
	var parsed int
	if err := json.Unmarshal(value.Raw, &parsed); err != nil {
		c.err = invalid(field, fmt.Sprintf("'%s' must be null or an integer from 0 to %d", field, maxValue))
		return
	}
	value.Value = &parsed
	c.count(field, value.Value, maxValue)
}

func (c *cleaner) bullets(values *[]models.Bullet) {
	if c.err != nil {
		return
	}
	cleaned := make([]models.Bullet, 0, len(*values))
	for _, bullet := range *values {
		bullet.Text = strings.TrimSpace(bullet.Text)
		if bullet.Text == "" {
			continue
		}
		c.text("bullets.text", &bullet.Text, 300, false)
		c.text("bullets.result", &bullet.Result, 200, false)
		c.list("bullets.skills", &bullet.Skills, 10, 60, true)
		cleaned = append(cleaned, bullet)
	}
	if c.err == nil && len(cleaned) > 10 {
		c.err = invalid("bullets", "'bullets' must have at most 10 entries")
	}
	*values = cleaned
}

// Limits of notes.
const (
	MaxNotes    = 30
	MaxNoteText = 1000
)

func (c *cleaner) notes(values *[]models.Note) {
	if c.err != nil || *values == nil {
		return
	}
	cleaned := make([]models.Note, 0, len(*values))
	for _, note := range *values {
		note.Text = strings.TrimSpace(note.Text)
		if note.Text == "" {
			continue
		}
		c.text("notes.text", &note.Text, MaxNoteText, false)
		c.list("notes.skills", &note.Skills, 10, 60, true)
		cleaned = append(cleaned, note)
	}
	if c.err == nil && len(cleaned) > MaxNotes {
		c.err = invalid("notes", fmt.Sprintf("'notes' must have at most %d entries", MaxNotes))
	}
	*values = cleaned
}

// CleanPatch trims and validates every sent field.
func CleanPatch(patch models.ProfilePatch) (models.ProfilePatch, error) {
	var c cleaner
	c.optionalText("name", patch.Name, 120)
	c.optionalText("headline", patch.Headline, 120)
	c.optionalText("about", patch.About, 280)
	c.optionalText("phone", patch.Phone, 200)
	c.optionalText("github", patch.GitHub, 200)
	c.optionalText("linkedin", patch.LinkedIn, 200)
	c.optionalText("city", patch.City, 60)
	c.optionalText("country", patch.Country, 60)
	c.optionalText("work_authorization_region", patch.WorkAuthorizationRegion, 60)
	if c.err == nil && patch.Email != nil {
		*patch.Email = strings.TrimSpace(*patch.Email)
		if *patch.Email != "" && !validEmail(*patch.Email) {
			c.err = invalid("email", "'email' must be empty or a valid address")
		}
	}
	c.enumList("work_modes", &patch.WorkModes, WorkModes)
	c.enum("work_authorization", patch.WorkAuthorization, WorkAuthorizations, false)
	c.nullable("notice_period_weeks", &patch.NoticePeriodWeeks, 52)
	c.month("available_from", patch.AvailableFrom)
	c.optionalList("target_roles", &patch.TargetRoles, 10, 120, true)
	c.enumList("target_seniority", &patch.TargetSeniority, Seniorities)
	c.enumList("employment_types", &patch.EmploymentTypes, EmploymentTypes)
	c.nullable("salary_min", &patch.SalaryMin, 100_000_000)
	if c.err == nil && patch.SalaryCurrency != nil {
		*patch.SalaryCurrency = strings.ToUpper(strings.TrimSpace(*patch.SalaryCurrency))
		if *patch.SalaryCurrency != "" && (len(*patch.SalaryCurrency) != 3 || strings.Trim(*patch.SalaryCurrency, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "") {
			c.err = invalid("salary_currency", "'salary_currency' must be empty or three letters")
		}
	}
	c.enum("salary_period", patch.SalaryPeriod, SalaryPeriods, false)
	c.enum("app_language", patch.AppLanguage, AppLanguages, false)
	c.enumList("domains", &patch.Domains, Domains)
	c.optionalList("additional_info", &patch.AdditionalInfo, 10, 300, false)
	c.notes(&patch.Notes)
	return patch, c.err
}

// CleanSkills trims, drops empty entries and case-insensitive duplicates
// (keeping the first spelling), then enforces at most 50 entries of at most
// 60 characters.
func CleanSkills(skills []string) ([]string, error) {
	var c cleaner
	c.list("skills", &skills, 50, 60, true)
	return skills, c.err
}

func CleanExperience(data models.ExperienceInput) (models.ExperienceInput, error) {
	var c cleaner
	c.text("role", &data.Role, 120, true)
	c.text("company", &data.Company, 120, false)
	c.text("location", &data.Location, 120, false)
	c.enum("employment_type", &data.EmploymentType, EmploymentTypes, false)
	c.text("industry", &data.Industry, 60, false)
	c.count("team_size", data.TeamSize, 100_000)
	c.count("people_managed", data.PeopleManaged, 100_000)
	c.month("start", &data.Start)
	c.month("end", &data.End)
	c.order(data.Start, data.End, "end", "start")
	c.list("tools", &data.Tools, 30, 60, true)
	c.bullets(&data.Bullets)
	c.notes(&data.Notes)
	return data, c.err
}

func CleanProject(data models.ProjectInput) (models.ProjectInput, error) {
	var c cleaner
	c.text("name", &data.Name, 120, true)
	c.text("role", &data.Role, 120, false)
	c.text("url", &data.URL, 200, false)
	c.month("start", &data.Start)
	c.month("end", &data.End)
	c.order(data.Start, data.End, "end", "start")
	c.list("tools", &data.Tools, 30, 60, true)
	c.bullets(&data.Bullets)
	c.notes(&data.Notes)
	return data, c.err
}

func CleanEducation(data models.EducationInput) (models.EducationInput, error) {
	var c cleaner
	c.enum("level", &data.Level, EducationLevels, true)
	c.text("field", &data.Field, 120, false)
	c.text("degree", &data.Degree, 120, false)
	c.text("school", &data.School, 120, false)
	c.year("start_year", &data.StartYear)
	c.year("end_year", &data.EndYear)
	c.order(data.StartYear, data.EndYear, "end_year", "start_year")
	return data, c.err
}

func CleanCertification(data models.CertificationInput) (models.CertificationInput, error) {
	var c cleaner
	c.text("name", &data.Name, 120, true)
	c.text("issuer", &data.Issuer, 120, false)
	c.month("issued", &data.Issued)
	c.month("expires", &data.Expires)
	c.order(data.Issued, data.Expires, "expires", "issued")
	c.text("credential_id", &data.CredentialID, 120, false)
	c.text("url", &data.URL, 200, false)
	return data, c.err
}

func CleanLanguage(data models.LanguageInput) (models.LanguageInput, error) {
	var c cleaner
	c.text("name", &data.Name, 120, true)
	c.enum("level", &data.Level, Levels, true)
	return data, c.err
}

func validYear(value string) bool {
	year, err := strconv.Atoi(value)
	return len(value) == 4 && digits(value) && err == nil && year >= minYear && year <= maxYear
}

func digits(value string) bool {
	return strings.Trim(value, "0123456789") == ""
}

func validEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	if err != nil || address.Name != "" || address.Address != value {
		return false
	}
	domain := value[strings.LastIndex(value, "@")+1:]
	return strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

func invalid(field, message string) error {
	err := apperrors.NewValidation(message)
	err.Context = map[string]any{"field": field}
	return err
}
