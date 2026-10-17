-- +goose Up
-- extensions are created by the init script as superuser, these are no-ops there
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;

-- index builds run with search_path = pg_catalog since PostgreSQL 17, so bodies name their schema
-- +goose StatementBegin
CREATE FUNCTION immutable_unaccent(input text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
AS $$
    SELECT public.unaccent('public.unaccent'::regdictionary, input)
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION search_key(input text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
AS $$
    SELECT lower(public.immutable_unaccent(input))
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION search_key_with_aliases(name text, aliases text[]) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
AS $$
    SELECT public.search_key(name || ' ' || coalesce(array_to_string(aliases, ' '), ''))
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION search_key_with_aliases(text, text[]);
DROP FUNCTION search_key(text);
DROP FUNCTION immutable_unaccent(text);
