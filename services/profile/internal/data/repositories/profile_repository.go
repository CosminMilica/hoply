// Package repositories holds the persistence contracts and their pgx
// implementations.
//
// Every method is scoped to the caller: a profile is owned by the identity's
// tenant and subject, and child rows are reached only through that profile.
// Services depend on the interface, so unit tests substitute an in-memory
// implementation.
package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"profile/internal/apperrors"
	"profile/internal/identity"
	"profile/internal/models"
)

// ProfileRepository is the persistence contract for a caller's profile.
// Patch, SetSkills and Add create the profile when it does not exist yet.
// Update and Delete never do, and report a missing child row as nil or false.
type ProfileRepository interface {
	Ensure(ctx context.Context, ident identity.IdentityContext) (*models.Profile, error)
	Patch(ctx context.Context, ident identity.IdentityContext, patch models.ProfilePatch) (*models.Profile, error)
	SetSkills(ctx context.Context, ident identity.IdentityContext, skills []string) (*models.Profile, error)
	AddExperience(ctx context.Context, ident identity.IdentityContext, data models.ExperienceInput) (*models.Experience, error)
	UpdateExperience(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.ExperienceInput) (*models.Experience, error)
	DeleteExperience(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error)
	AddProject(ctx context.Context, ident identity.IdentityContext, data models.ProjectInput) (*models.Project, error)
	UpdateProject(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.ProjectInput) (*models.Project, error)
	DeleteProject(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error)
	AddEducation(ctx context.Context, ident identity.IdentityContext, data models.EducationInput) (*models.Education, error)
	UpdateEducation(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.EducationInput) (*models.Education, error)
	DeleteEducation(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error)
	AddCertification(ctx context.Context, ident identity.IdentityContext, data models.CertificationInput) (*models.Certification, error)
	UpdateCertification(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.CertificationInput) (*models.Certification, error)
	DeleteCertification(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error)
	AddLanguage(ctx context.Context, ident identity.IdentityContext, data models.LanguageInput) (*models.Language, error)
	DeleteLanguage(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error)
}

const (
	profileColumns = "id, name, headline, about, email, phone, github, linkedin, city, country, work_modes, " +
		"open_to_relocation, work_authorization, work_authorization_region, notice_period_weeks, available_from, " +
		"target_roles, target_seniority, employment_types, salary_min, salary_currency, salary_period, " +
		"additional_info, notes, skills, created_at, updated_at"
	// ownedProfile selects the caller's profile id from the $1 tenant and $2 subject.
	ownedProfile = "(SELECT id FROM profiles WHERE customer_id = $1 AND subject = $2)"
)

// child describes a child table: its name, the order its rows are listed in
// and the data columns (without id) that inserts and updates write.
type child struct {
	table   string
	order   string
	columns []string
	updated bool
}

var (
	experienceTable = child{"profile_experience", "position DESC", []string{"role", "company", "location",
		"employment_type", "industry", "team_size", "people_managed", "start_month", "end_month", "tools", "bullets", "notes"}, true}
	projectTable = child{"profile_projects", "position DESC", []string{"name", "role", "url", "start_month",
		"end_month", "tools", "bullets", "notes"}, true}
	educationTable = child{"profile_education", "position", []string{"level", "field", "degree", "school",
		"start_year", "end_year"}, false}
	certificationTable = child{"profile_certifications", "position", []string{"name", "issuer", "issued", "expires",
		"credential_id", "url"}, false}
	languageTable = child{"profile_languages", "position", []string{"name", "level"}, false}
)

func (c child) returning() string {
	return "id, " + strings.Join(c.columns, ", ")
}

// keptWhenAbsent are the columns a request may leave out (sent as NULL): an
// insert stores their default, an update keeps their value.
var keptWhenAbsent = map[string]string{"notes": "'[]'::jsonb"}

func (c child) insert() string {
	placeholders := make([]string, len(c.columns))
	for i, column := range c.columns {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		if fallback, ok := keptWhenAbsent[column]; ok {
			placeholders[i] = fmt.Sprintf("COALESCE($%d::jsonb, %s)", i+2, fallback)
		}
	}
	return "INSERT INTO " + c.table + " (profile_id, " + strings.Join(c.columns, ", ") + ") VALUES ($1, " +
		strings.Join(placeholders, ", ") + ") RETURNING " + c.returning()
}

