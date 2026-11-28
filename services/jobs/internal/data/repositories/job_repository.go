// Package repositories holds the persistence contracts and their pgx
// implementations.
//
// A job is either in the catalog (no owner) or added by a person (owned by the
// identity's tenant; subject records who added it). Every query sees the catalog plus the
// caller's own jobs; other people's jobs are never visible. Services depend on
// the interface, so unit tests substitute an in-memory implementation.
package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
	List(ctx context.Context, ident identity.IdentityContext, filter models.JobFilter) ([]models.JobCard, error)
	Count(ctx context.Context, ident identity.IdentityContext, filter models.JobFilter) (int, error)
	Get(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (*models.Job, error)
	Create(ctx context.Context, ident identity.IdentityContext, data models.JobInput) (*models.Job, error)
	Delete(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error)
	SetRequirements(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, data models.RequirementsInput) (*models.Job, error)
}

// ImportStore keeps the jobs the importer reads from company boards.
type ImportStore interface {
	// Import inserts or updates a job its board lists at seen, reopening it if it was closed; it reports whether
	// the job is new. Requirements read by Claude's analysis are kept.
	Import(ctx context.Context, job models.ImportedJob, seen time.Time) (bool, error)
	// Close closes the open jobs of a board that it last listed before seen, and returns how many.
	Close(ctx context.Context, origin, board string, seen time.Time) (int64, error)
	// Fresh returns the ids of a board's open jobs whose description was read at or after since.
	Fresh(ctx context.Context, origin, board string, since time.Time) (map[string]bool, error)
	// Seen marks a board's jobs as listed at seen, without reading them again.
	Seen(ctx context.Context, origin, board string, ids []string, seen time.Time) error
	// CloseUnlisted closes the open jobs of every board but these, given as platform and board pairs, and returns
	// how many: the jobs of a company taken off the list.
	CloseUnlisted(ctx context.Context, origins, boards []string) (int64, error)
}

const (
	jobColumns = "id, title, company, city, country, work_mode, level, years_required, must_have, nice_to_have, " +
		"responsibilities, qualifications, bonuses, about, about_company, product, perks, areas, url, source, read_from, requirements_from, sample, " +
		"customer_id IS NOT NULL, created_at, origin, posted_at, closed_at"
	cardColumns = "id, title, company, city, country, work_mode, level, years_required, must_have, nice_to_have, " +
		"responsibilities, about, areas, url, source, requirements_from, sample, customer_id IS NOT NULL, created_at, origin, posted_at, closed_at"
	// visible keeps the catalog and the jobs of the $1 tenant.
	visible = "(customer_id IS NULL OR customer_id = $1)"
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

// filtered is the WHERE clause and its arguments for a filter, as the caller sees the jobs.
func filtered(ident identity.IdentityContext, filter models.JobFilter) (string, []any) {
	where, args := []string{visible}, []any{ident.TenantID}
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
	if len(filter.IDs) > 0 {
		where = append(where, "id = ANY("+arg(filter.IDs)+")")
	} else {
		where = append(where, "closed_at IS NULL")
	}
	return strings.Join(where, " AND "), args
}

func (r *PgJobRepository) List(ctx context.Context, ident identity.IdentityContext, filter models.JobFilter) ([]models.JobCard, error) {
	where, args := filtered(ident, filter)
	limit := ""
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		limit = fmt.Sprintf(" LIMIT $%d", len(args))
	}
	var jobs []models.JobCard
	err := r.run(ctx, ident, func(db DBTX) error {
		rows, err := db.Query(ctx, "SELECT "+cardColumns+" FROM jobs WHERE "+where+
			" ORDER BY COALESCE(posted_at, created_at) DESC, id"+limit, args...)
		if err != nil {
			return err
		}
		jobs, err = pgx.CollectRows(rows, scanCard)
		return err
	})
	if err != nil {
		return nil, err
	}
	return nonNil(jobs), nil
}

// Count is how many jobs the filter finds, its limit aside.
func (r *PgJobRepository) Count(ctx context.Context, ident identity.IdentityContext, filter models.JobFilter) (int, error) {
	where, args := filtered(ident, filter)
	total := 0
	err := r.run(ctx, ident, func(db DBTX) error {
		return db.QueryRow(ctx, "SELECT count(*) FROM jobs WHERE "+where, args...).Scan(&total)
	})
	return total, err
}

