-- +goose Up
ALTER TABLE profiles
    ADD COLUMN city TEXT NOT NULL DEFAULT '',
    ADD COLUMN country TEXT NOT NULL DEFAULT '',
    ADD COLUMN work_modes TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN open_to_relocation BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN work_authorization TEXT NOT NULL DEFAULT '',
    ADD COLUMN work_authorization_region TEXT NOT NULL DEFAULT '',
    ADD COLUMN notice_period_weeks INT,
    ADD COLUMN available_from VARCHAR(7) NOT NULL DEFAULT '',
    ADD COLUMN target_roles TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN target_seniority TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN employment_types TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN salary_min INT,
    ADD COLUMN salary_currency VARCHAR(3) NOT NULL DEFAULT '',
    ADD COLUMN salary_period TEXT NOT NULL DEFAULT '',
    ADD COLUMN additional_info TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE profile_experience
    ADD COLUMN employment_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN industry TEXT NOT NULL DEFAULT '',
    ADD COLUMN team_size INT,
    ADD COLUMN people_managed INT,
    ADD COLUMN start_month VARCHAR(7) NOT NULL DEFAULT '',
    ADD COLUMN end_month VARCHAR(7) NOT NULL DEFAULT '',
    ADD COLUMN bullet_items JSONB NOT NULL DEFAULT '[]';

UPDATE profile_experience SET
    start_month = CASE WHEN start_year <> '' THEN start_year || '-01' ELSE '' END,
    end_month = CASE WHEN end_year <> '' THEN end_year || '-01' ELSE '' END,
    bullet_items = COALESCE((
        SELECT jsonb_agg(jsonb_build_object('text', b.text, 'result', '', 'skills', '[]'::jsonb) ORDER BY b.ord)
        FROM unnest(bullets) WITH ORDINALITY AS b(text, ord)
    ), '[]'::jsonb);

ALTER TABLE profile_experience DROP COLUMN start_year, DROP COLUMN end_year, DROP COLUMN bullets;
ALTER TABLE profile_experience RENAME COLUMN bullet_items TO bullets;

ALTER TABLE profile_education
    ADD COLUMN level TEXT NOT NULL DEFAULT 'other',
    ADD COLUMN field TEXT NOT NULL DEFAULT '',
    ADD COLUMN start_year VARCHAR(4) NOT NULL DEFAULT '',
    ADD COLUMN end_year VARCHAR(4) NOT NULL DEFAULT '';

UPDATE profile_education SET
    level = CASE
        WHEN degree ILIKE '%bsc%' OR degree ILIKE '%bachelor%' OR degree ILIKE '%licen%' THEN 'bachelor'
        WHEN degree ILIKE '%msc%' OR degree ILIKE '%master%' THEN 'master'
        WHEN degree ILIKE '%phd%' OR degree ILIKE '%doctor%' THEN 'doctorate'
        ELSE 'other'
    END,
    start_year = COALESCE(substring(years FROM '([0-9]{4})'), ''),
    end_year = COALESCE(substring(years FROM '.*([0-9]{4})'), '');

ALTER TABLE profile_education DROP COLUMN years;
ALTER TABLE profile_education ALTER COLUMN degree SET DEFAULT '';

CREATE TABLE profile_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    position BIGINT GENERATED ALWAYS AS IDENTITY,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    start_month VARCHAR(7) NOT NULL DEFAULT '',
    end_month VARCHAR(7) NOT NULL DEFAULT '',
    tools TEXT[] NOT NULL DEFAULT '{}',
    bullets JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_profile_projects_profile ON profile_projects (profile_id, position);

CREATE TABLE profile_certifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    position BIGINT GENERATED ALWAYS AS IDENTITY,
    name TEXT NOT NULL,
    issuer TEXT NOT NULL DEFAULT '',
    issued VARCHAR(7) NOT NULL DEFAULT '',
    expires VARCHAR(7) NOT NULL DEFAULT '',
    credential_id TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_profile_certifications_profile ON profile_certifications (profile_id, position);

-- +goose Down
DROP TABLE profile_certifications;
DROP TABLE profile_projects;

ALTER TABLE profile_education ADD COLUMN years TEXT NOT NULL DEFAULT '';
UPDATE profile_education SET years = CASE
    WHEN start_year <> '' AND end_year <> '' AND start_year <> end_year THEN start_year || ' – ' || end_year
    WHEN start_year <> '' THEN start_year
    ELSE end_year
END;
UPDATE profile_education SET degree = CASE
    WHEN field <> '' THEN initcap(replace(level, '_', ' ')) || ', ' || field
    ELSE initcap(replace(level, '_', ' '))
END
WHERE degree = '';
ALTER TABLE profile_education ALTER COLUMN degree DROP DEFAULT;
ALTER TABLE profile_education DROP COLUMN level, DROP COLUMN field, DROP COLUMN start_year, DROP COLUMN end_year;

ALTER TABLE profile_experience
    ADD COLUMN start_year VARCHAR(4) NOT NULL DEFAULT '',
    ADD COLUMN end_year VARCHAR(4) NOT NULL DEFAULT '',
    ADD COLUMN bullet_texts TEXT[] NOT NULL DEFAULT '{}';
UPDATE profile_experience SET
    start_year = left(start_month, 4),
    end_year = left(end_month, 4),
    bullet_texts = COALESCE(ARRAY(
        SELECT b.item ->> 'text' FROM jsonb_array_elements(bullets) WITH ORDINALITY AS b(item, ord) ORDER BY b.ord
    ), '{}');
ALTER TABLE profile_experience
    DROP COLUMN bullets, DROP COLUMN start_month, DROP COLUMN end_month,
    DROP COLUMN employment_type, DROP COLUMN industry, DROP COLUMN team_size, DROP COLUMN people_managed;
ALTER TABLE profile_experience RENAME COLUMN bullet_texts TO bullets;

ALTER TABLE profiles
    DROP COLUMN city, DROP COLUMN country, DROP COLUMN work_modes, DROP COLUMN open_to_relocation,
    DROP COLUMN work_authorization, DROP COLUMN work_authorization_region, DROP COLUMN notice_period_weeks,
    DROP COLUMN available_from, DROP COLUMN target_roles, DROP COLUMN target_seniority,
    DROP COLUMN employment_types, DROP COLUMN salary_min, DROP COLUMN salary_currency,
    DROP COLUMN salary_period, DROP COLUMN additional_info;
