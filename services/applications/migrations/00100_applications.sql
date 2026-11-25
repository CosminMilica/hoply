-- +goose Up
CREATE TABLE applications (
    customer_id UUID NOT NULL,
    subject TEXT NOT NULL,
    job_id UUID NOT NULL,
    status TEXT NOT NULL,
    saved_at TIMESTAMPTZ,
    cv_ready_at TIMESTAMPTZ,
    applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (customer_id, subject, job_id),
    CONSTRAINT ck_applications_status CHECK (status IN ('saved', 'cv_ready', 'applied'))
);

CREATE INDEX ix_applications_owner_updated ON applications (customer_id, subject, updated_at DESC);

CREATE TABLE skips (
    customer_id UUID NOT NULL,
    subject TEXT NOT NULL,
    job_id UUID NOT NULL,
    skipped_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (customer_id, subject, job_id)
);

CREATE INDEX ix_skips_owner_skipped ON skips (customer_id, subject, skipped_at DESC);

-- +goose Down
DROP TABLE skips;
DROP TABLE applications;