func (c child) update() string {
	sets := make([]string, len(c.columns))
	for i, column := range c.columns {
		sets[i] = fmt.Sprintf("%s = $%d", column, i+4)
		if _, ok := keptWhenAbsent[column]; ok {
			sets[i] = fmt.Sprintf("%s = COALESCE($%d::jsonb, %s)", column, i+4, column)
		}
	}
	if c.updated {
		sets = append(sets, "updated_at = NOW()")
	}
	return "UPDATE " + c.table + " SET " + strings.Join(sets, ", ") +
		" WHERE id = $3 AND profile_id = " + ownedProfile + " RETURNING " + c.returning()
}

func (c child) delete() string {
	return "DELETE FROM " + c.table + " WHERE id = $3 AND profile_id = " + ownedProfile
}

func (c child) list() string {
	return "SELECT " + c.returning() + " FROM " + c.table + " WHERE profile_id = $1 ORDER BY " + c.order
}

// DBTX is the query surface shared by *pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Scope runs fn with the connection a caller's queries use.
// NewPgProfileRepository runs fn on the pool; a scope may instead open a
// transaction bound to the caller.
type Scope func(ctx context.Context, ident identity.IdentityContext, fn func(DBTX) error) error

// PgProfileRepository is the PostgreSQL implementation.
type PgProfileRepository struct {
	scope Scope
}

func NewPgProfileRepository(pool *pgxpool.Pool) *PgProfileRepository {
	return &PgProfileRepository{scope: func(_ context.Context, _ identity.IdentityContext, fn func(DBTX) error) error {
		return fn(pool)
	}}
}

// WithScope returns a copy whose queries run through scope.
func (r *PgProfileRepository) WithScope(scope Scope) *PgProfileRepository {
	scoped := *r
	scoped.scope = scope
	return &scoped
}

func (r *PgProfileRepository) Ensure(ctx context.Context, ident identity.IdentityContext) (*models.Profile, error) {
	return r.profile(ctx, ident, func(DBTX, uuid.UUID) error { return nil })
}

func (r *PgProfileRepository) Patch(ctx context.Context, ident identity.IdentityContext, patch models.ProfilePatch) (*models.Profile, error) {
	args := []any{nil}
	sets := []string{"updated_at = NOW()"}
	set := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	for _, field := range []struct {
		column string
		value  *string
	}{
		{"name", patch.Name}, {"headline", patch.Headline}, {"about", patch.About}, {"email", patch.Email},
		{"phone", patch.Phone}, {"github", patch.GitHub}, {"linkedin", patch.LinkedIn}, {"city", patch.City},
		{"country", patch.Country}, {"work_authorization", patch.WorkAuthorization},
		{"work_authorization_region", patch.WorkAuthorizationRegion}, {"available_from", patch.AvailableFrom},
		{"salary_currency", patch.SalaryCurrency}, {"salary_period", patch.SalaryPeriod},
	} {
		if field.value != nil {
			set(field.column, *field.value)
		}
	}
	for _, field := range []struct {
		column string
		value  []string
	}{
		{"work_modes", patch.WorkModes}, {"target_roles", patch.TargetRoles},
		{"target_seniority", patch.TargetSeniority}, {"employment_types", patch.EmploymentTypes},
		{"additional_info", patch.AdditionalInfo},
	} {
		if field.value != nil {
			set(field.column, field.value)
		}
	}
	if patch.Notes != nil {
		set("notes", notes(patch.Notes))
	}
	if patch.OpenToRelocation != nil {
		set("open_to_relocation", *patch.OpenToRelocation)
	}
	if patch.NoticePeriodWeeks.Set {
		set("notice_period_weeks", patch.NoticePeriodWeeks.Value)
	}
	if patch.SalaryMin.Set {
		set("salary_min", patch.SalaryMin.Value)
	}
	return r.profile(ctx, ident, func(db DBTX, id uuid.UUID) error {
		args[0] = id
		_, err := db.Exec(ctx, "UPDATE profiles SET "+strings.Join(sets, ", ")+" WHERE id = $1", args...)
		return err
	})
}

