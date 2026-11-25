package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"jobs/internal/data/repositories"
	"jobs/internal/identity"
	"jobs/internal/models"
	"jobs/internal/services"
	"jobs/tests/testutil"
)

const jobsPath = "/api/v1/jobs"

func titles(jobs []models.Job) []string {
	out := make([]string, len(jobs))
	for i, job := range jobs {
		out[i] = job.Title
	}
	return out
}

// catalogFixtures are synthetic catalog jobs, oldest last; the migration seeds
// none.
var catalogFixtures = []struct {
	title, company, city, workMode, level string
	mustHave, areas                       []string
	sample                                bool
}{
	{"Snowflake Administrator DevOps", "Northwind Data", "Iași", "onsite", "senior", []string{"Snowflake", "SQL"}, []string{"DevOps", "Data"}, false},
	{"DevOps Engineer", "Fabrikam", "Bucharest", "hybrid", "mid", []string{"Kubernetes", "Docker"}, []string{"DevOps"}, true},
	{"Software Engineer", "Contoso", "Bucharest", "hybrid", "mid", []string{"Java", "Python"}, []string{"Backend"}, true},
	{"System Administrator", "Tailspin Systems", "Cluj-Napoca", "onsite", "mid", []string{"Linux", "Active Directory"}, []string{"Sysadmin"}, true},
}

