-- +goose Up
CREATE TABLE model_versions (
    id            uuid PRIMARY KEY,
    component     text NOT NULL CHECK (component IN ('image_encoder', 'xmodal_text_encoder', 'text_encoder', 'ocr',
                                                     'category_classifier', 'colour_extractor', 'brand_matcher', 'fusion')),
    name          text NOT NULL UNIQUE CHECK (char_length(name) BETWEEN 3 AND 120),
    identity      jsonb NOT NULL DEFAULT '{}',
    config        jsonb,
    config_sha256 bytea,
    eval_run_id   text,
    metrics       jsonb,
    status        text NOT NULL CHECK (status IN ('candidate', 'active', 'retired')),
    activated_at  timestamptz,
    activated_by  uuid REFERENCES users (id),
    retired_at    timestamptz,
    created_at    timestamptz NOT NULL,
    CHECK (component <> 'fusion' OR config IS NOT NULL)
);

CREATE UNIQUE INDEX model_versions_one_active ON model_versions (component) WHERE status = 'active';

CREATE TABLE image_embeddings (
    media_id         uuid NOT NULL REFERENCES media (id) ON DELETE CASCADE,
    model_version_id uuid NOT NULL REFERENCES model_versions (id),
    report_type      text NOT NULL CHECK (report_type IN ('lost', 'found')),
    report_id        uuid NOT NULL,
    embedding        vector(512) NOT NULL,
    created_at       timestamptz NOT NULL,
    PRIMARY KEY (media_id, model_version_id)
);

CREATE INDEX image_embeddings_hnsw ON image_embeddings USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);
CREATE INDEX image_embeddings_report ON image_embeddings (report_type, report_id);

CREATE TABLE text_embeddings_e5 (
    report_type      text NOT NULL CHECK (report_type IN ('lost', 'found')),
    report_id        uuid NOT NULL,
    model_version_id uuid NOT NULL REFERENCES model_versions (id),
    embedding        vector(384) NOT NULL,
    text_sha256      bytea NOT NULL,
    created_at       timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL,
    PRIMARY KEY (report_type, report_id, model_version_id)
);

CREATE INDEX text_embeddings_e5_hnsw ON text_embeddings_e5 USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);

CREATE TABLE text_embeddings_clip (
    report_type      text NOT NULL CHECK (report_type IN ('lost', 'found')),
    report_id        uuid NOT NULL,
    model_version_id uuid NOT NULL REFERENCES model_versions (id),
    embedding        vector(512) NOT NULL,
    text_sha256      bytea NOT NULL,
    created_at       timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL,
    PRIMARY KEY (report_type, report_id, model_version_id)
);

CREATE INDEX text_embeddings_clip_hnsw ON text_embeddings_clip USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);

CREATE TABLE image_attributes (
    media_id        uuid PRIMARY KEY REFERENCES media (id) ON DELETE CASCADE,
    versions        jsonb NOT NULL DEFAULT '{}',
    category_scores jsonb NOT NULL DEFAULT '[]',
    colours         jsonb NOT NULL DEFAULT '[]',
    visible_text    text,
    brands          jsonb NOT NULL DEFAULT '[]',
    created_at      timestamptz NOT NULL
);

CREATE TABLE matches (
    id                 uuid PRIMARY KEY,
    lost_report_id     uuid NOT NULL REFERENCES lost_reports (id) ON DELETE CASCADE,
    found_report_id    uuid NOT NULL REFERENCES found_reports (id) ON DELETE CASCADE,
    visual_mode        text NOT NULL CHECK (visual_mode IN ('image', 'cross_modal', 'none')),
    s_visual           numeric(6, 4) CHECK (s_visual BETWEEN 0 AND 1),
    s_semantic         numeric(6, 4) CHECK (s_semantic BETWEEN 0 AND 1),
    s_attr             numeric(6, 4) CHECK (s_attr BETWEEN 0 AND 1),
    s_text             numeric(6, 4) CHECK (s_text BETWEEN 0 AND 1),
    s_location         numeric(6, 4) CHECK (s_location BETWEEN 0 AND 1),
    s_time             numeric(6, 4) CHECK (s_time BETWEEN 0 AND 1),
    category_factor    numeric(4, 2) NOT NULL CHECK (category_factor BETWEEN 0 AND 1),
    overall            numeric(6, 4) NOT NULL CHECK (overall BETWEEN 0 AND 1),
    -- null once a rescoring drops the pair below the storage floor, the row then stays only for its history
    tier               text CHECK (tier IN ('high', 'medium', 'low')),
    rank_for_lost      smallint CHECK (rank_for_lost BETWEEN 1 AND 10),
    rank_for_found     smallint CHECK (rank_for_found BETWEEN 1 AND 10),
    flags              text[] NOT NULL DEFAULT '{}',
    explanation        jsonb NOT NULL DEFAULT '{}',
    status             text NOT NULL CHECK (status IN ('active', 'superseded', 'dismissed', 'confirmed', 'rejected')),
    owner_dismissed_at timestamptz,
    -- the tier the owner was last told about, BR-NTF-01 folds everything newer into the next notification
    notified_tier      text CHECK (notified_tier IN ('high', 'medium')),
    pinned_by          uuid REFERENCES users (id),
    pinned_at          timestamptz,
    fusion_version_id  uuid NOT NULL REFERENCES model_versions (id),
    component_versions jsonb NOT NULL DEFAULT '{}',
    first_scored_at    timestamptz NOT NULL,
    scored_at          timestamptz NOT NULL,
    version            integer NOT NULL DEFAULT 1,
    UNIQUE (lost_report_id, found_report_id)
);

CREATE INDEX matches_lost ON matches (lost_report_id, status, overall DESC);
CREATE INDEX matches_found ON matches (found_report_id, status, overall DESC);
CREATE INDEX matches_status_tier ON matches (status, tier, scored_at);

CREATE TABLE match_feedback (
    id         uuid PRIMARY KEY,
    match_id   uuid NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    actor_id   uuid REFERENCES users (id),
    actor_role text NOT NULL CHECK (actor_role IN ('member', 'officer', 'admin')),
    verdict    text NOT NULL CHECK (verdict IN ('relevant', 'not_relevant')),
    reason     text CHECK (char_length(reason) <= 300),
    created_at timestamptz NOT NULL
);

CREATE INDEX match_feedback_match ON match_feedback (match_id, created_at);

-- +goose Down
DROP TABLE match_feedback;
DROP TABLE matches;
DROP TABLE image_attributes;
DROP TABLE text_embeddings_clip;
DROP TABLE text_embeddings_e5;
DROP TABLE image_embeddings;
DROP TABLE model_versions;
