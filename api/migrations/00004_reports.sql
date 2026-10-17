-- +goose Up
CREATE TABLE code_sequences (
    prefix     text NOT NULL CHECK (prefix IN ('LST', 'FND', 'CLM')),
    yymm       text NOT NULL CHECK (yymm ~ '^[0-9]{4}$'),
    last_value integer NOT NULL,
    PRIMARY KEY (prefix, yymm)
);

CREATE TABLE lost_reports (
    id                uuid PRIMARY KEY,
    code              text NOT NULL UNIQUE,
    reporter_id       uuid REFERENCES users (id),
    created_by        uuid REFERENCES users (id),
    proxy_name        text CHECK (char_length(proxy_name) BETWEEN 2 AND 100),
    proxy_phone       text CHECK (char_length(proxy_phone) <= 20),
    category_id       uuid NOT NULL REFERENCES categories (id),
    item_name         text NOT NULL CHECK (char_length(item_name) BETWEEN 3 AND 80),
    description       text NOT NULL CHECK (char_length(description) BETWEEN 20 AND 1000),
    private_details   text CHECK (char_length(private_details) <= 500),
    colours           text[] NOT NULL DEFAULT '{}' CHECK (cardinality(colours) <= 2),
    brand             text CHECK (char_length(brand) <= 40),
    lost_from         timestamptz NOT NULL,
    lost_until        timestamptz NOT NULL,
    location_unknown  boolean NOT NULL DEFAULT false,
    location_note     text CHECK (char_length(location_note) <= 200),
    status            text NOT NULL CHECK (status IN ('reported', 'potential_match', 'under_verification', 'claimed', 'returned', 'closed')),
    close_reason      text CHECK (close_reason IN ('returned', 'found_by_owner', 'no_longer_needed', 'expired', 'duplicate', 'account_deleted')),
    closed_at         timestamptz,
    duplicate_of      uuid REFERENCES lost_reports (id),
    hidden_at         timestamptz,
    hidden_reason     text,
    assist_outcome    jsonb,
    last_activity_at  timestamptz NOT NULL DEFAULT now(),
    stale_reminded_at timestamptz,
    anonymised_at     timestamptz,
    version           integer NOT NULL DEFAULT 1,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CHECK (lost_from <= lost_until),
    CHECK (lost_until - lost_from <= interval '7 days'),
    CHECK ((status = 'closed') = (close_reason IS NOT NULL)),
    CHECK ((status = 'closed') = (closed_at IS NOT NULL)),
    CHECK (reporter_id IS NOT NULL OR proxy_name IS NOT NULL OR anonymised_at IS NOT NULL)
);

CREATE INDEX lost_reports_reporter_status ON lost_reports (reporter_id, status);
CREATE INDEX lost_reports_status_until ON lost_reports (status, lost_until);
CREATE INDEX lost_reports_created ON lost_reports (created_at DESC, id DESC);
CREATE INDEX lost_reports_search_trgm ON lost_reports USING gin (search_key(item_name || ' ' || description) gin_trgm_ops);

CREATE TABLE lost_report_places (
    lost_report_id uuid NOT NULL REFERENCES lost_reports (id) ON DELETE CASCADE,
    position       smallint NOT NULL CHECK (position BETWEEN 1 AND 3),
    location_id    uuid REFERENCES locations (id),
    zone_id        uuid REFERENCES zones (id),
    PRIMARY KEY (lost_report_id, position),
    CHECK (num_nonnulls(location_id, zone_id) = 1)
);

CREATE INDEX lost_report_places_location ON lost_report_places (location_id);

