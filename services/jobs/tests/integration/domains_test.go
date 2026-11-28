package integration_test

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"jobs/migrations"
	"jobs/tests/testutil"
)

// The jobs there are when domains arrive get theirs from their areas, a person's own jobs included, and the
// importer reads every imported job again to classify it with the new areas.
func TestDomainsMigrationClassifiesTheJobsThereAre(t *testing.T) {
	ctx := context.Background()
	schema := testutil.NewSchema(t)
	migrator := openMigrator(t, schema.URL, migrations.Embedded())
	if _, err := migrator.UpTo(ctx, 107); err != nil {
		t.Fatal(err)
	}
	db, err := pgxpool.New(ctx, schema.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	owner := uuid.New()
	for _, row := range []struct {
		title  string
		areas  []string
		source string
		owner  any
	}{
		{"DevOps Engineer", []string{"DevOps"}, "import", nil},
		{"Accountant", []string{"Accounting"}, "import", nil},
		{"Compliance Controller", []string{"Finance & control", "Risk & compliance"}, "import", nil},
		{"Consilier bancar", []string{"Banking"}, "link", owner},
	} {
		if _, err := db.Exec(ctx, "INSERT INTO jobs (customer_id, subject, title, url, source, read_from, areas, origin, "+
			"external_id, board, read_at) VALUES ($1, CASE WHEN $1::uuid IS NULL THEN NULL ELSE 'ana' END, $2, '', $3, 'text', $4, "+
			"CASE WHEN $3 = 'import' THEN 'workday' ELSE '' END, $2, 'board', now())",
			row.owner, row.title, row.source, row.areas); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrations.Up(ctx, schema.URL); err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query(ctx, "SELECT title, domains, areas, read_at IS NULL OR source <> 'import' FROM jobs ORDER BY title")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := map[string][2][]string{
		"Accountant":            {{"accounting"}, {}},
		"Compliance Controller": {{"finance"}, {"Risk & compliance"}},
		"Consilier bancar":      {{"banking"}, {}},
		"DevOps Engineer":       {{"it"}, {"DevOps"}},
	}
	for rows.Next() {
		var title string
		var domains, areas []string
		var reread bool
		if err := rows.Scan(&title, &domains, &areas, &reread); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(domains, want[title][0]) || !slices.Equal(areas, want[title][1]) || !reread {
			t.Errorf("%s: %v %v read again %v", title, domains, areas, reread)
		}
		delete(want, title)
	}
	if len(want) > 0 || rows.Err() != nil {
		t.Fatalf("missing %v: %v", want, rows.Err())
	}
	var forced bool
	if err := db.QueryRow(ctx, "SELECT relforcerowsecurity FROM pg_class WHERE oid = 'jobs'::regclass").Scan(&forced); err != nil || !forced {
		t.Fatalf("row-level security is forced again: %v %v", forced, err)
	}
}
