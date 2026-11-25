// Package worker writes the queued CVs, one at a time, in the service's process.
package worker

import (
	"context"
	"log/slog"
	"time"

	"cv/internal/cvdoc"
	"cv/internal/data/repositories"
)

// Worker takes queued CVs and writes them with the rules engine, as of the
// moment each was asked for.
type Worker struct {
	queue repositories.Queue
	wake  chan struct{}
	poll  time.Duration
}

// New polls the queue every poll interval, and at once when Wake is called.
func New(queue repositories.Queue, poll time.Duration) *Worker {
	return &Worker{queue: queue, wake: make(chan struct{}, 1), poll: poll}
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

// Next writes the oldest queued CV and reports whether there was one.
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
	if err := w.write(ctx, task); err != nil {
		slog.Error("cv writing failed", "cv_id", task.ID, "error", err)
		if err := w.queue.Fail(ctx, task.ID); err != nil {
			slog.Error("cv could not be marked failed", "cv_id", task.ID, "error", err)
		}
	}
	return true
}

func (w *Worker) write(ctx context.Context, task *repositories.Task) error {
	job := cvdoc.ReadJob(task.Job)
	if err := w.queue.Checking(ctx, task.ID, cvdoc.RewritePreview(task.Profile, job, task.Request)); err != nil {
		return err
	}
	return w.queue.Complete(ctx, task.ID, cvdoc.Write(task.Profile, job, task.Request, task.CreatedAt))
}
