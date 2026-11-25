-- +goose Up
-- Where a job's requirements come from: the rules that read the ad, or Claude's analysis of it (written by the cv
-- service), which tells skills from practices and must-haves from nice-to-haves.
ALTER TABLE jobs
    ADD COLUMN requirements_from TEXT NOT NULL DEFAULT 'rules',
    ADD CONSTRAINT ck_jobs_requirements_from CHECK (requirements_from IN ('rules', 'analysis'));

-- +goose Down
ALTER TABLE jobs DROP CONSTRAINT ck_jobs_requirements_from, DROP COLUMN requirements_from;
