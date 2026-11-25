-- +goose Up
-- Notes: what the person tells Prepare your CV about a role, a project or themselves, in their own words. Claude
-- writes CV bullets from them; they are never printed as they are.
ALTER TABLE profile_experience ADD COLUMN notes JSONB NOT NULL DEFAULT '[]';
ALTER TABLE profile_projects ADD COLUMN notes JSONB NOT NULL DEFAULT '[]';
ALTER TABLE profiles ADD COLUMN notes JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE profiles DROP COLUMN notes;
ALTER TABLE profile_projects DROP COLUMN notes;
ALTER TABLE profile_experience DROP COLUMN notes;
