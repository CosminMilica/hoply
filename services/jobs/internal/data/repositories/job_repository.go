// Package repositories holds the persistence contracts and their pgx
// implementations.
//
// A job is either in the catalog (no owner) or added by a person (owned by the
// identity's tenant and subject). Every query sees the catalog plus the
// caller's own jobs; other people's jobs are never visible. Services depend on
// the interface, so unit tests substitute an in-memory implementation.
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

	"jobs/internal/apperrors"
	"jobs/internal/identity"
	"jobs/internal/models"
)

// JobRepository is the persistence contract for jobs. Get returns nil for a
// job the caller cannot see; Delete removes only the caller's own job.
type JobRepository interface {
	List(ctx context.Context, ident identity.IdentityContext, filter models.JobFilter) ([]models.Job, error)
	Get(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (*models.Job, error)
	Create(ctx context.Context, ident identity.IdentityContext, data models.JobInput) (*models.Job, error)
	Delete(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error)
}

const (
	jobColumns = "id, title, company, city, country, work_mode, level, years_required, must_have, nice_to_have, " +
		"responsibilities, about, about_company, product, perks, areas, url, source, read_from, sample, " +
		"customer_id IS NOT NULL, created_at"
	// visible keeps the catalog and the jobs of the $1 tenant and $2 subject.
	visible = "(customer_id IS NULL OR (customer_id = $1 AND subject = $2))"
)

// folded lower-cases a column and removes Romanian diacritics, as
// extract.Fold does in Go.
func folded(column string) string {
	return "lower(translate(" + column + ", 'ĂÂÎȘŞȚŢăâîșşțţ', 'AAISSTTaaissstt'))"
}

// DBTX is the query surface shared by *pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Scope runs fn with the connection a caller's queries use.
// NewPgJobRepository runs fn on the pool; a scope may instead open a
// transaction bound to the caller.
type Scope func(ctx context.Context, ident identity.IdentityContext, fn func(DBTX) error) error

// PgJobRepository is the PostgreSQL implementation.
type PgJobRepository struct {
	scope Scope
}

func NewPgJobRepository(pool *pgxpool.Pool) *PgJobRepository {
	return &PgJobRepository{scope: func(_ context.Context, _ identity.IdentityContext, fn func(DBTX) error) error {
		return fn(pool)
	}}
}

// WithScope returns a copy whose queries run through scope.
func (r *PgJobRepository) WithScope(scope Scope) *PgJobRepository {
	scoped := *r
	scoped.scope = scope
	return &scoped
}

func (r *PgJobRepository) List(ctx context.Context, ident identity.IdentityContext, filter models.JobFilter) ([]models.Job, error) {
	where, args := []string{visible}, []any{ident.TenantID, ident.Subject}
	arg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Q != "" {
		pattern := arg("%" + likeEscape(filter.Q) + "%")
		where = append(where, fmt.Sprintf("(title ILIKE %[1]s OR company ILIKE %[1]s OR "+
			"array_to_string(must_have || nice_to_have, ' ') ILIKE %[1]s)", pattern))
	}
	if len(filter.Cities) > 0 {
		patterns := make([]string, len(filter.Cities))
		for i, city := range filter.Cities {
			patterns[i] = "%" + likeEscape(city) + "%"
		}
		clause := folded("city") + " LIKE ANY(" + arg(patterns) + ")"
		if filter.IncludeRemote {
			clause = "(" + clause + " OR work_mode = 'remote')"
		}
		where = append(where, clause)
	}
	if filter.Country != "" {
		where = append(where, folded("country")+" = "+arg(filter.Country))
	}
	if len(filter.Areas) > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM unnest(areas) AS area WHERE lower(area) = ANY("+arg(filter.Areas)+"))")
	}
	if filter.Level != "" {
		where = append(where, "level = "+arg(filter.Level))
	}
	var jobs []models.Job
	err := r.run(ctx, ident, func(db DBTX) error {
		rows, err := db.Query(ctx, "SELECT "+jobColumns+" FROM jobs WHERE "+strings.Join(where, " AND ")+
			" ORDER BY created_at DESC, id", args...)
		if err != nil {
			return err
		}
		jobs, err = pgx.CollectRows(rows, scanJob)
		return err
	})
	if err != nil {
		return nil, err
	}
	return nonNil(jobs), nil
}

func (r *PgJobRepository) Get(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (*models.Job, error) {
	return r.one(ctx, ident, "SELECT "+jobColumns+" FROM jobs WHERE id = $3 AND "+visible, ident.TenantID, ident.Subject, id)
}

func (r *PgJobRepository) Create(ctx context.Context, ident identity.IdentityContext, d models.JobInput) (*models.Job, error) {
	return r.one(ctx, ident, "INSERT INTO jobs (customer_id, subject, title, company, city, country, work_mode, level, "+
		"years_required, must_have, nice_to_have, responsibilities, about, url, source, read_from) "+
		"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING "+jobColumns,
		ident.TenantID, ident.Subject, d.Title, d.Company, d.City, d.Country, d.WorkMode, d.Level, d.YearsRequired,
		nonNil(d.MustHave), nonNil(d.NiceToHave), nonNil(d.Responsibilities), d.About, d.URL, d.Source, d.ReadFrom)
}

func (r *PgJobRepository) Delete(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	removed := false
	err := r.run(ctx, ident, func(db DBTX) error {
		tag, err := db.Exec(ctx, "DELETE FROM jobs WHERE id = $3 AND customer_id = $1 AND subject = $2",
			ident.TenantID, ident.Subject, id)
		removed = err == nil && tag.RowsAffected() > 0
		return err
	})
	return removed, err
}

func (r *PgJobRepository) one(ctx context.Context, ident identity.IdentityContext, query string, args ...any) (*models.Job, error) {
	var job *models.Job
	err := r.run(ctx, ident, func(db DBTX) error {
		rows, err := db.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		found, err := pgx.CollectExactlyOneRow(rows, scanJob)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		job = &found
		return nil
	})
	if err != nil {
		return nil, err
	}
	return job, nil
}

// run executes fn through the scope and maps database errors to the RFC-007
// contract.
func (r *PgJobRepository) run(ctx context.Context, ident identity.IdentityContext, fn func(DBTX) error) error {
	if err := r.scope(ctx, ident, fn); err != nil {
		return apperrors.FromDatabase(err)
	}
	return nil
}

func scanJob(row pgx.CollectableRow) (models.Job, error) {
	var j models.Job
	err := row.Scan(&j.ID, &j.Title, &j.Company, &j.City, &j.Country, &j.WorkMode, &j.Level, &j.YearsRequired,
		&j.MustHave, &j.NiceToHave, &j.Responsibilities, &j.About, &j.AboutCompany, &j.Product, &j.Perks, &j.Areas,
		&j.URL, &j.Source, &j.ReadFrom, &j.Sample, &j.AddedByMe, &j.CreatedAt)
	for _, list := range []*[]string{&j.MustHave, &j.NiceToHave, &j.Responsibilities, &j.Perks, &j.Areas} {
		*list = nonNil(*list)
	}
	return j, err
}

// likeEscape makes value match literally inside a LIKE pattern.
func likeEscape(value string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(value)
}

// nonNil keeps empty lists serialised as [] and stored as '{}' rather than NULL.
func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
