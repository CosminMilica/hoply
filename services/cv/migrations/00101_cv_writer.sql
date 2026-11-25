-- +goose Up
-- Who wrote each CV: Claude (and which model) or the rules engine, how many of Claude's sentences the checks
-- replaced, and why the rules wrote it when Claude did not.
ALTER TABLE cvs
    ADD COLUMN written_by TEXT,
    ADD COLUMN model TEXT NOT NULL DEFAULT '',
    ADD COLUMN refused INT NOT NULL DEFAULT 0,
    ADD COLUMN rules_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN limit_until TIMESTAMPTZ,
    ADD CONSTRAINT ck_cvs_written_by CHECK (written_by IN ('claude', 'rules'));

-- +goose Down
ALTER TABLE cvs
    DROP CONSTRAINT ck_cvs_written_by,
    DROP COLUMN limit_until,
    DROP COLUMN rules_reason,
    DROP COLUMN refused,
    DROP COLUMN model,
    DROP COLUMN written_by;
