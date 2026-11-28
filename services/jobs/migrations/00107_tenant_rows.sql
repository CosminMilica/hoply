-- +goose Up
-- A person is a tenant. Row-level security keeps the catalog (no owner) shared and a person's added jobs theirs alone:
-- everyone reads the catalog and their own jobs; a request bound to a tenant adds, changes and deletes only its own
-- jobs, and may also change a catalog job (an analysis of the ad is the same for everyone); without a tenant (the
-- importer) only catalog jobs are written. app.current_tenant is bound per transaction (00099).
ALTER TABLE jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs FORCE ROW LEVEL SECURITY;
CREATE POLICY jobs_read ON jobs FOR SELECT
    USING (customer_id IS NULL OR customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
CREATE POLICY jobs_add ON jobs FOR INSERT
    WITH CHECK (customer_id IS NOT DISTINCT FROM NULLIF(current_setting('app.current_tenant', true), '')::uuid);
CREATE POLICY jobs_change ON jobs FOR UPDATE
    USING (customer_id IS NULL OR customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (customer_id IS NULL OR customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
CREATE POLICY jobs_remove ON jobs FOR DELETE
    USING (customer_id IS NOT DISTINCT FROM NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- +goose Down
DROP POLICY jobs_remove ON jobs;
DROP POLICY jobs_change ON jobs;
DROP POLICY jobs_add ON jobs;
DROP POLICY jobs_read ON jobs;
ALTER TABLE jobs NO FORCE ROW LEVEL SECURITY;
ALTER TABLE jobs DISABLE ROW LEVEL SECURITY;