func (r *PgProfileRepository) SetSkills(ctx context.Context, ident identity.IdentityContext, skills []string) (*models.Profile, error) {
	return r.profile(ctx, ident, func(db DBTX, id uuid.UUID) error {
		_, err := db.Exec(ctx, "UPDATE profiles SET skills = $2, updated_at = NOW() WHERE id = $1", id, nonNil(skills))
		return err
	})
}

func experienceValues(d models.ExperienceInput) []any {
	return []any{d.Role, d.Company, d.Location, d.EmploymentType, d.Industry, d.TeamSize, d.PeopleManaged,
		d.Start, d.End, nonNil(d.Tools), bullets(d.Bullets), sentNotes(d.Notes)}
}

func (r *PgProfileRepository) AddExperience(ctx context.Context, ident identity.IdentityContext, data models.ExperienceInput) (*models.Experience, error) {
	return add(ctx, r, ident, experienceTable, scanExperience, experienceValues(data))
}

func (r *PgProfileRepository) UpdateExperience(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.ExperienceInput) (*models.Experience, error) {
	return replace(ctx, r, ident, id, experienceTable, scanExperience, experienceValues(data))
}

func (r *PgProfileRepository) DeleteExperience(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	return r.remove(ctx, ident, experienceTable, id)
}

func projectValues(d models.ProjectInput) []any {
	return []any{d.Name, d.Role, d.URL, d.Start, d.End, nonNil(d.Tools), bullets(d.Bullets), sentNotes(d.Notes)}
}

func (r *PgProfileRepository) AddProject(ctx context.Context, ident identity.IdentityContext, data models.ProjectInput) (*models.Project, error) {
	return add(ctx, r, ident, projectTable, scanProject, projectValues(data))
}

func (r *PgProfileRepository) UpdateProject(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.ProjectInput) (*models.Project, error) {
	return replace(ctx, r, ident, id, projectTable, scanProject, projectValues(data))
}

func (r *PgProfileRepository) DeleteProject(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	return r.remove(ctx, ident, projectTable, id)
}

func educationValues(d models.EducationInput) []any {
	return []any{d.Level, d.Field, d.Degree, d.School, d.StartYear, d.EndYear}
}

func (r *PgProfileRepository) AddEducation(ctx context.Context, ident identity.IdentityContext, data models.EducationInput) (*models.Education, error) {
	return add(ctx, r, ident, educationTable, scanEducation, educationValues(data))
}

func (r *PgProfileRepository) UpdateEducation(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.EducationInput) (*models.Education, error) {
	return replace(ctx, r, ident, id, educationTable, scanEducation, educationValues(data))
}

func (r *PgProfileRepository) DeleteEducation(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	return r.remove(ctx, ident, educationTable, id)
}

func certificationValues(d models.CertificationInput) []any {
	return []any{d.Name, d.Issuer, d.Issued, d.Expires, d.CredentialID, d.URL}
}

func (r *PgProfileRepository) AddCertification(ctx context.Context, ident identity.IdentityContext, data models.CertificationInput) (*models.Certification, error) {
	return add(ctx, r, ident, certificationTable, scanCertification, certificationValues(data))
}

func (r *PgProfileRepository) UpdateCertification(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.CertificationInput) (*models.Certification, error) {
	return replace(ctx, r, ident, id, certificationTable, scanCertification, certificationValues(data))
}

func (r *PgProfileRepository) DeleteCertification(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	return r.remove(ctx, ident, certificationTable, id)
}

func (r *PgProfileRepository) AddLanguage(ctx context.Context, ident identity.IdentityContext, data models.LanguageInput) (*models.Language, error) {
	return add(ctx, r, ident, languageTable, scanLanguage, []any{data.Name, data.Level})
}

func (r *PgProfileRepository) DeleteLanguage(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	return r.remove(ctx, ident, languageTable, id)
}

