package integration_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cv/internal/data/repositories"
	"cv/internal/identity"
	"cv/internal/models"
	"cv/migrations"
	"cv/tests/testutil"
)

// A person is a tenant: their CVs are theirs alone, kept apart by row-level security as well as by the queries. The
// worker writes for every tenant: it claims across them, then changes each CV inside that CV's tenant.
func TestRLSKeepsEachTenantsCVsAndTheWorkerWritesForEach(t *testing.T) {
	ctx := context.Background()
	pool := asAppRole(t, migrated(t))
	repo := repositories.NewPgCvRepository(pool).WithScope(bindTenant(pool))
	queue := repositories.NewPgCvRepository(pool)
	ana, bob := identity.New(uuid.New(), "ana"), identity.New(uuid.New(), "bob")

	cv := newCv(t, repo, ana)
	if found, err := repo.Get(ctx, bob, cv.ID); err != nil || found != nil {
		t.Fatalf("bob cannot read ana's CV: %+v %v", found, err)
	}
	if list, err := repo.List(ctx, bob, nil); err != nil || len(list) != 0 {
		t.Fatalf("bob lists none of ana's CVs: %+v %v", list, err)
	}
	if removed, err := repo.Delete(ctx, bob, cv.ID); err != nil || removed {
		t.Fatalf("bob cannot delete ana's CV: %v %v", removed, err)
	}

	task, err := queue.Claim(ctx)
	if err != nil || task == nil || task.ID != cv.ID || task.Tenant != ana.TenantID {
		t.Fatalf("the worker claims ana's CV and knows it is hers: %+v %v", task, err)
	}
	if err := queue.Checking(ctx, task.ID, &models.Preview{From: "a", To: "b"}); err != nil {
		t.Fatal(err)
	}
	if err := queue.Complete(ctx, task.ID, models.Written{}, models.Authorship{WrittenBy: "rules"}); err != nil {
		t.Fatal(err)
	}
	if done, err := repo.Get(ctx, ana, cv.ID); err != nil || done.Status != "ready" {
		t.Fatalf("the worker's changes land on ana's CV: %+v %v", done, err)
	}
	if err := queue.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	if !refused(t, pool, bob.TenantID, "UPDATE cvs SET customer_id = $1", bob.TenantID) &&
		!refused(t, pool, bob.TenantID, "INSERT INTO probes (customer_id, subject, job_id, fingerprint, status, questions) "+
			"VALUES ($1, 'bob', $2, 'f', 'ready', '[]')", ana.TenantID, uuid.New()) {
		t.Fatal("the database refuses rows for another tenant")
	}
	var seen int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM cvs").Scan(&seen); err != nil || seen != 0 {
		t.Fatalf("without a tenant no CV is visible: %d %v", seen, err)
	}
}

// asAppRole connects to the schema as a new role row-level security applies to (NOSUPERUSER NOBYPASSRLS, owning
// nothing), with the grants deploy/compose/db-app-role.sh gives a service's runtime role.
func asAppRole(t *testing.T, schema testutil.TestSchema) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	role, password := "hoply_rls_"+strings.ReplaceAll(uuid.NewString(), "-", "")[:16], uuid.NewString()
	name := pgx.Identifier{role}.Sanitize()
	for _, statement := range []string{
		"CREATE ROLE " + name + " LOGIN PASSWORD '" + password + "' NOSUPERUSER NOBYPASSRLS",
		"GRANT USAGE ON SCHEMA " + schema.Name + " TO " + name,
		"GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA " + schema.Name + " TO " + name,
		"GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA " + schema.Name + " TO " + name,
	} {
		if _, err := schema.Admin.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	parsed, err := url.Parse(schema.URL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.User = url.UserPassword(role, password)
	pool, err := pgxpool.New(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		for _, statement := range []string{"DROP OWNED BY " + name, "DROP ROLE " + name} {
			if _, err := schema.Admin.Exec(context.Background(), statement); err != nil {
				t.Errorf("%s: %v", statement, err)
			}
		}
	})
	return pool
}

// migrated is a new schema with the service's migrations.
func migrated(t *testing.T) testutil.TestSchema {
	t.Helper()
	schema := testutil.NewSchema(t)
	if err := migrations.Up(context.Background(), schema.URL); err != nil {
		t.Fatal(err)
	}
	return schema
}

// refused reports whether writing a row for another tenant than the one bound fails under row-level security.
func refused(t *testing.T, pool *pgxpool.Pool, bound uuid.UUID, statement string, args ...any) bool {
	t.Helper()
	err := pgx.BeginFunc(context.Background(), pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(context.Background(), "SELECT set_config('app.current_tenant', $1, true)", bound.String()); err != nil {
			return err
		}
		_, err := tx.Exec(context.Background(), statement, args...)
		return err
	})
	return err != nil && strings.Contains(err.Error(), "row-level security")
}
