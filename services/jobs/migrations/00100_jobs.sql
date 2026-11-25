-- +goose Up
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID,
    subject TEXT,
    title TEXT NOT NULL,
    company TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL DEFAULT '',
    country TEXT NOT NULL DEFAULT '',
    work_mode TEXT NOT NULL DEFAULT '',
    level TEXT NOT NULL DEFAULT '',
    years_required INT,
    must_have TEXT[] NOT NULL DEFAULT '{}',
    nice_to_have TEXT[] NOT NULL DEFAULT '{}',
    responsibilities TEXT[] NOT NULL DEFAULT '{}',
    about TEXT NOT NULL DEFAULT '',
    about_company TEXT NOT NULL DEFAULT '',
    product TEXT NOT NULL DEFAULT '',
    perks TEXT[] NOT NULL DEFAULT '{}',
    areas TEXT[] NOT NULL DEFAULT '{}',
    url TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL,
    read_from TEXT NOT NULL DEFAULT '',
    sample BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_jobs_owner CHECK ((customer_id IS NULL) = (subject IS NULL)),
    CONSTRAINT ck_jobs_source CHECK (source IN ('catalog', 'link', 'text')),
    CONSTRAINT ck_jobs_catalog_owner CHECK ((source = 'catalog') = (customer_id IS NULL))
);

CREATE INDEX ix_jobs_owner_created ON jobs (customer_id, subject, created_at DESC);
CREATE INDEX ix_jobs_catalog_created ON jobs (created_at DESC) WHERE customer_id IS NULL;

-- +goose Down
DROP TABLE jobs;
