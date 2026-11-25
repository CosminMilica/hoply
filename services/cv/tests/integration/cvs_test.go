package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"cv/internal/app"
	"cv/internal/apperrors"
	"cv/internal/config"
	"cv/internal/data/repositories"
	"cv/internal/identity"
	"cv/internal/models"
	"cv/internal/writer"
	"cv/tests/testutil"
)

const jobID = "00000000-0000-4000-8000-000000000001"

func testdata(t *testing.T, name string) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile("../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// seedSources serves the sample profile and the first seed job.
type seedSources struct {
	profile json.RawMessage
	job     json.RawMessage
}

func newSeedSources(t *testing.T) seedSources {
	var jobs []json.RawMessage
	if err := json.Unmarshal(testdata(t, "jobs.json"), &jobs); err != nil {
		t.Fatal(err)
	}
	return seedSources{profile: testdata(t, "profile.json"), job: jobs[0]}
}

func (s seedSources) Profile(context.Context) (models.Profile, json.RawMessage, error) {
	var profile models.Profile
	err := json.Unmarshal(s.profile, &profile)
	return profile, s.profile, err
}

func (s seedSources) Job(_ context.Context, id uuid.UUID) (models.Job, json.RawMessage, error) {
	var job models.Job
	if err := json.Unmarshal(s.job, &job); err != nil || job.ID != id {
		return job, nil, apperrors.NewNotFound("Job", id.String())
	}
	return job, s.job, nil
}

func newCv(t *testing.T, repo *repositories.PgCvRepository, ident identity.IdentityContext) *models.Cv {
	t.Helper()
	sources := newSeedSources(t)
	cv, err := repo.Create(context.Background(), ident, repositories.NewCv{
		JobID:           uuid.MustParse(jobID),
		Request:         models.CvRequest{JobID: jobID, SkipRoles: []string{}, Focus: []string{}, Confirmed: []int{1}},
		ProfileSnapshot: sources.profile,
		JobSnapshot:     sources.job,
		JobTitle:        "Snowflake Administrator DevOps",
		Company:         "Capgemini Romania",
		FileName:        "Andrei_Popescu_CV_Capgemini_Romania.pdf",
	})
	if err != nil {
		t.Fatal(err)
	}
	return cv
}

func TestCvRepositoryIsScopedToTheCaller(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgCvRepository(pool).WithScope(bindTenant(pool))
	me := identity.New(uuid.New(), uuid.NewString())
	other := identity.New(uuid.New(), uuid.NewString())
	first, second := newCv(t, repo, me), newCv(t, repo, me)
	if first.Status != models.StatusQueued || first.Tailored != nil || first.Preview != nil || first.Highlights != nil {
		t.Fatalf("a new CV is queued and unwritten: %+v", first)
	}
	if found, err := repo.Get(ctx, other, first.ID); err != nil || found != nil {
		t.Fatalf("another person's read: %v %v", found, err)
	}
	if items, err := repo.List(ctx, other, nil); err != nil || len(items) != 0 {
		t.Fatalf("another person's list: %v %v", items, err)
	}
	if removed, err := repo.Delete(ctx, other, first.ID); err != nil || removed {
		t.Fatalf("another person's delete: %v %v", removed, err)
	}
	job := uuid.MustParse(jobID)
	items, err := repo.List(ctx, me, &job)
	if err != nil || len(items) != 2 || items[0].ID != second.ID {
		t.Fatalf("newest first: %v %v", items, err)
	}
	unknown := uuid.New()
	if items, err := repo.List(ctx, me, &unknown); err != nil || len(items) != 0 {
		t.Fatalf("another job: %v %v", items, err)
	}
	if removed, err := repo.Delete(ctx, me, first.ID); err != nil || !removed {
		t.Fatalf("delete: %v %v", removed, err)
	}
	if found, err := repo.Get(ctx, me, first.ID); err != nil || found != nil {
		t.Fatalf("deleted: %v %v", found, err)
	}
}

func TestQueueTakesTheOldestAndRecoversWhatAStoppedWorkerLeft(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgCvRepository(pool)
	me := identity.New(uuid.New(), uuid.NewString())
	first, second := newCv(t, repo, me), newCv(t, repo, me)

	claimed, err := repo.Claim(ctx)
	if err != nil || claimed.ID != first.ID || claimed.Profile.Name != "Andrei Popescu" || claimed.Job.Title != "Snowflake Administrator DevOps" ||
		claimed.Request.Confirmed[0] != 1 {
		t.Fatalf("oldest first, with its snapshots: %+v %v", claimed, err)
	}
	if next, err := repo.Claim(ctx); err != nil || next.ID != second.ID {
		t.Fatalf("then the next: %v %v", next, err)
	}
	if none, err := repo.Claim(ctx); err != nil || none != nil {
		t.Fatalf("an empty queue: %v %v", none, err)
	}
	if err := repo.Checking(ctx, first.ID, &models.Preview{From: "a", To: "b"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uuid.UUID{first.ID, second.ID} {
		if cv, _ := repo.Get(ctx, me, id); cv.Status != models.StatusQueued {
			t.Fatalf("recovered to queued: %+v", cv)
		}
	}

	claimed, _ = repo.Claim(ctx)
	if err := repo.Checking(ctx, claimed.ID, &models.Preview{From: "from", To: "to"}); err != nil {
		t.Fatal(err)
	}
	if cv, _ := repo.Get(ctx, me, claimed.ID); cv.Status != models.StatusChecking || cv.Preview.To != "to" {
		t.Fatalf("checking: %+v", cv)
	}
	written := models.Written{Highlights: []string{"SQL"}, Tailored: models.CvDocument{Summary: "Tailored."}, Ats: models.Ats{Passed: 1, Total: 2}}
	until := time.Date(2026, 11, 25, 18, 0, 0, 0, time.UTC)
	by := models.Authorship{WrittenBy: "claude", Model: "claude-sonnet-5-5", Refused: 2, RulesReason: "", LimitUntil: &until}
	if err := repo.Complete(ctx, claimed.ID, written, by); err != nil {
		t.Fatal(err)
	}
	ready, _ := repo.Get(ctx, me, claimed.ID)
	if ready.Status != models.StatusReady || ready.Tailored.Summary != "Tailored." || ready.Ats.Total != 2 || ready.Highlights[0] != "SQL" || ready.Profile == nil ||
		*ready.WrittenBy != "claude" || ready.Model != "claude-sonnet-5-5" || ready.Refused != 2 || !ready.LimitUntil.Equal(until) {
		t.Fatalf("ready: %+v", ready)
	}

	failing, _ := repo.Claim(ctx)
	if err := repo.Fail(ctx, failing.ID); err != nil {
		t.Fatal(err)
	}
	if cv, _ := repo.Get(ctx, me, failing.ID); cv.Status != models.StatusFailed {
		t.Fatalf("failed: %+v", cv)
	}
}

func TestQueueGivesUpAfterMaxAttempts(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgCvRepository(pool)
	me := identity.New(uuid.New(), uuid.NewString())
	cv := newCv(t, repo, me)
	for attempt := 1; attempt <= repositories.MaxAttempts; attempt++ {
		if claimed, err := repo.Claim(ctx); err != nil || claimed == nil {
			t.Fatalf("attempt %d: %v %v", attempt, claimed, err)
		}
		if err := repo.Recover(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if found, _ := repo.Get(ctx, me, cv.ID); found.Status != models.StatusFailed {
		t.Fatalf("given up: %+v", found)
	}
}

func TestQueueFailuresSurface(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	repo := repositories.NewPgCvRepository(pool)
	pool.Close()
	ctx := context.Background()
	if _, err := repo.Claim(ctx); err == nil {
		t.Fatal("claim on a closed pool")
	}
	if err := repo.Fail(ctx, uuid.New()); err == nil {
		t.Fatal("fail on a closed pool")
	}
	if _, err := repo.Create(ctx, identity.Dev(), repositories.NewCv{}); err == nil {
		t.Fatal("create on a closed pool")
	}
}

func TestServiceWritesACvThroughTheWorker(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	cfg := config.Defaults()
	service := app.BuildWith(&cfg, pool, newSeedSources(t), writer.New(nil), nil)
	status, cv := testutil.Do(t, service.Handler, http.MethodPost, "/api/v1/cvs", map[string]any{"job_id": jobID, "confirmed": []int{0}})
	if status != http.StatusAccepted || cv["status"] != "queued" {
		t.Fatalf("queued: %d %v", status, cv)
	}
	if !service.Worker.Next(context.Background()) {
		t.Fatal("the worker writes it")
	}
	status, ready := testutil.Do(t, service.Handler, http.MethodGet, "/api/v1/cvs/"+cv["id"].(string), nil)
	ats, _ := ready["ats"].(map[string]any)
	tailored, _ := ready["tailored"].(map[string]any)
	preview, _ := ready["preview"].(map[string]any)
	if status != http.StatusOK || ready["status"] != "ready" || ats["total"] != float64(12) || tailored["name"] != "Andrei Popescu" ||
		preview["from"] != "Built and maintained deployment pipelines for internal services." {
		t.Fatalf("ready: %d %v", status, ready)
	}
	if status, questions := testutil.Do(t, service.Handler, http.MethodGet, "/api/v1/jobs/"+jobID+"/questions", nil); status != http.StatusOK ||
		questions["base_score"] == nil {
		t.Fatalf("questions: %d %v", status, questions)
	}
}
