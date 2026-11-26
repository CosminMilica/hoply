-- +goose Up
-- Jobs the importer reads from company job boards: shared like the catalog, keyed by the platform and its job id,
-- and closed when their board stops listing them.
ALTER TABLE jobs
    ADD COLUMN origin TEXT NOT NULL DEFAULT '',
    ADD COLUMN external_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN board TEXT NOT NULL DEFAULT '',
    ADD COLUMN posted_at TIMESTAMPTZ,
    ADD COLUMN last_seen_at TIMESTAMPTZ,
    ADD COLUMN closed_at TIMESTAMPTZ,
    DROP CONSTRAINT ck_jobs_source,
    DROP CONSTRAINT ck_jobs_catalog_owner,
    ADD CONSTRAINT ck_jobs_source CHECK (source IN ('catalog', 'link', 'text', 'import')),
    ADD CONSTRAINT ck_jobs_catalog_owner CHECK ((source IN ('catalog', 'import')) = (customer_id IS NULL)),
    ADD CONSTRAINT ck_jobs_import CHECK ((source = 'import') = (origin <> '' AND external_id <> '' AND board <> ''));

CREATE UNIQUE INDEX ux_jobs_origin_external ON jobs (origin, external_id) WHERE source = 'import';
CREATE INDEX ix_jobs_board_open ON jobs (origin, board) WHERE source = 'import' AND closed_at IS NULL;

-- +goose Down
DROP INDEX ix_jobs_board_open;
DROP INDEX ux_jobs_origin_external;
DELETE FROM jobs WHERE source = 'import';
ALTER TABLE jobs
    DROP CONSTRAINT ck_jobs_import,
    DROP CONSTRAINT ck_jobs_catalog_owner,
    DROP CONSTRAINT ck_jobs_source,
    ADD CONSTRAINT ck_jobs_source CHECK (source IN ('catalog', 'link', 'text')),
    ADD CONSTRAINT ck_jobs_catalog_owner CHECK ((source = 'catalog') = (customer_id IS NULL)),
    DROP COLUMN closed_at,
    DROP COLUMN last_seen_at,
    DROP COLUMN posted_at,
    DROP COLUMN board,
    DROP COLUMN external_id,
    DROP COLUMN origin;
