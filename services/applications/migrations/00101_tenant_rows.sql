-- +goose Up
-- A person is a tenant: rows belong to the tenant, and subject records who wrote them. Row-level security keeps each
-- tenant's rows from every other, with the policy of 00099 (app.current_tenant, bound per transaction).
ALTER TABLE applications DROP CONSTRAINT applications_pkey, ADD PRIMARY KEY (customer_id, job_id);
DROP INDEX ix_applications_owner_updated;
CREATE INDEX ix_applications_tenant_updated ON applications (customer_id, updated_at DESC);
ALTER TABLE skips DROP CONSTRAINT skips_pkey, ADD PRIMARY KEY (customer_id, job_id);
DROP INDEX ix_skips_owner_skipped;
CREATE INDEX ix_skips_tenant_skipped ON skips (customer_id, skipped_at DESC);

ALTER TABLE applications ENABLE ROW LEVEL SECURITY;
ALTER TABLE applications FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_applications ON applications
    USING (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
ALTER TABLE skips ENABLE ROW LEVEL SECURITY;
ALTER TABLE skips FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_skips ON skips
    USING (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- +goose Down
DROP POLICY tenant_isolation_skips ON skips;
ALTER TABLE skips NO FORCE ROW LEVEL SECURITY;
ALTER TABLE skips DISABLE ROW LEVEL SECURITY;
DROP POLICY tenant_isolation_applications ON applications;
ALTER TABLE applications NO FORCE ROW LEVEL SECURITY;
ALTER TABLE applications DISABLE ROW LEVEL SECURITY;
DROP INDEX ix_skips_tenant_skipped;
CREATE INDEX ix_skips_owner_skipped ON skips (customer_id, subject, skipped_at DESC);
ALTER TABLE skips DROP CONSTRAINT skips_pkey, ADD PRIMARY KEY (customer_id, subject, job_id);
DROP INDEX ix_applications_tenant_updated;
CREATE INDEX ix_applications_owner_updated ON applications (customer_id, subject, updated_at DESC);
ALTER TABLE applications DROP CONSTRAINT applications_pkey, ADD PRIMARY KEY (customer_id, subject, job_id);
