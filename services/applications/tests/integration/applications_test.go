package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"applications/internal/data/repositories"
	"applications/internal/identity"
	"applications/internal/models"
	"applications/tests/testutil"
)

func put(t *testing.T, repo *repositories.PgApplicationRepository, ident identity.IdentityContext, job uuid.UUID, status string) models.Application {
	t.Helper()
	application, err := repo.Put(context.Background(), ident, job, status)
	if err != nil {
		t.Fatalf("put %s: %v", status, err)
	}
	return *application
}

func after(later *time.Time, earlier time.Time) bool { return later != nil && later.After(earlier) }

func TestStatusRules(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	repo := repositories.NewPgApplicationRepository(pool).WithScope(bindTenant(pool))
	me := identity.New(uuid.New(), uuid.NewString())
	job := uuid.New()

	saved := put(t, repo, me, job, "saved")
	if saved.Status != "saved" || saved.SavedAt == nil || saved.CVReadyAt != nil || saved.AppliedAt != nil ||
		saved.JobID != job || !saved.CreatedAt.Equal(saved.UpdatedAt) {
		t.Fatalf("a new application takes the status sent: %+v", saved)
	}
	again := put(t, repo, me, job, "saved")
	if !again.SavedAt.Equal(*saved.SavedAt) || !again.UpdatedAt.After(saved.UpdatedAt) || !again.CreatedAt.Equal(saved.CreatedAt) {
		t.Fatalf("the same status keeps its timestamp, updated_at moves: %+v", again)
	}
	ready := put(t, repo, me, job, "cv_ready")
	if ready.Status != "cv_ready" || !ready.SavedAt.Equal(*saved.SavedAt) || !after(ready.CVReadyAt, *saved.SavedAt) || ready.AppliedAt != nil {
		t.Fatalf("cv_ready keeps saved_at: %+v", ready)
	}
	applied := put(t, repo, me, job, "applied")
	if applied.Status != "applied" || !after(applied.AppliedAt, *ready.CVReadyAt) || !applied.CVReadyAt.Equal(*ready.CVReadyAt) {
		t.Fatalf("applied: %+v", applied)
	}
	newCV := put(t, repo, me, job, "cv_ready")
	if newCV.Status != "applied" || !after(newCV.CVReadyAt, *applied.AppliedAt) || !newCV.AppliedAt.Equal(*applied.AppliedAt) {
		t.Fatalf("a new CV keeps the application applied: %+v", newCV)
	}
	resaved := put(t, repo, me, job, "saved")
	if resaved.Status != "saved" || !after(resaved.SavedAt, *saved.SavedAt) || !resaved.AppliedAt.Equal(*applied.AppliedAt) ||
		!resaved.CVReadyAt.Equal(*newCV.CVReadyAt) {
		t.Fatalf("any other change sets its timestamp and keeps the rest: %+v", resaved)
	}

	ctx := context.Background()
	if removed, err := repo.Delete(ctx, me, job); err != nil || !removed {
		t.Fatalf("delete: %v %v", removed, err)
	}
	if removed, err := repo.Delete(ctx, me, job); err != nil || removed {
		t.Fatalf("delete twice: %v %v", removed, err)
	}
	fresh := put(t, repo, me, job, "applied")
	if fresh.SavedAt != nil || fresh.CVReadyAt != nil || fresh.AppliedAt == nil || !fresh.CreatedAt.After(saved.CreatedAt) {
		t.Fatalf("saving again after a delete starts over: %+v", fresh)
	}
	if _, err := repo.Put(ctx, me, uuid.New(), "rejected"); err == nil {
		t.Fatal("the database refuses an unknown status")
	}
}

