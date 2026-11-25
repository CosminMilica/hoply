-- +goose Up
-- The cover letter and the messages to the recruiter Claude wrote for one CV, on request.
CREATE TABLE letters (
    cv_id UUID PRIMARY KEY REFERENCES cvs (id) ON DELETE CASCADE,
    customer_id UUID NOT NULL,
    subject TEXT NOT NULL,
    status TEXT NOT NULL,
    recruiter TEXT NOT NULL DEFAULT '',
    written JSONB,
    model TEXT NOT NULL DEFAULT '',
    refused INT NOT NULL DEFAULT 0,
    reason TEXT NOT NULL DEFAULT '',
    limit_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_letters_status CHECK (status IN ('writing', 'ready', 'failed'))
);

-- +goose Down
DROP TABLE letters;
