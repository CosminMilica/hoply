#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT 'CREATE DATABASE jobs'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'jobs')\gexec
    SELECT 'CREATE DATABASE profile'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'profile')\gexec
EOSQL
