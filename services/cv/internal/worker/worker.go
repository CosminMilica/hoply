// Package worker writes the queued CVs, one at a time, in the service's process.
package worker

import (
	"context"
	"log/slog"
	"time"

	"cv/internal/cvdoc"
	"cv/internal/data/repositories"
	"cv/internal/identity"
	"cv/internal/models"
)

// Writer writes one CV; it fails only when ctx ends.
type Writer interface {
	Write(ctx context.Context, task *repositories.Task) (models.Written, models.Authorship, error)
}

// Worker takes queued CVs and writes them, as of the moment each was asked for.
type Worker struct {
	queue  repositories.Queue
	writer Writer
	wake   chan struct{}
	poll   time.Duration
}

// New polls the queue every poll interval, and at once when Wake is called.
func New(queue repositories.Queue, writer Writer, poll time.Duration) *Worker {
	return &Worker{queue: queue, writer: writer, wake: make(chan struct{}, 1), poll: poll}
}

// Wake makes a waiting worker look at the queue now; it never blocks.
func (w *Worker) Wake() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

// Run recovers what a stopped worker left, then writes every queued CV until
// ctx ends.
func (w *Worker) Run(ctx context.Context) {
	if err := w.queue.Recover(ctx); err != nil {
		slog.Error("cv queue recovery failed", "error", err)
	}
	ticker := time.NewTicker(w.poll)
	defer ticker.Stop()
	for {
		for w.Next(ctx) {
		}
		select {
		case <-ctx.Done():
			return
		case <-w.wake:
		case <-ticker.C:
		}
	}
}

// Next writes the oldest queued CV and reports whether there was one. A CV
// interrupted by ctx ending stays where it is, for Recover to queue again.
func (w *Worker) Next(ctx context.Context) bool {
	task, err := w.queue.Claim(ctx)
	if err != nil {
		if ctx.Err() == nil {
			slog.Error("cv queue claim failed", "error", err)
		}
		return false
	}
	if task == nil {
		return false
	}
	// The CV is written for its tenant: the services it calls see that person.
	ctx = identity.WithContext(ctx, identity.New(task.Tenant, "svc-cv"))
	if err := w.write(ctx, task); err != nil && ctx.Err() == nil {
		slog.Error("cv writing failed", "cv_id", task.ID, "error", err)
		if err := w.queue.Fail(ctx, task.ID); err != nil {
			slog.Error("cv could not be marked failed", "cv_id", task.ID, "error", err)
		}
	}
	return ctx.Err() == nil
}

func (w *Worker) write(ctx context.Context, task *repositories.Task) error {
	written, by, err := w.writer.Write(ctx, task)
	if err != nil {
		return err
	}
	preview := cvdoc.RewritePreview(task.Profile, cvdoc.ReadJob(task.Job), task.Request)
	if err := w.queue.Checking(ctx, task.ID, preview); err != nil {
		return err
	}
	return w.queue.Complete(ctx, task.ID, written, by)
}
