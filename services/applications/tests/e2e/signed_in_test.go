package e2e_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"forge.local/platform-auth-go/authtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"applications/tests/testutil"
)

// bearer is the token the contract tests call with: a person who is a tenant of their own.
var bearer string

// asPerson starts the service as the stack does, with platform auth on and a database role row-level security
// applies to, trusting a test issuer whose token for a new tenant the calls send.
func asPerson(t *testing.T, schema testutil.TestSchema) []string {
	t.Helper()
	issuer := authtest.NewIssuer()
	t.Cleanup(issuer.Close)
	tenant := uuid.NewString()
	bearer = issuer.Token(issuer.Options("svc-e2e", "person", tenant))
	t.Cleanup(func() { bearer = "" })
	return []string{"DATABASE_URL=" + appRole(t, schema), "APP__SECURITY__AUTH__ENABLED=true",
		"GATEKEEPER_ISSUER=" + issuer.URL(), "SERVICE_AUDIENCE=svc-e2e"}
}

// appRole is a URL to the schema as a new role row-level security applies to, with the grants
// deploy/compose/db-app-role.sh gives a service's runtime role.
func appRole(t *testing.T, schema testutil.TestSchema) string {
	t.Helper()
	ctx := context.Background()
	role, password := "hoply_e2e_"+strings.ReplaceAll(uuid.NewString(), "-", "")[:16], uuid.NewString()
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
	t.Cleanup(func() {
		for _, statement := range []string{"DROP OWNED BY " + name, "DROP ROLE " + name} {
			if _, err := schema.Admin.Exec(context.Background(), statement); err != nil {
				t.Errorf("%s: %v", statement, err)
			}
		}
	})
	parsed, err := url.Parse(schema.URL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.User = url.UserPassword(role, password)
	return parsed.String()
}