func (r *PgJobRepository) Get(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (*models.Job, error) {
	return r.one(ctx, ident, "SELECT "+jobColumns+" FROM jobs WHERE id = $2 AND "+visible, ident.TenantID, id)
}

func (r *PgJobRepository) Create(ctx context.Context, ident identity.IdentityContext, d models.JobInput) (*models.Job, error) {
	return r.one(ctx, ident, "INSERT INTO jobs (customer_id, subject, title, company, city, country, work_mode, level, "+
		"years_required, must_have, nice_to_have, responsibilities, about, url, source, read_from, qualifications, bonuses, "+
		"about_company, perks) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20) "+
		"RETURNING "+jobColumns,
		ident.TenantID, ident.Subject, d.Title, d.Company, d.City, d.Country, d.WorkMode, d.Level, d.YearsRequired,
		nonNil(d.MustHave), nonNil(d.NiceToHave), nonNil(d.Responsibilities), d.About, d.URL, d.Source, d.ReadFrom,
		nonNil(d.Qualifications), nonNil(d.Bonuses), d.AboutCompany, nonNil(d.Perks))
}

// SetRequirements replaces the requirements of a job the caller sees and marks
// them as read by the analysis; nil when the caller sees no such job. A
// catalog or imported job's ad is the same for everyone, so its analysis is
// shared.
func (r *PgJobRepository) SetRequirements(ctx context.Context, ident identity.IdentityContext, id uuid.UUID, d models.RequirementsInput) (*models.Job, error) {
	return r.one(ctx, ident, "UPDATE jobs SET must_have = $3, nice_to_have = $4, requirements_from = 'analysis' "+
		"WHERE id = $2 AND "+visible+" RETURNING "+jobColumns,
		ident.TenantID, id, nonNil(d.MustHave), nonNil(d.NiceToHave))
}

func (r *PgJobRepository) Delete(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	removed := false
	err := r.run(ctx, ident, func(db DBTX) error {
		tag, err := db.Exec(ctx, "DELETE FROM jobs WHERE id = $2 AND customer_id = $1", ident.TenantID, id)
		removed = err == nil && tag.RowsAffected() > 0
		return err
	})
	return removed, err
}

func (r *PgJobRepository) Import(ctx context.Context, j models.ImportedJob, seen time.Time) (bool, error) {
	created := false
	err := r.run(ctx, identity.IdentityContext{}, func(db DBTX) error {
		return db.QueryRow(ctx, "INSERT INTO jobs (title, company, city, country, work_mode, level, years_required, must_have, "+
			"nice_to_have, responsibilities, about, areas, url, source, read_from, origin, external_id, board, posted_at, last_seen_at, "+
			"qualifications, bonuses, about_company, perks, read_at) "+
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'import', $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $19) "+
			"ON CONFLICT (origin, board, external_id) WHERE source = 'import' DO UPDATE SET title = EXCLUDED.title, "+
			"company = EXCLUDED.company, city = EXCLUDED.city, country = EXCLUDED.country, work_mode = EXCLUDED.work_mode, "+
			"level = EXCLUDED.level, years_required = EXCLUDED.years_required, "+
			"must_have = CASE WHEN jobs.requirements_from = 'analysis' THEN jobs.must_have ELSE EXCLUDED.must_have END, "+
			"nice_to_have = CASE WHEN jobs.requirements_from = 'analysis' THEN jobs.nice_to_have ELSE EXCLUDED.nice_to_have END, "+
			"responsibilities = EXCLUDED.responsibilities, about = EXCLUDED.about, areas = EXCLUDED.areas, url = EXCLUDED.url, "+
			"qualifications = EXCLUDED.qualifications, bonuses = EXCLUDED.bonuses, about_company = EXCLUDED.about_company, perks = EXCLUDED.perks, "+
			"read_from = EXCLUDED.read_from, board = EXCLUDED.board, posted_at = COALESCE(EXCLUDED.posted_at, jobs.posted_at), "+
			"last_seen_at = EXCLUDED.last_seen_at, read_at = EXCLUDED.read_at, closed_at = NULL RETURNING xmax = 0",
			j.Title, j.Company, j.City, j.Country, j.WorkMode, j.Level, j.YearsRequired, nonNil(j.MustHave), nonNil(j.NiceToHave),
			nonNil(j.Responsibilities), j.About, nonNil(j.Areas), j.URL, j.ReadFrom, j.Origin, j.ExternalID, j.Board, j.PostedAt, seen,
			nonNil(j.Qualifications), nonNil(j.Bonuses), j.AboutCompany, nonNil(j.Perks),
		).Scan(&created)
	})
	return created, err
}

