package integration_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"applications/internal/data/repositories"
	"applications/internal/identity"
	"applications/migrations"
	"applications/tests/testutil"
)

// A person is a tenant: their applications and skips are theirs alone, kept apart by row-level security as well as
// by the queries.
func TestRLSKeepsEachTenantsApplications(t *testing.T) {
	ctx := context.Background()
	pool := asAppRole(t, migrated(t))
	repo := repositories.NewPgApplicationRepository(pool).WithScope(bindTenant(pool))
	ana, bob := identity.New(uuid.New(), "ana"), identity.New(uuid.New(), "bob")
	job, skipped := uuid.New(), uuid.New()

	put(t, repo, ana, job, "saved")
	if err := repo.Skip(ctx, ana, skipped); err != nil {
		t.Fatal(err)
	}
	if list, err := repo.List(ctx, bob); err != nil || len(list) != 0 {
		t.Fatalf("bob sees none of ana's applications: %v %v", list, err)
	}
	if skips, err := repo.Skips(ctx, bob); err != nil || len(skips) != 0 {
		t.Fatalf("bob sees none of ana's skips: %v %v", skips, err)
	}
	put(t, repo, bob, job, "applied")
	if removed, err := repo.Delete(ctx, bob, job); err != nil || !removed {
		t.Fatalf("bob removes his own: %v %v", removed, err)
	}
	if err := repo.ClearSkips(ctx, bob); err != nil {
		t.Fatal(err)
	}
	onAnotherDevice := identity.New(ana.TenantID, "ana-phone")
	if list, err := repo.List(ctx, onAnotherDevice); err != nil || len(list) != 1 || list[0].Status != "saved" {
		t.Fatalf("ana's applications belong to her tenant, whoever signs in to it: %v %v", list, err)
	}
	if skips, err := repo.Skips(ctx, ana); err != nil || len(skips) != 1 {
		t.Fatalf("ana keeps her skip: %v %v", skips, err)
	}
	if !refused(t, pool, bob.TenantID, "INSERT INTO applications (customer_id, subject, job_id, status) VALUES ($1, 'bob', $2, 'saved')",
		ana.TenantID, uuid.New()) {
		t.Fatal("the database refuses a row for another tenant")
	}
	var seen int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM applications").Scan(&seen); err != nil || seen != 0 {
		t.Fatalf("without a tenant no row is visible: %d %v", seen, err)
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
