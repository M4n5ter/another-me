#!/bin/bash
# Exit immediately if a command exits with a non-zero status.
set -e

# Execute SQL commands to create extensions in the target database
# The database name is typically taken from POSTGRES_DB environment variable
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Enable Apache AGE Extension
    CREATE EXTENSION IF NOT EXISTS age;
    -- Load AGE libraries required by the session
    LOAD 'age';
    -- Set the search path to include AGE's catalog
    SET search_path = ag_catalog, "\$user", public;
    -- Verify AGE setup (optional)
    SELECT * FROM ag_catalog.ag_label;

    -- Enable pgvector Extension
    CREATE EXTENSION IF NOT EXISTS vector;
    -- Verify pgvector setup (optional)
    SELECT VEC_TO_STRING(ARRAY[1,2,3]::vector);

    -- Enable zhparser Extension
    CREATE EXTENSION IF NOT EXISTS zhparser;
    -- Verify zhparser setup (optional)
    SELECT ts_lexize('zhparser', '我们都是中国人');

EOSQL

echo "**** Extensions AGE, pgvector, zhparser created successfully ****"