func insertCatalog(t *testing.T, pool *pgxpool.Pool) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, len(catalogFixtures))
	for i, job := range catalogFixtures {
		err := pool.QueryRow(context.Background(), "INSERT INTO jobs (title, company, city, country, work_mode, level, "+
			"years_required, must_have, responsibilities, about_company, product, perks, areas, source, sample, created_at) "+
			"VALUES ($1, $2, $3, 'Romania', $4, $5, 3, $6, '{Run things}', 'About the company', 'a product', '{Learning budget}', $7, "+
			"'catalog', $8, NOW() - make_interval(mins => $9)) RETURNING id",
			job.title, job.company, job.city, job.workMode, job.level, job.mustHave, job.areas, job.sample, i+1).Scan(&ids[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	return ids
}

func TestCatalogStartsEmpty(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	repo := repositories.NewPgJobRepository(pool)
	ctx := context.Background()
	if jobs, err := repo.List(ctx, identity.Dev(), models.JobFilter{}); err != nil || len(jobs) != 0 {
		t.Fatalf("the migration seeds no jobs: %q %v", titles(jobs), err)
	}
	ids := insertCatalog(t, pool)
	jobs, err := repo.List(ctx, identity.New(uuid.New(), uuid.NewString()), models.JobFilter{})
	if err != nil || !slices.Equal(titles(jobs), []string{"Snowflake Administrator DevOps", "DevOps Engineer", "Software Engineer", "System Administrator"}) {
		t.Fatalf("catalog order: %q %v", titles(jobs), err)
	}
	first := jobs[0]
	if first.ID != ids[0] || first.Sample || first.Source != "catalog" || first.ReadFrom != "" || first.AddedByMe || first.URL != "" ||
		*first.YearsRequired != 3 || !slices.Equal(first.Areas, []string{"DevOps", "Data"}) || first.NiceToHave == nil ||
		len(first.NiceToHave) != 0 || first.Product != "a product" || first.AboutCompany == "" || len(first.Perks) != 1 {
		t.Fatalf("catalog job: %+v", first)
	}
}

func TestJobRepositoryAgainstPostgres(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	insertCatalog(t, pool)
	repo := repositories.NewPgJobRepository(pool).WithScope(bindTenant(pool))
	me := identity.New(uuid.New(), uuid.NewString())
	other := identity.New(me.TenantID, uuid.NewString())

	five := 5
	mine, err := repo.Create(ctx, me, models.JobInput{Title: "Remote Go Engineer", Company: "Acme", City: "Timișoara",
		Country: "Romania", WorkMode: "remote", Level: "senior", YearsRequired: &five, MustHave: []string{"Go", "Kafka"},
		NiceToHave: []string{"Rust"}, Responsibilities: []string{"Build"}, About: "About", URL: "https://acme.dev/jobs/1",
		Source: "link", ReadFrom: "json_ld"})
	if err != nil || !mine.AddedByMe || mine.Sample || mine.Source != "link" || mine.ReadFrom != "json_ld" || *mine.YearsRequired != 5 ||
		mine.Perks == nil || mine.Areas == nil || mine.AboutCompany != "" || mine.CreatedAt.IsZero() {
		t.Fatalf("create: %+v %v", mine, err)
	}
	theirs, err := repo.Create(ctx, other, models.JobInput{Title: "Their job", Source: "text", ReadFrom: "text"})
	if err != nil || theirs.YearsRequired != nil || theirs.MustHave == nil {
		t.Fatalf("create other: %+v %v", theirs, err)
	}

	all, err := repo.List(ctx, me, models.JobFilter{})
	if err != nil || len(all) != 5 || all[0].ID != mine.ID || slices.Contains(titles(all), "Their job") {
		t.Fatalf("my list: %q %v", titles(all), err)
	}
	if found, err := repo.Get(ctx, me, theirs.ID); err != nil || found != nil {
		t.Fatalf("another person's job is invisible: %v %v", found, err)
	}
	if found, err := repo.Get(ctx, other, mine.ID); err != nil || found != nil {
		t.Fatalf("my job is invisible to others: %v %v", found, err)
	}
	catalogID := all[1].ID
	if found, err := repo.Get(ctx, other, catalogID); err != nil || found == nil || found.AddedByMe {
		t.Fatalf("the catalog is visible to everyone: %v %v", found, err)
	}

	cases := []struct {
		name   string
		filter services.JobQuery
		want   []string
	}{
		{"q title", services.JobQuery{Q: "devops"}, []string{"Snowflake Administrator DevOps", "DevOps Engineer"}},
		{"q company", services.JobQuery{Q: "CONTOSO"}, []string{"Software Engineer"}},
		{"q requirements", services.JobQuery{Q: "active directory"}, []string{"System Administrator"}},
		{"q nice to have", services.JobQuery{Q: "rust"}, []string{"Remote Go Engineer"}},
		{"q literal percent", services.JobQuery{Q: "100%"}, []string{}},
		{"city without diacritics", services.JobQuery{City: "Iasi"}, []string{"Snowflake Administrator DevOps"}},
		{"city alias", services.JobQuery{City: "București"}, []string{"DevOps Engineer", "Software Engineer"}},
		{"city prefix", services.JobQuery{City: "cluj"}, []string{"System Administrator"}},
		{"city and remote", services.JobQuery{City: "iasi", IncludeRemote: "true"}, []string{"Remote Go Engineer", "Snowflake Administrator DevOps"}},
		{"remote without city", services.JobQuery{IncludeRemote: "true"}, []string{"Remote Go Engineer", "Snowflake Administrator DevOps",
			"DevOps Engineer", "Software Engineer", "System Administrator"}},
		{"country", services.JobQuery{Country: "România"}, []string{"Remote Go Engineer", "Snowflake Administrator DevOps",
			"DevOps Engineer", "Software Engineer", "System Administrator"}},
		{"other country", services.JobQuery{Country: "Germany"}, []string{}},
		{"areas", services.JobQuery{Areas: []string{"data", "Sysadmin"}}, []string{"Snowflake Administrator DevOps", "System Administrator"}},
		{"level", services.JobQuery{Level: "senior"}, []string{"Remote Go Engineer", "Snowflake Administrator DevOps"}},
		{"combined", services.JobQuery{Q: "engineer", City: "bucharest", Areas: []string{"backend"}, Level: "mid"}, []string{"Software Engineer"}},
	}
	for _, tc := range cases {
		filter, err := services.CleanQuery(tc.filter)
		if err != nil {
			t.Fatal(err)
		}
		jobs, err := repo.List(ctx, me, filter)
		if err != nil || !slices.Equal(titles(jobs), tc.want) {
			t.Errorf("%s: %q %v, want %q", tc.name, titles(jobs), err, tc.want)
		}
	}

	if removed, err := repo.Delete(ctx, me, catalogID); err != nil || removed {
		t.Fatalf("a catalog job cannot be deleted: %v %v", removed, err)
	}
	if removed, err := repo.Delete(ctx, other, mine.ID); err != nil || removed {
		t.Fatalf("another person cannot delete my job: %v %v", removed, err)
	}
	if removed, err := repo.Delete(ctx, me, mine.ID); err != nil || !removed {
		t.Fatalf("delete my job: %v %v", removed, err)
	}
	if removed, err := repo.Delete(ctx, me, mine.ID); err != nil || removed {
		t.Fatalf("delete twice: %v %v", removed, err)
	}
	if _, err := repo.Create(ctx, me, models.JobInput{Title: "Bad", Source: "catalog"}); err == nil {
		t.Fatal("the database refuses an owned catalog job")
	}
	pool.Close()
	if _, err := repo.List(ctx, me, models.JobFilter{}); err == nil {
		t.Fatal("a closed pool fails lists")
	}
	if _, err := repo.Get(ctx, me, catalogID); err == nil {
		t.Fatal("a closed pool fails gets")
	}
	if _, err := repo.Delete(ctx, me, catalogID); err == nil {
		t.Fatal("a closed pool fails deletes")
	}
}

func TestJobRoutesUseRealPostgres(t *testing.T) {
	handler, pool := application(t, isolatedSchema(t))
	catalog := insertCatalog(t, pool)
	status, list := testutil.Do(t, handler, http.MethodGet, jobsPath+"?city=Iasi&include_remote=true", nil)
	if status != http.StatusOK || list["total"] != float64(1) {
		t.Fatalf("list: %d %v", status, list)
	}
	item := list["items"].([]any)[0].(map[string]any)
	if item["sample"] != false || item["added_by_me"] != false || item["source"] != "catalog" || fmt.Sprint(item["nice_to_have"]) != "[]" {
		t.Fatalf("catalog shape: %v", item)
	}
	status, draft := testutil.Do(t, handler, http.MethodPost, jobsPath+"/read", map[string]any{"text": "Junior QA Engineer\n" +
		"Testăm aplicații web pentru clienți din toată lumea, din biroul nostru din Sibiu.\nCerințe\n- Cypress sau Playwright\n- 1 an de SQL"})
	if status != http.StatusOK || draft["city"] != "Sibiu" || draft["level"] != "junior" {
		t.Fatalf("read: %d %v", status, draft)
	}
	delete(draft, "host")
	status, created := testutil.Do(t, handler, http.MethodPost, jobsPath, draft)
	if status != http.StatusCreated || created["added_by_me"] != true || fmt.Sprint(created["must_have"]) != "[Cypress Playwright SQL]" {
		t.Fatalf("create from the draft: %d %v", status, created)
	}
	path := jobsPath + "/" + created["id"].(string)
	if status, got := testutil.Do(t, handler, http.MethodGet, path, nil); status != http.StatusOK || got["title"] != "Junior QA Engineer" {
		t.Fatalf("get: %d %v", status, got)
	}
	if status, _ := testutil.Do(t, handler, http.MethodDelete, path, nil); status != http.StatusNoContent {
		t.Fatalf("delete: %d", status)
	}
	if status, _ := testutil.Do(t, handler, http.MethodDelete, jobsPath+"/"+catalog[1].String(), nil); status != http.StatusNotFound {
		t.Fatalf("delete catalog: %d", status)
	}
}
