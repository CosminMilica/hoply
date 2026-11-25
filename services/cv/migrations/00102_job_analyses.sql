-- +goose Up
-- What each job ad says its company needs, read once by Claude and kept for every CV written for that job.
CREATE TABLE job_analyses (
    job_id UUID PRIMARY KEY,
    analysis JSONB NOT NULL,
    model TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE job_analyses;
