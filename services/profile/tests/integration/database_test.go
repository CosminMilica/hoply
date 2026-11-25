package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"profile/internal/app"
	"profile/internal/config"
	"profile/internal/data/repositories"
	"profile/internal/db"
	"profile/internal/identity"
	"profile/migrations"
	"profile/tests/testutil"
)

// isolatedSchema migrates a fresh schema and returns a url whose connections
// use it, so tests never touch the developer's tables.
func isolatedSchema(t *testing.T) string {
	t.Helper()
	scoped := testutil.NewSchema(t).URL
	ctx := context.Background()
	if err := migrations.Up(ctx, scoped); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Up(ctx, scoped); err != nil {
		t.Fatalf("migrations must be idempotent: %v", err)
	}
	return scoped
}

func application(t *testing.T, databaseURL string) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	pool, err := db.NewPool(context.Background(), databaseURL, 4, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cfg := config.Defaults()
	return app.New(&cfg, pool), pool
}

// bindTenant runs each call in a transaction bound to the caller's tenant, as
// database.multitenancy=shared_rls does, so the test holds under forced RLS.
func bindTenant(pool *pgxpool.Pool) repositories.Scope {
	return func(ctx context.Context, ident identity.IdentityContext, fn func(repositories.DBTX) error) error {
		return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", ident.TenantID.String()); err != nil {
				return err
			}
			return fn(tx)
		})
	}
}

func TestReadinessQueriesRealPostgres(t *testing.T) {
	handler, _ := application(t, testutil.DatabaseURL(t))
	status, body := testutil.Do(t, handler, http.MethodGet, "/api/v1/health/ready", nil)
	if status != http.StatusOK || body["status"] != "UP" {
		t.Fatalf("ready: %d %v", status, body)
	}
}

func TestMigrationsRejectUnreachableDatabase(t *testing.T) {
	if err := migrations.Up(context.Background(), "postgresql://postgres:postgres@127.0.0.1:1/none?connect_timeout=1"); err == nil {
		t.Fatal("an unreachable database fails")
	}
}
