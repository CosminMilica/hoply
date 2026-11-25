// Package repositories holds the persistence contracts and their pgx
// implementations.
//
// Every CV belongs to the identity's tenant and subject, and every API query is
// scoped to the caller. The queue methods serve the worker, which writes CVs
// for every caller. Services depend on the interfaces, so unit tests substitute
// in-memory implementations.
package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"cv/internal/apperrors"
	"cv/internal/identity"
	"cv/internal/models"
)

// NewCv is a CV as it is queued.
type NewCv struct {
	JobID           uuid.UUID
	Request         models.CvRequest
	ProfileSnapshot json.RawMessage
	JobSnapshot     json.RawMessage
	JobTitle        string
	Company         string
	FileName        string
}

// CvRepository is the API's persistence contract.
type CvRepository interface {
	Create(ctx context.Context, ident identity.IdentityContext, cv NewCv) (*models.Cv, error)
	Get(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (*models.Cv, error)
	List(ctx context.Context, ident identity.IdentityContext, jobID *uuid.UUID) ([]models.Cv, error)
	Delete(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error)
}

// Task is a queued CV as the worker takes it.
type Task struct {
	ID        uuid.UUID
	Request   models.CvRequest
	Profile   models.Profile
	Job       models.Job
	CreatedAt time.Time
}

// Queue is the worker's persistence contract.
type Queue interface {
	// Claim moves the oldest queued CV to writing and returns it; nil when none is queued.
	Claim(ctx context.Context) (*Task, error)
	Checking(ctx context.Context, id uuid.UUID, preview *models.Preview) error
	Complete(ctx context.Context, id uuid.UUID, written models.Written, by models.Authorship) error
	Fail(ctx context.Context, id uuid.UUID) error
	// Recover puts the CVs a stopped worker left in writing or checking back in
	// the queue, or fails those already tried MaxAttempts times.
	Recover(ctx context.Context) error
}

// Analyses keeps each job's analysis, so it is read once per job.
type Analyses interface {
	Analysis(ctx context.Context, jobID uuid.UUID) (json.RawMessage, error)
	SaveAnalysis(ctx context.Context, jobID uuid.UUID, analysis json.RawMessage, model string) error
}

// StoredProbe is a job's questions as kept.
type StoredProbe struct {
	Fingerprint string
	Status      string
	Questions   []models.ProbeQuestion
	UpdatedAt   time.Time
}

// Probes keeps the questions Prepare your CV asks for each of a person's jobs.
type Probes interface {
	Probe(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) (*StoredProbe, error)
	SaveProbe(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID, probe StoredProbe) error
}

// MaxAttempts is how many times the worker takes a CV before it gives up.
const MaxAttempts = 3

const (
	cvColumns = "id, job_id, status, created_at, updated_at, job_title, company, file_name, preview, written, written_by, model, refused, rules_reason, limit_until"
	// owner matches the $1 tenant and $2 subject.
	owner = "customer_id = $1 AND subject = $2"
)

// DBTX is the query surface shared by *pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Scope runs fn with the connection a caller's queries use. NewPgCvRepository
// runs fn on the pool; a scope may instead open a transaction bound to the caller.
type Scope func(ctx context.Context, ident identity.IdentityContext, fn func(DBTX) error) error

// PgCvRepository is the PostgreSQL implementation of both contracts.
type PgCvRepository struct {
	pool  *pgxpool.Pool
	scope Scope
}

func NewPgCvRepository(pool *pgxpool.Pool) *PgCvRepository {
	return &PgCvRepository{pool: pool, scope: func(_ context.Context, _ identity.IdentityContext, fn func(DBTX) error) error {
		return fn(pool)
	}}
}

// WithScope returns a copy whose API queries run through scope.
func (r *PgCvRepository) WithScope(scope Scope) *PgCvRepository {
	scoped := *r
	scoped.scope = scope
	return &scoped
}

func (r *PgCvRepository) Create(ctx context.Context, ident identity.IdentityContext, cv NewCv) (*models.Cv, error) {
	request, err := json.Marshal(cv.Request)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	var created models.Cv
	err = r.run(ctx, ident, func(db DBTX) error {
		rows, err := db.Query(ctx, `INSERT INTO cvs (customer_id, subject, job_id, request, profile_snapshot, job_snapshot,
job_title, company, file_name) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING `+cvColumns,
			ident.TenantID, ident.Subject, cv.JobID, request, cv.ProfileSnapshot, cv.JobSnapshot, cv.JobTitle, cv.Company, cv.FileName)
		if err != nil {
			return err
		}
		created, err = pgx.CollectExactlyOneRow(rows, scanCv)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *PgCvRepository) Get(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (*models.Cv, error) {
	var found *models.Cv
	err := r.run(ctx, ident, func(db DBTX) error {
		rows, err := db.Query(ctx, "SELECT "+cvColumns+" FROM cvs WHERE "+owner+" AND id = $3", ident.TenantID, ident.Subject, id)
		if err != nil {
			return err
		}
		cv, err := pgx.CollectExactlyOneRow(rows, scanCv)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		found = &cv
		return err
	})
	return found, err
}

func (r *PgCvRepository) List(ctx context.Context, ident identity.IdentityContext, jobID *uuid.UUID) ([]models.Cv, error) {
	var items []models.Cv
	err := r.run(ctx, ident, func(db DBTX) error {
		rows, err := db.Query(ctx, "SELECT "+cvColumns+" FROM cvs WHERE "+owner+
			" AND ($3::uuid IS NULL OR job_id = $3) ORDER BY created_at DESC, id", ident.TenantID, ident.Subject, jobID)
		if err != nil {
			return err
		}
		items, err = pgx.CollectRows(rows, scanCv)
		return err
	})
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []models.Cv{}
	}
	return items, nil
}

func (r *PgCvRepository) Delete(ctx context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	removed := false
	err := r.run(ctx, ident, func(db DBTX) error {
		tag, err := db.Exec(ctx, "DELETE FROM cvs WHERE "+owner+" AND id = $3", ident.TenantID, ident.Subject, id)
		removed = err == nil && tag.RowsAffected() > 0
		return err
	})
	return removed, err
}

func (r *PgCvRepository) Claim(ctx context.Context) (*Task, error) {
	var task *Task
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var (
			claimed                  Task
			request, profile, jobRaw []byte
		)
		err := tx.QueryRow(ctx, `UPDATE cvs SET status = 'writing', attempts = attempts + 1, updated_at = NOW()
WHERE id = (SELECT id FROM cvs WHERE status = 'queued' ORDER BY created_at, id LIMIT 1 FOR UPDATE SKIP LOCKED)
RETURNING id, request, profile_snapshot, job_snapshot, created_at`).Scan(&claimed.ID, &request, &profile, &jobRaw, &claimed.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, part := range []struct {
			raw    []byte
			target any
		}{{request, &claimed.Request}, {profile, &claimed.Profile}, {jobRaw, &claimed.Job}} {
			if err := json.Unmarshal(part.raw, part.target); err != nil {
				return err
			}
		}
		task = &claimed
		return nil
	})
	if err != nil {
		return nil, apperrors.FromDatabase(err)
	}
	return task, nil
}

func (r *PgCvRepository) Checking(ctx context.Context, id uuid.UUID, preview *models.Preview) error {
	return r.update(ctx, "UPDATE cvs SET status = 'checking', preview = $2, updated_at = NOW() WHERE id = $1", id, preview)
}

func (r *PgCvRepository) Complete(ctx context.Context, id uuid.UUID, written models.Written, by models.Authorship) error {
	return r.update(ctx, `UPDATE cvs SET status = 'ready', written = $2, written_by = $3, model = $4, refused = $5, rules_reason = $6,
limit_until = $7, updated_at = NOW() WHERE id = $1`, id, written, by.WrittenBy, by.Model, by.Refused, by.RulesReason, by.LimitUntil)
}

func (r *PgCvRepository) Fail(ctx context.Context, id uuid.UUID) error {
	return r.update(ctx, "UPDATE cvs SET status = 'failed', updated_at = NOW() WHERE id = $1", id)
}

func (r *PgCvRepository) Recover(ctx context.Context) error {
	return r.update(ctx, `UPDATE cvs SET status = CASE WHEN attempts >= $1 THEN 'failed' ELSE 'queued' END, updated_at = NOW()
WHERE status IN ('writing', 'checking')`, MaxAttempts)
}

func (r *PgCvRepository) update(ctx context.Context, query string, args ...any) error {
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return apperrors.FromDatabase(err)
	}
	return nil
}

// run executes fn through the scope and maps database errors to the RFC-007 contract.
func (r *PgCvRepository) run(ctx context.Context, ident identity.IdentityContext, fn func(DBTX) error) error {
	if err := r.scope(ctx, ident, fn); err != nil {
		return apperrors.FromDatabase(err)
	}
	return nil
}

func scanCv(row pgx.CollectableRow) (models.Cv, error) {
	var (
		cv      models.Cv
		written *models.Written
	)
	if err := row.Scan(&cv.ID, &cv.JobID, &cv.Status, &cv.CreatedAt, &cv.UpdatedAt, &cv.JobTitle, &cv.Company,
		&cv.FileName, &cv.Preview, &written, &cv.WrittenBy, &cv.Model, &cv.Refused, &cv.RulesReason, &cv.LimitUntil); err != nil {
		return cv, err
	}
	if written != nil {
		cv.Highlights = written.Highlights
		cv.Profile = &written.Profile
		cv.Tailored = &written.Tailored
		cv.Ats = &written.Ats
		cv.Tailoring = written.Tailoring
	}
	return cv, nil
}

// Analysis is the job's stored analysis; nil when there is none.
func (r *PgCvRepository) Analysis(ctx context.Context, jobID uuid.UUID) (json.RawMessage, error) {
	var analysis json.RawMessage
	err := r.pool.QueryRow(ctx, "SELECT analysis FROM job_analyses WHERE job_id = $1", jobID).Scan(&analysis)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, apperrors.FromDatabase(err)
	}
	return analysis, nil
}

