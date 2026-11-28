-- +goose Up
-- The language a person uses the app in (en or ro; empty until they pick one) and the fields of work whose jobs Find
-- jobs shows them (the jobs service's domains); a person with no domains has not been through onboarding yet.
ALTER TABLE profiles
    ADD COLUMN app_language TEXT NOT NULL DEFAULT '',
    ADD COLUMN domains TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE profiles DROP COLUMN app_language, DROP COLUMN domains;
