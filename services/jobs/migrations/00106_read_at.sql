-- +goose Up
-- When the importer last read an imported job's description, so a run can skip the descriptions it read lately.
ALTER TABLE jobs ADD COLUMN read_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE jobs DROP COLUMN read_at;