func TestListOrderAndOwnerIsolation(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgApplicationRepository(pool)
	me := identity.New(uuid.New(), uuid.NewString())
	sameSubjectOtherTenant := identity.New(uuid.New(), me.Subject)
	colleague := identity.New(uuid.New(), uuid.NewString())

	first, second, third := uuid.New(), uuid.New(), uuid.New()
	put(t, repo, me, first, "saved")
	put(t, repo, me, second, "saved")
	put(t, repo, me, third, "applied")
	put(t, repo, me, first, "cv_ready")
	list, err := repo.List(ctx, me)
	if err != nil {
		t.Fatal(err)
	}
	var order []uuid.UUID
	for _, a := range list {
		order = append(order, a.JobID)
	}
	if !slices.Equal(order, []uuid.UUID{first, third, second}) {
		t.Fatalf("most recently changed first: %v", order)
	}

	for _, stranger := range []identity.IdentityContext{sameSubjectOtherTenant, colleague} {
		if list, err := repo.List(ctx, stranger); err != nil || len(list) != 0 {
			t.Fatalf("another person's applications are invisible: %v %v", list, err)
		}
		if removed, err := repo.Delete(ctx, stranger, first); err != nil || removed {
			t.Fatalf("another person cannot delete: %v %v", removed, err)
		}
		theirs := put(t, repo, stranger, first, "applied")
		if theirs.SavedAt != nil {
			t.Fatalf("another person's PUT creates their own application: %+v", theirs)
		}
		if err := repo.Skip(ctx, stranger, second); err != nil {
			t.Fatal(err)
		}
		if err := repo.ClearSkips(ctx, stranger); err != nil {
			t.Fatal(err)
		}
	}
	mine, _ := repo.List(ctx, me)
	if len(mine) != 3 || mine[0].JobID != first || mine[0].Status != "cv_ready" {
		t.Fatalf("my applications are untouched: %+v", mine)
	}

	if ids, err := repo.Skips(ctx, me); err != nil || len(ids) != 0 {
		t.Fatalf("no skips yet: %v %v", ids, err)
	}
	for _, job := range []uuid.UUID{first, second, third, first} {
		if err := repo.Skip(ctx, me, job); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Skip(ctx, colleague, third); err != nil {
		t.Fatal(err)
	}
	if ids, err := repo.Skips(ctx, me); err != nil || !slices.Equal(ids, []uuid.UUID{first, third, second}) {
		t.Fatalf("most recently skipped first, skipping twice moves it up: %v %v", ids, err)
	}
	if err := repo.Unskip(ctx, me, third); err != nil {
		t.Fatal(err)
	}
	if err := repo.Unskip(ctx, me, uuid.New()); err != nil {
		t.Fatalf("unskipping a job that was not skipped is fine: %v", err)
	}
	if ids, _ := repo.Skips(ctx, me); !slices.Equal(ids, []uuid.UUID{first, second}) {
		t.Fatalf("after unskip: %v", ids)
	}
	if err := repo.ClearSkips(ctx, me); err != nil {
		t.Fatal(err)
	}
	if ids, _ := repo.Skips(ctx, me); len(ids) != 0 {
		t.Fatalf("after clear: %v", ids)
	}
	if ids, _ := repo.Skips(ctx, colleague); !slices.Equal(ids, []uuid.UUID{third}) {
		t.Fatalf("clearing my skips keeps a colleague's: %v", ids)
	}

	pool.Close()
	for name, call := range map[string]func() error{
		"list":   func() error { _, err := repo.List(ctx, me); return err },
		"put":    func() error { _, err := repo.Put(ctx, me, first, "saved"); return err },
		"delete": func() error { _, err := repo.Delete(ctx, me, first); return err },
		"skips":  func() error { _, err := repo.Skips(ctx, me); return err },
		"skip":   func() error { return repo.Skip(ctx, me, first) },
	} {
		if err := call(); err == nil {
			t.Fatalf("%s on a closed pool must fail", name)
		}
	}
}

func TestApplicationRoutesUseRealPostgres(t *testing.T) {
	handler, _ := application(t, isolatedSchema(t))
	job := uuid.NewString()
	status, saved := testutil.Do(t, handler, http.MethodPut, "/api/v1/applications/"+job, map[string]any{"status": "saved"})
	if status != http.StatusOK || saved["status"] != "saved" || saved["saved_at"] == nil || saved["applied_at"] != nil {
		t.Fatalf("PUT: %d %v", status, saved)
	}
	if _, list := testutil.Do(t, handler, http.MethodGet, "/api/v1/applications", nil); len(list["items"].([]any)) != 1 {
		t.Fatalf("GET: %v", list)
	}
	if status, _ := testutil.Do(t, handler, http.MethodPut, "/api/v1/skips/"+job, nil); status != http.StatusNoContent {
		t.Fatalf("skip: %d", status)
	}
	if _, skips := testutil.Do(t, handler, http.MethodGet, "/api/v1/skips", nil); fmt.Sprint(skips["job_ids"]) != "["+job+"]" {
		t.Fatalf("skips: %v", skips)
	}
	if status, _ := testutil.Do(t, handler, http.MethodDelete, "/api/v1/applications/"+job, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE: %d", status)
	}
	if status, body := testutil.Do(t, handler, http.MethodDelete, "/api/v1/applications/"+job, nil); status != http.StatusNotFound {
		t.Fatalf("DELETE again: %d %v", status, body)
	}
}
