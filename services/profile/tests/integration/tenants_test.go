package integration_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"profile/internal/data/repositories"
	"profile/internal/identity"
	"profile/internal/models"
	"profile/migrations"
	"profile/tests/testutil"
)

// A person is a tenant: their profile and every list in it are theirs alone, kept apart by row-level security as
// well as by the queries.
func TestRLSKeepsEachTenantsProfile(t *testing.T) {
	ctx := context.Background()
	pool := asAppRole(t, migrated(t))
	repo := repositories.NewPgProfileRepository(pool).WithScope(bindTenant(pool))
	ana, bob := identity.New(uuid.New(), "ana"), identity.New(uuid.New(), "bob")
	name := "Ana"
	if _, err := repo.Patch(ctx, ana, models.ProfilePatch{Name: &name}); err != nil {
		t.Fatal(err)
	}
	role, err := repo.AddExperience(ctx, ana, models.ExperienceInput{Role: "Accountant"})
	if err != nil {
		t.Fatal(err)
	}
	if mine, err := repo.Ensure(ctx, bob); err != nil || mine.Name != "" || len(mine.Experience) != 0 {
		t.Fatalf("bob starts with a profile of his own: %+v %v", mine, err)
	}
	other := models.ExperienceInput{Role: "changed"}
	if changed, err := repo.UpdateExperience(ctx, bob, role.ID, other); err != nil || changed != nil {
		t.Fatalf("bob cannot change ana's role: %+v %v", changed, err)
	}
	if removed, err := repo.DeleteExperience(ctx, bob, role.ID); err != nil || removed {
		t.Fatalf("bob cannot delete ana's role: %v %v", removed, err)
	}
	onAnotherDevice := identity.New(ana.TenantID, "ana-phone")
	if hers, err := repo.Ensure(ctx, onAnotherDevice); err != nil || hers.Name != "Ana" || len(hers.Experience) != 1 ||
		hers.Experience[0].Role != "Accountant" {
		t.Fatalf("ana's profile belongs to her tenant, whoever signs in to it: %+v %v", hers, err)
	}
	if !refused(t, pool, bob.TenantID, "INSERT INTO profiles (customer_id, subject) VALUES ($1, 'bob')", uuid.New()) {
		t.Fatal("the database refuses a profile for another tenant")
	}
	var profiles, roles int
	if err := pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM profiles), (SELECT count(*) FROM profile_experience)").
		Scan(&profiles, &roles); err != nil || profiles != 0 || roles != 0 {
		t.Fatalf("without a tenant no profile or role is visible: %d %d %v", profiles, roles, err)
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
