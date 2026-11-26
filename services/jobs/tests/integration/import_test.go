package integration_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"jobs/internal/app"
	"jobs/internal/config"
	"jobs/internal/data/repositories"
	"jobs/internal/identity"
	"jobs/internal/importer"
	"jobs/internal/models"
	"jobs/tests/testutil"
)

func importedJob(id, title string) models.ImportedJob {
	posted := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	return models.ImportedJob{
		JobInput: models.JobInput{Title: title, Company: "Northwind", City: "Iași", Country: "Romania", WorkMode: "hybrid",
			MustHave: []string{"Kubernetes"}, URL: "https://example.com/" + id, ReadFrom: "ats_api",
			Qualifications: []string{"3+ years running Kubernetes"}, Bonuses: []string{"Terraform"}, AboutCompany: "We build payments.",
			Perks: []string{"Learning budget"}},
		Areas: []string{"DevOps"}, Origin: "greenhouse", ExternalID: id, Board: "northwind", PostedAt: &posted,
	}
}

func TestImportedJobsAgainstPostgres(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgJobRepository(pool)
	me := identity.New(uuid.New(), uuid.NewString())
	first := time.Now().Add(-time.Hour)

	for _, job := range []models.ImportedJob{importedJob("1", "DevOps Engineer"), importedJob("2", "SRE")} {
		if created, err := repo.Import(ctx, job, first); err != nil || !created {
			t.Fatalf("a new job: %v %v", created, err)
		}
	}
	jobs, err := repo.List(ctx, me, models.JobFilter{Areas: []string{"devops"}})
	if err != nil || len(jobs) != 2 || jobs[0].Source != "import" || jobs[0].Origin != "greenhouse" || jobs[0].PostedAt == nil ||
		jobs[0].ClosedAt != nil || jobs[0].AddedByMe {
		t.Fatalf("imported jobs are listed for everyone: %+v %v", jobs, err)
	}
	devops := jobs[slices.IndexFunc(jobs, func(j models.Job) bool { return j.Title == "DevOps Engineer" })]
	if !slices.Equal(devops.Qualifications, []string{"3+ years running Kubernetes"}) || !slices.Equal(devops.Bonuses, []string{"Terraform"}) ||
		devops.AboutCompany != "We build payments." || !slices.Equal(devops.Perks, []string{"Learning budget"}) {
		t.Fatalf("the ad's own lines are kept: %+v", devops)
	}
	if set, err := repo.SetRequirements(ctx, me, devops.ID, models.RequirementsInput{MustHave: []string{"Terraform"}}); err != nil || set == nil {
		t.Fatalf("an imported job's analysis is kept on the job: %+v %v", set, err)
	}

	second := time.Now()
	changed := importedJob("1", "Senior DevOps Engineer")
	changed.PostedAt = nil
	if created, err := repo.Import(ctx, changed, second); err != nil || created {
		t.Fatalf("a known job is updated: %v %v", created, err)
	}
	updated, _ := repo.Get(ctx, me, devops.ID)
	if updated.Title != "Senior DevOps Engineer" || !slices.Equal(updated.MustHave, []string{"Terraform"}) || updated.RequirementsFrom != "analysis" || updated.PostedAt == nil {
		t.Fatalf("the update keeps the analysis and the posting date: %+v", updated)
	}

	if closed, err := repo.Close(ctx, "greenhouse", "northwind", second); err != nil || closed != 1 {
		t.Fatalf("the job the board stopped listing closes: %d %v", closed, err)
	}
	open, _ := repo.List(ctx, me, models.JobFilter{})
	if len(open) != 1 || open[0].Title != "Senior DevOps Engineer" {
		t.Fatalf("closed jobs leave the list: %q", titles(open))
	}
	all, _ := repo.List(ctx, me, models.JobFilter{IDs: []uuid.UUID{devops.ID, jobs[0].ID, jobs[1].ID}})
	if len(all) != 2 || !slices.ContainsFunc(all, func(j models.Job) bool { return j.ClosedAt != nil }) {
		t.Fatalf("asked by id, closed jobs are still there: %+v", all)
	}
	if limited, _ := repo.List(ctx, me, models.JobFilter{IDs: []uuid.UUID{jobs[0].ID, jobs[1].ID}, Limit: 1}); len(limited) != 1 {
		t.Fatalf("limit: %q", titles(limited))
	}
	reopened := importedJob("2", "SRE")
	if created, err := repo.Import(ctx, reopened, time.Now()); err != nil || created {
		t.Fatalf("a job listed again: %v %v", created, err)
	}
	if open, _ := repo.List(ctx, me, models.JobFilter{}); len(open) != 2 {
		t.Fatalf("a job listed again opens again: %q", titles(open))
	}

	samePath := importedJob("2", "Another company's SRE")
	samePath.Board = "another"
	if created, err := repo.Import(ctx, samePath, time.Now()); err != nil || !created {
		t.Fatalf("the same job id on another company's board is another job: %v %v", created, err)
	}
	if closed, err := repo.Close(ctx, "greenhouse", "another", time.Now().Add(time.Minute)); err != nil || closed != 1 {
		t.Fatalf("closing one board leaves the other's job open: %d %v", closed, err)
	}
	other := importedJob("9", "Data Engineer")
	other.Origin, other.Board = "lever", "elsewhere"
	if _, err := repo.Import(ctx, other, time.Now()); err != nil {
		t.Fatal(err)
	}
	if closed, err := repo.CloseUnlisted(ctx, []string{"greenhouse"}, []string{"northwind"}); err != nil || closed != 1 {
		t.Fatalf("the jobs of a board no longer listed close: %d %v", closed, err)
	}
	if open, _ := repo.List(ctx, me, models.JobFilter{}); len(open) != 2 || slices.ContainsFunc(open, func(j models.Job) bool { return j.Title == "Data Engineer" }) {
		t.Fatalf("open after closing the unlisted: %q", titles(open))
	}

	fresh, err := repo.Fresh(ctx, "greenhouse", "northwind", time.Now().Add(-time.Hour))
	if err != nil || !fresh["1"] || !fresh["2"] || len(fresh) != 2 {
		t.Fatalf("the jobs read in the last hour: %v %v", fresh, err)
	}
	if stale, _ := repo.Fresh(ctx, "greenhouse", "northwind", time.Now().Add(time.Hour)); len(stale) != 0 {
		t.Fatalf("none read after now: %v", stale)
	}
	later := time.Now().Add(2 * time.Hour)
	if err := repo.Seen(ctx, "greenhouse", "northwind", []string{"1"}, later); err != nil {
		t.Fatal(err)
	}
	if closed, err := repo.Close(ctx, "greenhouse", "northwind", later); err != nil || closed != 1 {
		t.Fatalf("a job marked seen stays open, the other closes: %d %v", closed, err)
	}
	if _, err := repo.Import(ctx, models.ImportedJob{JobInput: models.JobInput{Title: "No key"}}, time.Now()); err == nil {
		t.Fatal("an imported job needs its origin, id and board")
	}
}

