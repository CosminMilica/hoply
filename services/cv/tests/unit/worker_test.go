package unit_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"cv/internal/apperrors"
	"cv/internal/data/repositories"
	"cv/internal/models"
	"cv/internal/upstream"
	"cv/internal/worker"
)

// memoryQueue records what the worker does with its tasks.
type memoryQueue struct {
	mu          sync.Mutex
	tasks       []*repositories.Task
	steps       []string
	preview     *models.Preview
	written     *models.Written
	recovered   int
	claimErr    error
	checkingErr error
	failErr     error
	drained     chan struct{}
}

func (q *memoryQueue) Claim(context.Context) (*repositories.Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.claimErr != nil {
		return nil, q.claimErr
	}
	if len(q.tasks) == 0 {
		if q.drained != nil {
			select {
			case q.drained <- struct{}{}:
			default:
			}
		}
		return nil, nil
	}
	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	q.steps = append(q.steps, "writing")
	return task, nil
}

func (q *memoryQueue) Checking(_ context.Context, _ uuid.UUID, preview *models.Preview) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.steps = append(q.steps, "checking")
	q.preview = preview
	return q.checkingErr
}

func (q *memoryQueue) Complete(_ context.Context, _ uuid.UUID, written models.Written) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.steps = append(q.steps, "ready")
	q.written = &written
	return nil
}

func (q *memoryQueue) Fail(context.Context, uuid.UUID) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.steps = append(q.steps, "failed")
	return q.failErr
}

func (q *memoryQueue) Recover(context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.recovered++
	return q.claimErr
}

func task(t *testing.T) *repositories.Task {
	return &repositories.Task{ID: uuid.New(), Request: request(), Profile: sampleProfile(t), Job: catalogJob(t, 0),
		CreatedAt: time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)}
}

func TestWorkerWritesAQueuedCVAsOfWhenItWasAskedFor(t *testing.T) {
	queue := &memoryQueue{tasks: []*repositories.Task{task(t)}}
	w := worker.New(queue, time.Hour)
	if !w.Next(context.Background()) {
		t.Fatal("a queued CV is written")
	}
	equal(t, queue.steps, []string{"writing", "checking", "ready"})
	equal(t, queue.preview.From, "Built and maintained deployment pipelines for internal services.")
	equal(t, queue.written.Tailored.Summary[:42], "DevOps Engineer with 5 years of experience")
	if w.Next(context.Background()) {
		t.Fatal("an empty queue has nothing to write")
	}
}

func TestWorkerFailsACVItCannotWrite(t *testing.T) {
	queue := &memoryQueue{tasks: []*repositories.Task{task(t)}, checkingErr: errors.New("gone"), failErr: errors.New("still gone")}
	if !worker.New(queue, time.Hour).Next(context.Background()) {
		t.Fatal("the CV was taken")
	}
	equal(t, queue.steps, []string{"writing", "checking", "failed"})
}

func TestWorkerStopsOnAClaimError(t *testing.T) {
	queue := &memoryQueue{claimErr: apperrors.New(apperrors.DatabaseUnavailable, "down")}
	if worker.New(queue, time.Hour).Next(context.Background()) {
		t.Fatal("nothing is written when the queue is down")
	}
}

func TestWorkerRunRecoversThenWritesWhatItIsWokenFor(t *testing.T) {
	queue := &memoryQueue{drained: make(chan struct{}, 1)}
	w := worker.New(queue, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()
	<-queue.drained
	queue.mu.Lock()
	queue.tasks = append(queue.tasks, task(t))
	queue.mu.Unlock()
	w.Wake()
	w.Wake()
	<-queue.drained
	cancel()
	<-done
	queue.mu.Lock()
	defer queue.mu.Unlock()
	equal(t, queue.recovered, 1)
	equal(t, queue.steps, []string{"writing", "checking", "ready"})
}

func TestWorkerRunPollsAndSurvivesARecoveryError(t *testing.T) {
	queue := &memoryQueue{claimErr: errors.New("down")}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	worker.New(queue, 5*time.Millisecond).Run(ctx)
	equal(t, queue.recovered, 1)
}

func TestUpstreamReadsTheProfileAndTheJob(t *testing.T) {
	job := uuid.MustParse(jobID)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/profiles/me":
			_, _ = w.Write([]byte(`{"name": "Ana", "skills": ["Go"], "work_modes": ["remote"]}`))
		case "/api/v1/jobs/" + jobID:
			_, _ = w.Write([]byte(`{"id": "` + jobID + `", "title": "Engineer", "company": "Acme", "perks": []}`))
		case "/api/v1/jobs/" + uuid.Nil.String():
			_, _ = w.Write([]byte(`{`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := upstream.New(server.URL+"/", server.URL)
	profile, raw, err := client.Profile(context.Background())
	if err != nil || profile.Name != "Ana" || string(raw) == "" {
		t.Fatalf("profile: %+v %s %v", profile, raw, err)
	}
	found, raw, err := client.Job(context.Background(), job)
	if err != nil || found.Title != "Engineer" || string(raw) == "" {
		t.Fatalf("job: %+v %v", found, err)
	}
	var appErr *apperrors.Error
	if _, _, err := client.Job(context.Background(), uuid.New()); !errors.As(err, &appErr) || appErr.Code != apperrors.NotFound {
		t.Fatalf("unknown job: %v", err)
	}
	if _, _, err := client.Job(context.Background(), uuid.Nil); !errors.As(err, &appErr) || appErr.Code != apperrors.DependencyUnavailable {
		t.Fatalf("unreadable job: %v", err)
	}
	broken := upstream.New(server.URL+"/nowhere", server.URL+"/nowhere")
	if _, _, err := broken.Profile(context.Background()); !errors.As(err, &appErr) || appErr.Code != apperrors.DependencyUnavailable {
		t.Fatalf("missing profile: %v", err)
	}
	if _, _, err := broken.Job(context.Background(), job); !errors.As(err, &appErr) || appErr.Code != apperrors.NotFound {
		t.Fatalf("missing jobs route: %v", err)
	}
}

func TestUpstreamReportsUnreachableAndUnreadableServices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/profiles/me" {
			_, _ = w.Write([]byte(`[`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	var appErr *apperrors.Error
	client := upstream.New(server.URL, server.URL)
	if _, _, err := client.Profile(context.Background()); !errors.As(err, &appErr) || appErr.Code != apperrors.DependencyUnavailable {
		t.Fatalf("unreadable profile: %v", err)
	}
	if _, _, err := client.Job(context.Background(), uuid.New()); !errors.As(err, &appErr) || appErr.Context["dependency"] != "jobs" {
		t.Fatalf("failing jobs: %v", err)
	}
	server.Close()
	if _, _, err := client.Profile(context.Background()); !errors.As(err, &appErr) || appErr.Context["dependency"] != "profile" {
		t.Fatalf("unreachable profile: %v", err)
	}
	if _, _, err := upstream.New("http://bad host", "").Profile(context.Background()); err == nil {
		t.Fatal("a bad URL fails")
	}
}

func TestUpstreamFromEnv(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"name": "From env"}`))
	}))
	defer server.Close()
	t.Setenv("PROFILE_SERVICE_URL", server.URL)
	t.Setenv("JOBS_SERVICE_URL", " ")
	profile, _, err := upstream.FromEnv().Profile(context.Background())
	if err != nil || profile.Name != "From env" {
		t.Fatalf("env profile: %+v %v", profile, err)
	}
}