func (r *PgCvRepository) SaveAnalysis(ctx context.Context, jobID uuid.UUID, analysis json.RawMessage, model string) error {
	return r.update(ctx, `INSERT INTO job_analyses (job_id, analysis, model) VALUES ($1, $2, $3)
ON CONFLICT (job_id) DO UPDATE SET analysis = EXCLUDED.analysis, model = EXCLUDED.model, created_at = NOW()`, jobID, analysis, model)
}

func (r *PgCvRepository) Probe(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) (*StoredProbe, error) {
	var found *StoredProbe
	err := r.run(ctx, ident, func(db DBTX) error {
		var probe StoredProbe
		err := db.QueryRow(ctx, "SELECT fingerprint, status, questions, updated_at FROM probes WHERE "+owner+" AND job_id = $3",
			ident.TenantID, ident.Subject, jobID).Scan(&probe.Fingerprint, &probe.Status, &probe.Questions, &probe.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		found = &probe
		return err
	})
	return found, err
}

func (r *PgCvRepository) SaveProbe(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID, probe StoredProbe) error {
	if probe.Questions == nil {
		probe.Questions = []models.ProbeQuestion{}
	}
	return r.run(ctx, ident, func(db DBTX) error {
		_, err := db.Exec(ctx, `INSERT INTO probes (customer_id, subject, job_id, fingerprint, status, questions)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (customer_id, subject, job_id) DO UPDATE SET fingerprint = EXCLUDED.fingerprint, status = EXCLUDED.status,
questions = EXCLUDED.questions, updated_at = NOW()`, ident.TenantID, ident.Subject, jobID, probe.Fingerprint, probe.Status, probe.Questions)
		return err
	})
}
