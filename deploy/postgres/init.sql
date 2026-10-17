-- runs once on an empty data directory, as the superuser, with psql variables owner_password, app_password and readonly_password
SELECT format('CREATE ROLE simpan_owner LOGIN PASSWORD %L', :'owner_password') \gexec
SELECT format('CREATE ROLE simpan_app LOGIN PASSWORD %L', :'app_password') \gexec
SELECT format('CREATE ROLE simpan_readonly LOGIN PASSWORD %L', :'readonly_password') \gexec
CREATE DATABASE simpan OWNER simpan_owner;

\connect simpan

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS vector;

REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO simpan_app, simpan_readonly;

ALTER DEFAULT PRIVILEGES FOR ROLE simpan_owner IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO simpan_app;
ALTER DEFAULT PRIVILEGES FOR ROLE simpan_owner IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO simpan_app;
ALTER DEFAULT PRIVILEGES FOR ROLE simpan_owner IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO simpan_app, simpan_readonly;
