-- +goose Up
-- A person is a tenant: one profile per tenant, and subject records who created it. Row-level security keeps each
-- tenant's profile from every other with the policy of 00099; a child row is visible when its profile is.
ALTER TABLE profiles DROP CONSTRAINT uq_profiles_customer_subject, ADD CONSTRAINT uq_profiles_customer UNIQUE (customer_id);

ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE profiles FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_profiles ON profiles
    USING (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (customer_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
ALTER TABLE profile_experience ENABLE ROW LEVEL SECURITY;
ALTER TABLE profile_experience FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_profile_experience ON profile_experience
    USING (profile_id IN (SELECT id FROM profiles))
    WITH CHECK (profile_id IN (SELECT id FROM profiles));
ALTER TABLE profile_education ENABLE ROW LEVEL SECURITY;
ALTER TABLE profile_education FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_profile_education ON profile_education
    USING (profile_id IN (SELECT id FROM profiles))
    WITH CHECK (profile_id IN (SELECT id FROM profiles));
ALTER TABLE profile_languages ENABLE ROW LEVEL SECURITY;
ALTER TABLE profile_languages FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_profile_languages ON profile_languages
    USING (profile_id IN (SELECT id FROM profiles))
    WITH CHECK (profile_id IN (SELECT id FROM profiles));
ALTER TABLE profile_projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE profile_projects FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_profile_projects ON profile_projects
    USING (profile_id IN (SELECT id FROM profiles))
    WITH CHECK (profile_id IN (SELECT id FROM profiles));
ALTER TABLE profile_certifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE profile_certifications FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_profile_certifications ON profile_certifications
    USING (profile_id IN (SELECT id FROM profiles))
    WITH CHECK (profile_id IN (SELECT id FROM profiles));

-- +goose Down
DROP POLICY tenant_isolation_profile_certifications ON profile_certifications;
ALTER TABLE profile_certifications NO FORCE ROW LEVEL SECURITY;
ALTER TABLE profile_certifications DISABLE ROW LEVEL SECURITY;
DROP POLICY tenant_isolation_profile_projects ON profile_projects;
ALTER TABLE profile_projects NO FORCE ROW LEVEL SECURITY;
ALTER TABLE profile_projects DISABLE ROW LEVEL SECURITY;
DROP POLICY tenant_isolation_profile_languages ON profile_languages;
ALTER TABLE profile_languages NO FORCE ROW LEVEL SECURITY;
ALTER TABLE profile_languages DISABLE ROW LEVEL SECURITY;
DROP POLICY tenant_isolation_profile_education ON profile_education;
ALTER TABLE profile_education NO FORCE ROW LEVEL SECURITY;
ALTER TABLE profile_education DISABLE ROW LEVEL SECURITY;
DROP POLICY tenant_isolation_profile_experience ON profile_experience;
ALTER TABLE profile_experience NO FORCE ROW LEVEL SECURITY;
ALTER TABLE profile_experience DISABLE ROW LEVEL SECURITY;
DROP POLICY tenant_isolation_profiles ON profiles;
ALTER TABLE profiles NO FORCE ROW LEVEL SECURITY;
ALTER TABLE profiles DISABLE ROW LEVEL SECURITY;
ALTER TABLE profiles DROP CONSTRAINT uq_profiles_customer, ADD CONSTRAINT uq_profiles_customer_subject UNIQUE (customer_id, subject);