// profile ensures the caller's profile, applies change to it and returns the
// whole profile.
func (r *PgProfileRepository) profile(ctx context.Context, ident identity.IdentityContext, change func(DBTX, uuid.UUID) error) (*models.Profile, error) {
	var profile *models.Profile
	err := r.run(ctx, ident, func(db DBTX) error {
		id, err := ensureID(ctx, db, ident)
		if err != nil {
			return err
		}
		if err := change(db, id); err != nil {
			return err
		}
		profile, err = load(ctx, db, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// add inserts a child row of the caller's profile, creating the profile first.
func add[T any](ctx context.Context, r *PgProfileRepository, ident identity.IdentityContext, table child, scan pgx.RowToFunc[T], values []any) (*T, error) {
	var created T
	err := r.run(ctx, ident, func(db DBTX) error {
		id, err := ensureID(ctx, db, ident)
		if err != nil {
			return err
		}
		rows, err := db.Query(ctx, table.insert(), append([]any{id}, values...)...)
		if err != nil {
			return err
		}
		if created, err = pgx.CollectExactlyOneRow(rows, scan); err != nil {
			return err
		}
		return touch(ctx, db, ident)
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// replace updates a child row of the caller's profile in place, so it keeps its
// position. A row the caller does not own yields nil.
func replace[T any](ctx context.Context, r *PgProfileRepository, ident identity.IdentityContext, id uuid.UUID, table child, scan pgx.RowToFunc[T], values []any) (*T, error) {
	var updated *T
	err := r.run(ctx, ident, func(db DBTX) error {
		rows, err := db.Query(ctx, table.update(), append([]any{ident.TenantID, ident.Subject, id}, values...)...)
		if err != nil {
			return err
		}
		found, err := pgx.CollectExactlyOneRow(rows, scan)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		updated = &found
		return touch(ctx, db, ident)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *PgProfileRepository) remove(ctx context.Context, ident identity.IdentityContext, table child, id uuid.UUID) (bool, error) {
	removed := false
	err := r.run(ctx, ident, func(db DBTX) error {
		tag, err := db.Exec(ctx, table.delete(), ident.TenantID, ident.Subject, id)
		if err != nil {
			return err
		}
		if removed = tag.RowsAffected() > 0; removed {
			return touch(ctx, db, ident)
		}
		return nil
	})
	return removed, err
}

// run executes fn through the scope and maps database errors to the RFC-007
// contract.
func (r *PgProfileRepository) run(ctx context.Context, ident identity.IdentityContext, fn func(DBTX) error) error {
	if err := r.scope(ctx, ident, fn); err != nil {
		return apperrors.FromDatabase(err)
	}
	return nil
}

// ensureID creates the caller's profile unless it exists and returns its id.
// The select runs as its own statement so it sees a row a concurrent request
// committed while the insert waited on the conflict.
func ensureID(ctx context.Context, db DBTX, ident identity.IdentityContext) (uuid.UUID, error) {
	if _, err := db.Exec(ctx, "INSERT INTO profiles (customer_id, subject) VALUES ($1, $2) "+
		"ON CONFLICT (customer_id, subject) DO NOTHING", ident.TenantID, ident.Subject); err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err := db.QueryRow(ctx, "SELECT id FROM profiles WHERE customer_id = $1 AND subject = $2",
		ident.TenantID, ident.Subject).Scan(&id)
	return id, err
}

func touch(ctx context.Context, db DBTX, ident identity.IdentityContext) error {
	_, err := db.Exec(ctx, "UPDATE profiles SET updated_at = NOW() WHERE customer_id = $1 AND subject = $2",
		ident.TenantID, ident.Subject)
	return err
}

func load(ctx context.Context, db DBTX, id uuid.UUID) (*models.Profile, error) {
	rows, err := db.Query(ctx, "SELECT "+profileColumns+" FROM profiles WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	profile, err := pgx.CollectExactlyOneRow(rows, scanProfile)
	if err != nil {
		return nil, err
	}
	if profile.Experience, err = children(ctx, db, experienceTable, scanExperience, id); err != nil {
		return nil, err
	}
	if profile.Projects, err = children(ctx, db, projectTable, scanProject, id); err != nil {
		return nil, err
	}
	if profile.Education, err = children(ctx, db, educationTable, scanEducation, id); err != nil {
		return nil, err
	}
	if profile.Certifications, err = children(ctx, db, certificationTable, scanCertification, id); err != nil {
		return nil, err
	}
	if profile.Languages, err = children(ctx, db, languageTable, scanLanguage, id); err != nil {
		return nil, err
	}
	return &profile, nil
}

func children[T any](ctx context.Context, db DBTX, table child, scan pgx.RowToFunc[T], id uuid.UUID) ([]T, error) {
	rows, err := db.Query(ctx, table.list(), id)
	if err != nil {
		return nil, err
	}
	found, err := pgx.CollectRows(rows, scan)
	if err != nil {
		return nil, err
	}
	return nonNil(found), nil
}

func scanProfile(row pgx.CollectableRow) (models.Profile, error) {
	var p models.Profile
	err := row.Scan(&p.ID, &p.Name, &p.Headline, &p.About, &p.Email, &p.Phone, &p.GitHub, &p.LinkedIn, &p.City,
		&p.Country, &p.WorkModes, &p.OpenToRelocation, &p.WorkAuthorization, &p.WorkAuthorizationRegion,
		&p.NoticePeriodWeeks, &p.AvailableFrom, &p.TargetRoles, &p.TargetSeniority, &p.EmploymentTypes, &p.SalaryMin,
		&p.SalaryCurrency, &p.SalaryPeriod, &p.AdditionalInfo, &p.Notes, &p.Skills, &p.CreatedAt, &p.UpdatedAt)
	p.Notes = notes(p.Notes)
	for _, list := range []*[]string{&p.WorkModes, &p.TargetRoles, &p.TargetSeniority, &p.EmploymentTypes,
		&p.AdditionalInfo, &p.Skills} {
		*list = nonNil(*list)
	}
	return p, err
}

func scanExperience(row pgx.CollectableRow) (models.Experience, error) {
	var e models.Experience
	err := row.Scan(&e.ID, &e.Role, &e.Company, &e.Location, &e.EmploymentType, &e.Industry, &e.TeamSize,
		&e.PeopleManaged, &e.Start, &e.End, &e.Tools, &e.Bullets, &e.Notes)
	e.Tools, e.Bullets, e.Notes = nonNil(e.Tools), bullets(e.Bullets), notes(e.Notes)
	return e, err
}

func scanProject(row pgx.CollectableRow) (models.Project, error) {
	var p models.Project
	err := row.Scan(&p.ID, &p.Name, &p.Role, &p.URL, &p.Start, &p.End, &p.Tools, &p.Bullets, &p.Notes)
	p.Tools, p.Bullets, p.Notes = nonNil(p.Tools), bullets(p.Bullets), notes(p.Notes)
	return p, err
}

func scanEducation(row pgx.CollectableRow) (models.Education, error) {
	var e models.Education
	err := row.Scan(&e.ID, &e.Level, &e.Field, &e.Degree, &e.School, &e.StartYear, &e.EndYear)
	return e, err
}

func scanCertification(row pgx.CollectableRow) (models.Certification, error) {
	var c models.Certification
	err := row.Scan(&c.ID, &c.Name, &c.Issuer, &c.Issued, &c.Expires, &c.CredentialID, &c.URL)
	return c, err
}

func scanLanguage(row pgx.CollectableRow) (models.Language, error) {
	var l models.Language
	err := row.Scan(&l.ID, &l.Name, &l.Level)
	return l, err
}

// bullets keeps every bullet's skills a list, so they serialise as [].
func bullets(items []models.Bullet) []models.Bullet {
	items = nonNil(items)
	for i := range items {
		items[i].Skills = nonNil(items[i].Skills)
	}
	return items
}

// sentNotes is NULL for notes the request left out, so they are kept.
func sentNotes(items []models.Note) any {
	if items == nil {
		return nil
	}
	return notes(items)
}

// notes keeps every note's skills a list, so they serialise as [].
func notes(items []models.Note) []models.Note {
	items = nonNil(items)
	for i := range items {
		items[i].Skills = nonNil(items[i].Skills)
	}
	return items
}

// nonNil keeps empty lists serialised as [] and stored as '{}' rather than NULL.
func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
