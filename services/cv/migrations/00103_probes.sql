-- +goose Up
-- The questions Prepare your CV asks for one job, asked by Claude for the person's facts as they were (fingerprint):
-- asked again when the facts change.
CREATE TABLE probes (
    customer_id UUID NOT NULL,
    subject TEXT NOT NULL,
    job_id UUID NOT NULL,
    fingerprint TEXT NOT NULL,
    status TEXT NOT NULL,
    questions JSONB NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (customer_id, subject, job_id),
    CONSTRAINT ck_probes_status CHECK (status IN ('ready', 'pending', 'unavailable'))
);

-- +goose Down
DROP TABLE probes;
