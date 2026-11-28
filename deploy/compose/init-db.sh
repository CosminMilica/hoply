#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT 'CREATE DATABASE applications'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'applications')\gexec
    SELECT 'CREATE DATABASE cv'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'cv')\gexec
    SELECT 'CREATE DATABASE jobs'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'jobs')\gexec
    SELECT 'CREATE DATABASE keycloak'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'keycloak')\gexec
    SELECT 'CREATE DATABASE profile'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'profile')\gexec
EOSQL
