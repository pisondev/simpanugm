-- +goose Up
CREATE TABLE media (
    id              uuid PRIMARY KEY,
    uploaded_by     uuid NOT NULL REFERENCES users (id),
    purpose         text NOT NULL CHECK (purpose IN ('lost_report', 'found_report')),
    lost_report_id  uuid REFERENCES lost_reports (id),
    found_report_id uuid REFERENCES found_reports (id),
    state           text NOT NULL CHECK (state IN ('staged', 'ready', 'deleted')),
    mime            text NOT NULL CHECK (mime IN ('image/jpeg', 'image/png', 'image/webp')),
    width           integer NOT NULL,
    height          integer NOT NULL,
    bytes           integer NOT NULL,
    sha256          bytea NOT NULL,
    phash           bigint NOT NULL,
    storage_keys    jsonb NOT NULL,
    position        smallint NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    attached_at     timestamptz,
    deleted_at      timestamptz,
    CHECK (num_nonnulls(lost_report_id, found_report_id) <= 1),
    CHECK (state <> 'ready' OR num_nonnulls(lost_report_id, found_report_id) = 1)
);

CREATE UNIQUE INDEX media_lost_sha ON media (lost_report_id, sha256) WHERE lost_report_id IS NOT NULL AND state <> 'deleted';
CREATE UNIQUE INDEX media_found_sha ON media (found_report_id, sha256) WHERE found_report_id IS NOT NULL AND state <> 'deleted';
CREATE INDEX media_state_created ON media (state, created_at);
CREATE INDEX media_lost ON media (lost_report_id, position) WHERE lost_report_id IS NOT NULL;
CREATE INDEX media_found ON media (found_report_id, position) WHERE found_report_id IS NOT NULL;

-- +goose Down
DROP TABLE media;
