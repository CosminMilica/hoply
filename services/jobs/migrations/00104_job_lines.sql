-- +goose Up
-- The ad's own lines for what it asks (qualifications) and what is a plus (bonuses), as the rules read them.
ALTER TABLE jobs
    ADD COLUMN qualifications TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN bonuses TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE jobs DROP COLUMN bonuses, DROP COLUMN qualifications;
