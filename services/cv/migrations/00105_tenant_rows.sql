-- +goose Up
-- A person is a tenant: rows belong to the tenant, and subject records who asked. Row-level security keeps each
-- tenant's CVs, questions and letters from every other, with the policy of 00099. The worker writes for every tenant:
-- cv_claim and cv_recover, owned by the migration's owner, are the only queries that see the whole queue, and
-- cv_tenant names a CV's tenant, which the worker binds before it changes that CV.
ALTER TABLE probes DROP CONSTRAINT probes_pkey, ADD PRIMARY KEY (customer_id, job_id);
DROP INDEX ix_cvs_owner_created;
CREATE INDEX ix_cvs_tenant_created ON cvs (customer_id, created_at DESC);

ALTER TABLE cvs ENABLE ROW LEVEL SECURITY;
ALTER TABLE cvs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_cvs ON cvs
    USING (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

ALTER TABLE probes ENABLE ROW LEVEL SECURITY;
ALTER TABLE probes FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_probes ON probes
    USING (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

ALTER TABLE letters ENABLE ROW LEVEL SECURITY;
ALTER TABLE letters FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_letters ON letters
    USING (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- +goose StatementBegin
CREATE FUNCTION cv_claim()
RETURNS TABLE (id UUID, customer_id UUID, request JSONB, profile_snapshot JSONB, job_snapshot JSONB, created_at TIMESTAMPTZ)
LANGUAGE sql SECURITY DEFINER SET search_path FROM CURRENT AS $$
    UPDATE cvs SET status = 'writing', attempts = cvs.attempts + 1, updated_at = NOW()
    WHERE cvs.id = (SELECT queued.id FROM cvs queued WHERE queued.status = 'queued'
                    ORDER BY queued.created_at, queued.id LIMIT 1 FOR UPDATE SKIP LOCKED)
    RETURNING cvs.id, cvs.customer_id, cvs.request, cvs.profile_snapshot, cvs.job_snapshot, cvs.created_at
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION cv_recover(max_attempts INT) RETURNS VOID
LANGUAGE sql SECURITY DEFINER SET search_path FROM CURRENT AS $$
    UPDATE cvs SET status = CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'queued' END, updated_at = NOW()
    WHERE status IN ('writing', 'checking')
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION cv_tenant(cv UUID) RETURNS UUID
LANGUAGE sql STABLE SECURITY DEFINER SET search_path FROM CURRENT AS $$
    SELECT customer_id FROM cvs WHERE id = cv
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION cv_tenant(UUID);
DROP FUNCTION cv_recover(INT);
DROP FUNCTION cv_claim();
DROP POLICY tenant_isolation_letters ON letters;
ALTER TABLE letters NO FORCE ROW LEVEL SECURITY;
ALTER TABLE letters DISABLE ROW LEVEL SECURITY;
DROP POLICY tenant_isolation_probes ON probes;
ALTER TABLE probes NO FORCE ROW LEVEL SECURITY;
ALTER TABLE probes DISABLE ROW LEVEL SECURITY;
DROP POLICY tenant_isolation_cvs ON cvs;
ALTER TABLE cvs NO FORCE ROW LEVEL SECURITY;
ALTER TABLE cvs DISABLE ROW LEVEL SECURITY;
DROP INDEX ix_cvs_tenant_created;
CREATE INDEX ix_cvs_owner_created ON cvs (customer_id, subject, created_at DESC);
ALTER TABLE probes DROP CONSTRAINT probes_pkey, ADD PRIMARY KEY (customer_id, subject, job_id);
