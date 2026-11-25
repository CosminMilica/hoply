-- +goose Up
-- A CV, its snapshots of the profile and the job, and its place in the writing queue: a CV is queued until the
-- worker takes it (FOR UPDATE SKIP LOCKED), then writing, checking, and ready or failed.
CREATE TABLE cvs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    subject TEXT NOT NULL,
    job_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    request JSONB NOT NULL,
    profile_snapshot JSONB NOT NULL,
    job_snapshot JSONB NOT NULL,
    job_title TEXT NOT NULL,
    company TEXT NOT NULL,
    file_name TEXT NOT NULL,
    preview JSONB,
    written JSONB,
    attempts INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_cvs_status CHECK (status IN ('queued', 'writing', 'checking', 'ready', 'failed'))
);

CREATE INDEX ix_cvs_owner_created ON cvs (customer_id, subject, created_at DESC);
CREATE INDEX ix_cvs_queued ON cvs (created_at) WHERE status = 'queued';

-- +goose Down
DROP TABLE cvs;