func (r *PgJobRepository) Close(ctx context.Context, origin, board string, seen time.Time) (int64, error) {
	var closed int64
	err := r.run(ctx, identity.IdentityContext{}, func(db DBTX) error {
		tag, err := db.Exec(ctx, "UPDATE jobs SET closed_at = now() WHERE source = 'import' AND origin = $1 AND board = $2 "+
			"AND closed_at IS NULL AND last_seen_at < $3", origin, board, seen)
		closed = tag.RowsAffected()
		return err
	})
	return closed, err
}

func (r *PgJobRepository) Fresh(ctx context.Context, origin, board string, since time.Time) (map[string]bool, error) {
	fresh := map[string]bool{}
	err := r.run(ctx, identity.IdentityContext{}, func(db DBTX) error {
		rows, err := db.Query(ctx, "SELECT external_id FROM jobs WHERE source = 'import' AND origin = $1 AND board = $2 "+
			"AND closed_at IS NULL AND read_at >= $3", origin, board, since)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
		for _, id := range ids {
			fresh[id] = true
		}
		return err
	})
	return fresh, err
}

func (r *PgJobRepository) Seen(ctx context.Context, origin, board string, ids []string, seen time.Time) error {
	return r.run(ctx, identity.IdentityContext{}, func(db DBTX) error {
		_, err := db.Exec(ctx, "UPDATE jobs SET last_seen_at = $4 WHERE source = 'import' AND origin = $1 AND board = $2 "+
			"AND external_id = ANY($3)", origin, board, nonNil(ids), seen)
		return err
	})
}

func (r *PgJobRepository) CloseUnlisted(ctx context.Context, origins, boards []string) (int64, error) {
	var closed int64
	err := r.run(ctx, identity.IdentityContext{}, func(db DBTX) error {
		tag, err := db.Exec(ctx, "UPDATE jobs SET closed_at = now() WHERE source = 'import' AND closed_at IS NULL "+
			"AND (origin, board) NOT IN (SELECT * FROM unnest($1::text[], $2::text[]))", nonNil(origins), nonNil(boards))
		closed = tag.RowsAffected()
		return err
	})
	return closed, err
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
		&j.MustHave, &j.NiceToHave, &j.Responsibilities, &j.Qualifications, &j.Bonuses, &j.About, &j.AboutCompany, &j.Product, &j.Perks, &j.Areas,
		&j.URL, &j.Source, &j.ReadFrom, &j.RequirementsFrom, &j.Sample, &j.AddedByMe, &j.CreatedAt, &j.Origin, &j.PostedAt, &j.ClosedAt)
	for _, list := range []*[]string{&j.MustHave, &j.NiceToHave, &j.Responsibilities, &j.Qualifications, &j.Bonuses, &j.Perks, &j.Areas} {
		*list = nonNil(*list)
	}
	return j, err
}

func scanCard(row pgx.CollectableRow) (models.JobCard, error) {
	var j models.Job
	err := row.Scan(&j.ID, &j.Title, &j.Company, &j.City, &j.Country, &j.WorkMode, &j.Level, &j.YearsRequired,
		&j.MustHave, &j.NiceToHave, &j.Responsibilities, &j.About, &j.Areas, &j.URL, &j.Source, &j.RequirementsFrom,
		&j.Sample, &j.AddedByMe, &j.CreatedAt, &j.Origin, &j.PostedAt, &j.ClosedAt)
	for _, list := range []*[]string{&j.MustHave, &j.NiceToHave, &j.Responsibilities, &j.Areas} {
		*list = nonNil(*list)
	}
	return models.CardOf(j), err
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
