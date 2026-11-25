// Package repositories holds the persistence contracts and their pgx
// implementations.
//
// Every row belongs to the identity's tenant and subject, and every query is
// scoped to the caller. Services depend on the interface, so unit tests
// substitute an in-memory implementation.
package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"applications/internal/apperrors"
	"applications/internal/identity"
	"applications/internal/models"
)

// ApplicationRepository is the persistence contract for applications and
// skips. Put applies the status rules of docs/api/applications.md.
type ApplicationRepository interface {
	List(ctx context.Context, ident identity.IdentityContext) ([]models.Application, error)
	Put(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID, status string) (*models.Application, error)
	Delete(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) (bool, error)
	Skips(ctx context.Context, ident identity.IdentityContext) ([]uuid.UUID, error)
	Skip(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) error
	Unskip(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) error
	ClearSkips(ctx context.Context, ident identity.IdentityContext) error
}

const (
	applicationColumns = "job_id, status, saved_at, cv_ready_at, applied_at, created_at, updated_at"
	// owner matches the $1 tenant and $2 subject.
	owner = "customer_id = $1 AND subject = $2"
)

// putApplication creates the application with the status's timestamp, or
// moves an existing one: cv_ready keeps an applied application applied, a
// status the application is not already in gets a new timestamp, and every
// other timestamp is kept.
const putApplication = `INSERT INTO applications (customer_id, subject, job_id, status, saved_at, cv_ready_at, applied_at)
VALUES ($1, $2, $3, $4::text,
        CASE WHEN $4::text = 'saved' THEN NOW() END,
        CASE WHEN $4::text = 'cv_ready' THEN NOW() END,
        CASE WHEN $4::text = 'applied' THEN NOW() END)
ON CONFLICT (customer_id, subject, job_id) DO UPDATE SET
    status = CASE WHEN EXCLUDED.status = 'cv_ready' AND applications.status = 'applied' THEN 'applied'
                  ELSE EXCLUDED.status END,
    saved_at = CASE WHEN EXCLUDED.status = 'saved' AND applications.status <> 'saved' THEN NOW()
                    ELSE applications.saved_at END,
    cv_ready_at = CASE WHEN EXCLUDED.status = 'cv_ready' AND applications.status <> 'cv_ready' THEN NOW()
                       ELSE applications.cv_ready_at END,
    applied_at = CASE WHEN EXCLUDED.status = 'applied' AND applications.status <> 'applied' THEN NOW()
                      ELSE applications.applied_at END,
    updated_at = NOW()
RETURNING ` + applicationColumns

// DBTX is the query surface shared by *pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Scope runs fn with the connection a caller's queries use.
// NewPgApplicationRepository runs fn on the pool; a scope may instead open a
// transaction bound to the caller.
type Scope func(ctx context.Context, ident identity.IdentityContext, fn func(DBTX) error) error

// PgApplicationRepository is the PostgreSQL implementation.
type PgApplicationRepository struct {
	scope Scope
}

func NewPgApplicationRepository(pool *pgxpool.Pool) *PgApplicationRepository {
	return &PgApplicationRepository{scope: func(_ context.Context, _ identity.IdentityContext, fn func(DBTX) error) error {
		return fn(pool)
	}}
}

// WithScope returns a copy whose queries run through scope.
func (r *PgApplicationRepository) WithScope(scope Scope) *PgApplicationRepository {
	scoped := *r
	scoped.scope = scope
	return &scoped
}

func (r *PgApplicationRepository) List(ctx context.Context, ident identity.IdentityContext) ([]models.Application, error) {
	var items []models.Application
	err := r.run(ctx, ident, func(db DBTX) error {
		rows, err := db.Query(ctx, "SELECT "+applicationColumns+" FROM applications WHERE "+owner+
			" ORDER BY updated_at DESC, job_id", ident.TenantID, ident.Subject)
		if err != nil {
			return err
		}
		items, err = pgx.CollectRows(rows, scanApplication)
		return err
	})
	if err != nil {
		return nil, err
	}
	return nonNil(items), nil
}

func (r *PgApplicationRepository) Put(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID, status string) (*models.Application, error) {
	var application models.Application
	err := r.run(ctx, ident, func(db DBTX) error {
		rows, err := db.Query(ctx, putApplication, ident.TenantID, ident.Subject, jobID, status)
		if err != nil {
			return err
		}
		application, err = pgx.CollectExactlyOneRow(rows, scanApplication)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &application, nil
}

func (r *PgApplicationRepository) Delete(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) (bool, error) {
	return r.exec(ctx, ident, "DELETE FROM applications WHERE "+owner+" AND job_id = $3", jobID)
}

func (r *PgApplicationRepository) Skips(ctx context.Context, ident identity.IdentityContext) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.run(ctx, ident, func(db DBTX) error {
		rows, err := db.Query(ctx, "SELECT job_id FROM skips WHERE "+owner+" ORDER BY skipped_at DESC, job_id",
			ident.TenantID, ident.Subject)
		if err != nil {
			return err
		}
		ids, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		return err
	})
	if err != nil {
		return nil, err
	}
	return nonNil(ids), nil
}

// Skip records the skip; skipping again moves it to the top.
func (r *PgApplicationRepository) Skip(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) error {
	_, err := r.exec(ctx, ident, "INSERT INTO skips (customer_id, subject, job_id) VALUES ($1, $2, $3) "+
		"ON CONFLICT (customer_id, subject, job_id) DO UPDATE SET skipped_at = NOW()", jobID)
	return err
}

func (r *PgApplicationRepository) Unskip(ctx context.Context, ident identity.IdentityContext, jobID uuid.UUID) error {
	_, err := r.exec(ctx, ident, "DELETE FROM skips WHERE "+owner+" AND job_id = $3", jobID)
	return err
}

func (r *PgApplicationRepository) ClearSkips(ctx context.Context, ident identity.IdentityContext) error {
	_, err := r.exec(ctx, ident, "DELETE FROM skips WHERE "+owner)
	return err
}

// exec runs a statement whose first two parameters are the caller's tenant and
// subject, and reports whether it touched a row.
func (r *PgApplicationRepository) exec(ctx context.Context, ident identity.IdentityContext, query string, args ...any) (bool, error) {
	touched := false
	err := r.run(ctx, ident, func(db DBTX) error {
		tag, err := db.Exec(ctx, query, append([]any{ident.TenantID, ident.Subject}, args...)...)
		touched = err == nil && tag.RowsAffected() > 0
		return err
	})
	return touched, err
}

// run executes fn through the scope and maps database errors to the RFC-007
// contract.
func (r *PgApplicationRepository) run(ctx context.Context, ident identity.IdentityContext, fn func(DBTX) error) error {
	if err := r.scope(ctx, ident, fn); err != nil {
		return apperrors.FromDatabase(err)
	}
	return nil
}

func scanApplication(row pgx.CollectableRow) (models.Application, error) {
	var a models.Application
	err := row.Scan(&a.JobID, &a.Status, &a.SavedAt, &a.CVReadyAt, &a.AppliedAt, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

// nonNil keeps empty lists serialised as [] rather than null.
func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
