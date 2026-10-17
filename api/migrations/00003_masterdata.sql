-- +goose Up
CREATE TABLE zones (
    id           uuid PRIMARY KEY,
    slug         text NOT NULL UNIQUE,
    name         text NOT NULL,
    centroid_lat double precision NOT NULL,
    centroid_lng double precision NOT NULL,
    active       boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE locations (
    id         uuid PRIMARY KEY,
    zone_id    uuid NOT NULL REFERENCES zones (id),
    slug       text NOT NULL UNIQUE,
    name       text NOT NULL,
    aliases    text[] NOT NULL DEFAULT '{}',
    kind       text NOT NULL CHECK (kind IN ('building', 'outdoor', 'facility', 'transport', 'parking')),
    lat        double precision NOT NULL,
    lng        double precision NOT NULL,
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX locations_search_trgm ON locations USING gin (search_key_with_aliases(name, aliases) gin_trgm_ops);
CREATE INDEX locations_zone ON locations (zone_id);

CREATE TABLE posts (
    id            uuid PRIMARY KEY,
    slug          text NOT NULL UNIQUE,
    name          text NOT NULL,
    location_id   uuid NOT NULL REFERENCES locations (id),
    phone         text,
    opening_hours jsonb NOT NULL DEFAULT '{}',
    active        boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE officer_posts (
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    post_id     uuid NOT NULL REFERENCES posts (id),
    assigned_at timestamptz NOT NULL DEFAULT now(),
    assigned_by uuid REFERENCES users (id),
    PRIMARY KEY (user_id, post_id)
);

CREATE TABLE categories (
    id                     uuid PRIMARY KEY,
    parent_id              uuid REFERENCES categories (id),
    slug                   text NOT NULL,
    name                   text NOT NULL,
    sensitivity            text CHECK (sensitivity IN ('normal', 'valuable', 'identity')),
    retention_days         integer CHECK (retention_days > 0),
    verification_questions jsonb NOT NULL DEFAULT '[]',
    clip_prompts           text[] NOT NULL DEFAULT '{}',
    sort_order             integer NOT NULL DEFAULT 0,
    active                 boolean NOT NULL DEFAULT true,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    CHECK (parent_id IS NULL OR sensitivity IS NOT NULL)
);

CREATE INDEX categories_parent ON categories (parent_id);
-- a parent and its only leaf may share a slug, as in Payung and payung, so uniqueness is per level
CREATE UNIQUE INDEX categories_parent_slug ON categories (slug) WHERE parent_id IS NULL;
CREATE UNIQUE INDEX categories_leaf_slug ON categories (slug) WHERE parent_id IS NOT NULL;

CREATE TABLE category_compatibility (
    category_a uuid NOT NULL REFERENCES categories (id),
    category_b uuid NOT NULL REFERENCES categories (id),
    factor     numeric(3, 2) NOT NULL CHECK (factor BETWEEN 0 AND 1),
    PRIMARY KEY (category_a, category_b),
    CHECK (category_a < category_b)
);

CREATE TABLE colours (
    name       text PRIMARY KEY,
    hex        text NOT NULL CHECK (hex ~ '^#[0-9A-F]{6}$'),
    sort_order integer NOT NULL DEFAULT 0
);

CREATE TABLE brands (
    name    text PRIMARY KEY,
    aliases text[] NOT NULL DEFAULT '{}'
);

CREATE TABLE synonyms (
    term      text PRIMARY KEY,
    expansion text NOT NULL
);

CREATE TABLE system_settings (
    key        text PRIMARY KEY,
    value      jsonb NOT NULL,
    updated_by uuid REFERENCES users (id),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE system_settings;
DROP TABLE synonyms;
DROP TABLE brands;
DROP TABLE colours;
DROP TABLE category_compatibility;
DROP TABLE categories;
DROP TABLE officer_posts;
DROP TABLE posts;
DROP TABLE locations;
DROP TABLE zones;
