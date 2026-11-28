-- +goose Up
-- A job's domains are the fields of work its title names (extract.Domains); Find jobs shows a person the jobs in the
-- domains they picked. The rows here get their domains from their areas, and the areas that became domains go; the
-- next import reads every job again (read_at) and classifies it with the new areas. Row-level security is lifted for
-- the update so the people's own jobs change too.
ALTER TABLE jobs ADD COLUMN domains TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE jobs NO FORCE ROW LEVEL SECURITY;
UPDATE jobs SET domains = array_remove(ARRAY[
        CASE WHEN areas && ARRAY['DevOps', 'Backend', 'Frontend', 'Full stack', 'Mobile', 'Data', 'AI', 'QA', 'Security',
            'Embedded', 'Sysadmin', 'Product', 'Design'] THEN 'it' END,
        CASE WHEN 'Banking' = ANY(areas) THEN 'banking' END,
        CASE WHEN 'Accounting' = ANY(areas) THEN 'accounting' END,
        CASE WHEN areas && ARRAY['Finance & control', 'Risk & compliance'] THEN 'finance' END
    ], NULL),
    areas = array_remove(array_remove(array_remove(areas, 'Banking'), 'Accounting'), 'Finance & control');
UPDATE jobs SET read_at = NULL WHERE source = 'import';
ALTER TABLE jobs FORCE ROW LEVEL SECURITY;
CREATE INDEX jobs_domains ON jobs USING GIN (domains);

-- +goose Down
DROP INDEX jobs_domains;
ALTER TABLE jobs DROP COLUMN domains;