func TestImportLockAgainstPostgres(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	lock := importer.PgLock(pool)
	unlock, ok, err := lock(ctx)
	if err != nil || !ok {
		t.Fatalf("lock: %v %v", ok, err)
	}
	if _, ok, err := lock(ctx); err != nil || ok {
		t.Fatalf("a second importer waits for the next run: %v %v", ok, err)
	}
	unlock()
	again, ok, err := lock(ctx)
	if err != nil || !ok {
		t.Fatalf("free again: %v %v", ok, err)
	}
	again()
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, ok, err := lock(cancelled); err == nil || ok {
		t.Fatalf("no lock without a connection: %v %v", ok, err)
	}
}

func TestFollowedBoardsAgainstPostgres(t *testing.T) {
	handler, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgBoardRepository(pool)
	if boards, err := repo.Followed(ctx); err != nil || len(boards) != 0 {
		t.Fatalf("none followed yet: %v %v", boards, err)
	}
	first, created, err := repo.Follow(ctx, models.Board{Company: "Northwind", Platform: "lever", Board: "northwind"})
	if err != nil || !created || first.FollowedAt == nil || first.Company != "Northwind" {
		t.Fatalf("follow: %+v %v %v", first, created, err)
	}
	again, created, err := repo.Follow(ctx, models.Board{Company: "Renamed", Platform: "lever", Board: "northwind"})
	if err != nil || created || again.Company != "Northwind" || !again.FollowedAt.Equal(*first.FollowedAt) {
		t.Fatalf("a board followed again stays as it was: %+v %v %v", again, created, err)
	}
	if boards, _ := repo.Followed(ctx); len(boards) != 1 {
		t.Fatalf("followed: %+v", boards)
	}

	status, body := testutil.Do(t, handler, http.MethodPost, "/api/v1/boards", map[string]any{"url": "https://jobs.ashbyhq.com/linear/1", "company": "Linear"})
	if status != http.StatusCreated || body["board"] != "linear" {
		t.Fatalf("follow through the API: %d %v", status, body)
	}
	if status, body := testutil.Do(t, handler, http.MethodGet, "/api/v1/boards/of?url=https://jobs.ashbyhq.com/linear/2", nil); status != http.StatusOK || body["imported"] != true {
		t.Fatalf("of: %d %v", status, body)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.Followed(cancelled); err == nil {
		t.Fatal("a cancelled read fails")
	}
	if _, _, err := repo.Follow(cancelled, models.Board{Company: "X", Platform: "lever", Board: "x"}); err == nil {
		t.Fatal("a cancelled follow fails")
	}
}

func TestBuildStartsTheImporterOnlyWhenAsked(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	cfg := config.Defaults()
	t.Setenv("JOBS_IMPORT", "")
	if service, err := app.Build(&cfg, pool); err != nil || service.Importer != nil || service.Handler == nil {
		t.Fatalf("off: %+v %v", service, err)
	}
	t.Setenv("JOBS_IMPORT", "on")
	service, err := app.Build(&cfg, pool)
	if err != nil || service.Importer == nil || service.Interval != importer.Interval || len(service.Importer.Listed()) == 0 {
		t.Fatalf("on: %+v %v", service, err)
	}
	t.Setenv("JOBS_IMPORT_INTERVAL", "soon")
	if _, err := app.Build(&cfg, pool); err == nil {
		t.Fatal("a bad interval fails")
	}
}
