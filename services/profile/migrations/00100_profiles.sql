-- +goose Up
CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    subject TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    headline TEXT NOT NULL DEFAULT '',
    about TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    github TEXT NOT NULL DEFAULT '',
    linkedin TEXT NOT NULL DEFAULT '',
    skills TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_profiles_customer_subject UNIQUE (customer_id, subject)
);

CREATE TABLE profile_experience (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    position BIGINT GENERATED ALWAYS AS IDENTITY,
    role TEXT NOT NULL,
    company TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    start_year VARCHAR(4) NOT NULL DEFAULT '',
    end_year VARCHAR(4) NOT NULL DEFAULT '',
    tools TEXT[] NOT NULL DEFAULT '{}',
    bullets TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_profile_experience_profile ON profile_experience (profile_id, position);

CREATE TABLE profile_education (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    position BIGINT GENERATED ALWAYS AS IDENTITY,
    degree TEXT NOT NULL,
    school TEXT NOT NULL DEFAULT '',
    years TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_profile_education_profile ON profile_education (profile_id, position);

CREATE TABLE profile_languages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    position BIGINT GENERATED ALWAYS AS IDENTITY,
    name TEXT NOT NULL,
    level VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_profile_languages_profile ON profile_languages (profile_id, position);

-- +goose Down
DROP TABLE profile_languages;
DROP TABLE profile_education;
DROP TABLE profile_experience;
DROP TABLE profiles;