CREATE TABLE found_reports (
    id                         uuid PRIMARY KEY,
    code                       text NOT NULL UNIQUE,
    finder_id                  uuid REFERENCES users (id),
    created_by                 uuid REFERENCES users (id),
    proxy_finder_name          text CHECK (char_length(proxy_finder_name) <= 100),
    category_id                uuid NOT NULL REFERENCES categories (id),
    item_name                  text NOT NULL CHECK (char_length(item_name) BETWEEN 3 AND 80),
    description                text NOT NULL CHECK (char_length(description) BETWEEN 10 AND 500),
    hidden_details             text CHECK (char_length(hidden_details) <= 500),
    colours                    text[] NOT NULL DEFAULT '{}' CHECK (cardinality(colours) <= 2),
    brand                      text CHECK (char_length(brand) <= 40),
    found_at                   timestamptz NOT NULL,
    found_location_id          uuid NOT NULL REFERENCES locations (id),
    location_note              text CHECK (char_length(location_note) <= 200),
    custody_status             text NOT NULL CHECK (custody_status IN ('with_finder', 'in_custody', 'released', 'disposed', 'cancelled')),
    custody_keep               boolean NOT NULL DEFAULT false,
    intended_post_id           uuid REFERENCES posts (id),
    holding_post_id            uuid REFERENCES posts (id),
    storage_slot               text CHECK (char_length(storage_slot) <= 40),
    received_at                timestamptz,
    received_by                uuid REFERENCES users (id),
    retention_due_at           timestamptz,
    retention_extended_at      timestamptz,
    retention_extension_reason text,
    overdue_flagged_at         timestamptz,
    status                     text NOT NULL CHECK (status IN ('reported', 'potential_match', 'under_verification', 'claimed', 'returned', 'unclaimed', 'closed')),
    close_reason               text CHECK (close_reason IN ('returned', 'returned_directly', 'reported_in_error', 'disposed', 'duplicate')),
    close_note                 text CHECK (char_length(close_note) <= 500),
    closed_at                  timestamptz,
    duplicate_of               uuid REFERENCES found_reports (id),
    hidden_at                  timestamptz,
    hidden_reason              text,
    assist_outcome             jsonb,
    anonymised_at              timestamptz,
    version                    integer NOT NULL DEFAULT 1,
    created_at                 timestamptz NOT NULL DEFAULT now(),
    updated_at                 timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'closed') = (close_reason IS NOT NULL)),
    CHECK ((status = 'closed') = (closed_at IS NOT NULL)),
    CHECK (custody_status <> 'in_custody' OR (holding_post_id IS NOT NULL AND received_at IS NOT NULL AND retention_due_at IS NOT NULL))
);

CREATE INDEX found_reports_finder_status ON found_reports (finder_id, status);
CREATE INDEX found_reports_custody ON found_reports (holding_post_id, custody_status, retention_due_at);
CREATE INDEX found_reports_status_found ON found_reports (status, found_at);
CREATE INDEX found_reports_created ON found_reports (created_at DESC, id DESC);
CREATE INDEX found_reports_search_trgm ON found_reports USING gin (search_key(item_name || ' ' || description) gin_trgm_ops);

CREATE TABLE status_history (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    entity_type text NOT NULL CHECK (entity_type IN ('lost_report', 'found_report', 'custody', 'claim', 'match')),
    entity_id   uuid NOT NULL,
    from_status text,
    to_status   text NOT NULL,
    transition  text NOT NULL,
    reason      text,
    actor_id    uuid REFERENCES users (id),
    actor_kind  text NOT NULL CHECK (actor_kind IN ('user', 'system')),
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX status_history_entity ON status_history (entity_type, entity_id, created_at);

CREATE TABLE custody_events (
    id              uuid PRIMARY KEY,
    found_report_id uuid NOT NULL REFERENCES found_reports (id) ON DELETE CASCADE,
    kind            text NOT NULL CHECK (kind IN ('received', 'transferred', 'extended', 'released', 'disposed', 'cancelled')),
    from_post_id    uuid REFERENCES posts (id),
    to_post_id      uuid REFERENCES posts (id),
    officer_id      uuid REFERENCES users (id),
    note            text CHECK (char_length(note) <= 500),
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX custody_events_report ON custody_events (found_report_id, created_at);

-- +goose Down
DROP TABLE custody_events;
DROP TABLE status_history;
DROP TABLE found_reports;
DROP TABLE lost_report_places;
DROP TABLE lost_reports;
DROP TABLE code_sequences;
