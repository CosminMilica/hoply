package integration_test

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"jobs/internal/data/repositories"
	"jobs/internal/identity"
	"jobs/internal/models"
	"jobs/migrations"
	"jobs/tests/testutil"
)

// A person is a tenant: the jobs they add are theirs alone, and the catalog is everyone's. Row-level security lets
// the importer, which runs without a tenant, write only the catalog.
func TestRLSKeepsEachTenantsJobsAndSharesTheCatalog(t *testing.T) {
	ctx := context.Background()
	pool := asAppRole(t, migrated(t))
	repo := repositories.NewPgJobRepository(pool).WithScope(bindTenant(pool))
	importer := repositories.NewPgJobRepository(pool)
	ana, bob := identity.New(uuid.New(), "ana"), identity.New(uuid.New(), "bob")

	if _, err := importer.Import(ctx, models.ImportedJob{JobInput: models.JobInput{Title: "Accountant", Company: "Genpact",
		City: "Iași", Country: "Romania", URL: "https://genpact.example/1", Source: "import", ReadFrom: "ats_api"},
		Areas: []string{"General accounting"}, Domains: []string{"accounting"}, Origin: "workday", ExternalID: "1", Board: "genpact"}, time.Now()); err != nil {
		t.Fatalf("the importer writes the catalog: %v", err)
	}
	added, err := repo.Create(ctx, ana, models.JobInput{Title: "Consilier bancar", Company: "BRD", Country: "Romania",
		URL: "https://brd.example/1", Source: "link", ReadFrom: "page_text"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if found, err := repo.Get(ctx, bob, added.ID); err != nil || found != nil {
		t.Fatalf("bob cannot read the job ana added: %+v %v", found, err)
	}
	seen, err := repo.List(ctx, bob, models.JobFilter{Limit: 10})
	if err != nil || len(seen) != 1 || seen[0].Title != "Accountant" {
		t.Fatalf("bob sees the catalog only: %+v %v", seen, err)
	}
	if analysed, err := repo.SetRequirements(ctx, bob, seen[0].ID, models.RequirementsInput{MustHave: []string{"SAP"}}); err != nil ||
		analysed == nil {
		t.Fatalf("an analysis of a catalog ad is shared: %+v %v", analysed, err)
	}
	if removed, err := repo.Delete(ctx, bob, added.ID); err != nil || removed {
		t.Fatalf("bob cannot delete ana's job: %v %v", removed, err)
	}
	if removed, err := repo.Delete(ctx, bob, seen[0].ID); err != nil || removed {
		t.Fatalf("nobody deletes a catalog job: %v %v", removed, err)
	}
	if hers, err := repo.List(ctx, ana, models.JobFilter{Limit: 10}); err != nil || len(hers) != 2 {
		t.Fatalf("ana sees the catalog and her job: %+v %v", hers, err)
	}
	if !refused(t, pool, bob.TenantID, "INSERT INTO jobs (customer_id, subject, title, company, url, source, read_from) "+
		"VALUES ($1, 'bob', 't', 'c', 'u', 'link', 'page_text')", ana.TenantID) {
		t.Fatal("the database refuses a job for another tenant")
	}
	if !refused(t, pool, bob.TenantID, "INSERT INTO jobs (title, company, url, source, read_from) VALUES ('t', 'c', 'u', 'import', 'ats_api')") {
		t.Fatal("a request bound to a tenant cannot add to the catalog")
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
