package integration_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"profile/internal/data/repositories"
	"profile/internal/identity"
	"profile/migrations"
	"profile/tests/testutil"
)

func TestProfileV2MigrationConvertsExistingData(t *testing.T) {
	ctx := context.Background()
	schema := testutil.NewSchema(t)
	migrator := openMigrator(t, schema.URL, migrations.Embedded())
	if _, err := migrator.UpTo(ctx, 100); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, schema.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}

	owner := identity.New(uuid.New(), uuid.NewString())
	var profileID uuid.UUID
	if err := pool.QueryRow(ctx, "INSERT INTO profiles (customer_id, subject, name, headline, skills) "+
		"VALUES ($1, $2, 'Ana', 'Engineer', '{Go,Vue}') RETURNING id", owner.TenantID, owner.Subject).Scan(&profileID); err != nil {
		t.Fatal(err)
	}
	exec("INSERT INTO profile_experience (profile_id, role, start_year, end_year, tools, bullets) VALUES "+
		"($1, 'Developer', '2016', '2019', '{PHP}', '{\"Built billing\",\"Cut costs\"}')", profileID)
	exec("INSERT INTO profile_experience (profile_id, role) VALUES ($1, 'Lead')", profileID)
	education := []struct{ degree, years, level, start, end string }{
		{"BSc Computer Science", "2016 – 2020", "bachelor", "2016", "2020"},
		{"Master of Science", "2020-2022", "master", "2020", "2022"},
		{"Licență în Informatică", "2012", "bachelor", "2012", "2012"},
		{"bachelor's degree", "", "bachelor", "", ""},
		{"MSc", "since 2023", "master", "2023", "2023"},
		{"PhD", "2001 to 2005, defended 2007", "doctorate", "2001", "2007"},
		{"Doctor of Medicine", "n/a", "doctorate", "", ""},
		{"High school", "graduated in '09", "other", "", ""},
	}
	for _, e := range education {
		exec("INSERT INTO profile_education (profile_id, degree, school, years) VALUES ($1, $2, 'Uni', $3)", profileID, e.degree, e.years)
	}
	exec("INSERT INTO profile_languages (profile_id, name, level) VALUES ($1, 'English', 'C1')", profileID)

	if _, err := migrator.Up(ctx); err != nil {
		t.Fatal(err)
	}
	repo := repositories.NewPgProfileRepository(pool)
	profile, err := repo.Ensure(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if profile.ID != profileID || profile.Name != "Ana" || !slices.Equal(profile.Skills, []string{"Go", "Vue"}) ||
		profile.City != "" || profile.NoticePeriodWeeks != nil || len(profile.WorkModes) != 0 || len(profile.Projects) != 0 ||
		len(profile.Certifications) != 0 || len(profile.Languages) != 1 {
		t.Fatalf("profile keeps its data with empty new fields: %+v", profile)
	}
	lead, developer := profile.Experience[0], profile.Experience[1]
	if lead.Role != "Lead" || lead.Start != "" || lead.End != "" || len(lead.Bullets) != 0 {
		t.Fatalf("experience without years or bullets: %+v", lead)
	}
	if developer.Start != "2016-01" || developer.End != "2019-01" || !slices.Equal(developer.Tools, []string{"PHP"}) ||
		fmt.Sprint(developer.Bullets) != "[{Built billing  []} {Cut costs  []}]" || developer.Bullets[0].Skills == nil {
		t.Fatalf("experience months and bullets: %+v", developer)
	}
	if len(profile.Education) != len(education) {
		t.Fatalf("education rows: %+v", profile.Education)
	}
	for i, e := range education {
		got := profile.Education[i]
		if got.Degree != e.degree || got.Level != e.level || got.StartYear != e.start || got.EndYear != e.end || got.Field != "" || got.School != "Uni" {
			t.Fatalf("education %q %q: %+v", e.degree, e.years, got)
		}
	}

	exec("INSERT INTO profile_education (profile_id, level, field, degree, start_year, end_year) "+
		"VALUES ($1, 'high_school', 'Science', '', '2005', '2009')", profileID)
	exec("INSERT INTO profile_projects (profile_id, name) VALUES ($1, 'dropped on down')", profileID)
	if _, err := migrator.DownTo(ctx, 100); err != nil {
		t.Fatal(err)
	}
	var startYear, endYear string
	var bullets []string
	if err := pool.QueryRow(ctx, "SELECT start_year, end_year, bullets FROM profile_experience WHERE role = 'Developer'").
		Scan(&startYear, &endYear, &bullets); err != nil || startYear != "2016" || endYear != "2019" ||
		!slices.Equal(bullets, []string{"Built billing", "Cut costs"}) {
		t.Fatalf("down restores experience: %q %q %q %v", startYear, endYear, bullets, err)
	}
	rows, err := pool.Query(ctx, "SELECT degree, years FROM profile_education ORDER BY position")
	if err != nil {
		t.Fatal(err)
	}
	var restored []string
	for rows.Next() {
		var degree, years string
		if err := rows.Scan(&degree, &years); err != nil {
			t.Fatal(err)
		}
		restored = append(restored, degree+" | "+years)
	}
	want := []string{"BSc Computer Science | 2016 – 2020", "Master of Science | 2020 – 2022", "Licență în Informatică | 2012",
		"bachelor's degree | ", "MSc | 2023", "PhD | 2001 – 2007", "Doctor of Medicine | ", "High school | ",
		"High School, Science | 2005 – 2009"}
	if rows.Err() != nil || !slices.Equal(restored, want) {
		t.Fatalf("down restores education:\n got %q\nwant %q", restored, want)
	}
	if _, err := migrator.Up(ctx); err != nil {
		t.Fatalf("up after down: %v", err)
	}
}
