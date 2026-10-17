-- +goose Up
CREATE TABLE users (
    id                  uuid PRIMARY KEY,
    email               citext NOT NULL UNIQUE,
    full_name           text NOT NULL CHECK (char_length(full_name) BETWEEN 2 AND 100),
    nim_nip             text CHECK (char_length(nim_nip) <= 30),
    phone               text CHECK (char_length(phone) <= 20),
    password_hash       text,
    role                text NOT NULL CHECK (role IN ('member', 'officer', 'admin')),
    status              text NOT NULL CHECK (status IN ('pending_verification', 'active', 'suspended', 'deleted')),
    email_verified_at   timestamptz,
    claim_restricted_at timestamptz,
    privacy_version     text,
    privacy_accepted_at timestamptz,
    invited_by          uuid REFERENCES users (id),
    last_login_at       timestamptz,
    deleted_at          timestamptz,
    version             integer NOT NULL DEFAULT 1,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CHECK (status <> 'active' OR password_hash IS NOT NULL)
);

CREATE INDEX users_full_name_trgm ON users USING gin (search_key(full_name) gin_trgm_ops);
CREATE INDEX users_role_status ON users (role, status);

CREATE TABLE sessions (
    id              uuid PRIMARY KEY,
    user_id         uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash      bytea NOT NULL UNIQUE,
    created_at      timestamptz NOT NULL DEFAULT now(),
    last_seen_at    timestamptz NOT NULL DEFAULT now(),
    expires_at      timestamptz NOT NULL,
    idle_expires_at timestamptz,
    ip              inet,
    user_agent      text,
    revoked_at      timestamptz
);

CREATE INDEX sessions_user_active ON sessions (user_id) WHERE revoked_at IS NULL;
CREATE INDEX sessions_expires ON sessions (expires_at);

CREATE TABLE auth_tokens (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose    text NOT NULL CHECK (purpose IN ('verify_email', 'reset_password', 'invite', 'change_email')),
    token_hash bytea NOT NULL UNIQUE,
    new_email  citext,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX auth_tokens_user_purpose ON auth_tokens (user_id, purpose);

CREATE TABLE idempotency_keys (
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    key          text NOT NULL CHECK (char_length(key) BETWEEN 8 AND 100),
    endpoint     text NOT NULL,
    request_hash bytea NOT NULL,
    resource_id  uuid,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    PRIMARY KEY (user_id, key)
);

CREATE INDEX idempotency_keys_expires ON idempotency_keys (expires_at);

-- +goose Down
DROP TABLE idempotency_keys;
DROP TABLE auth_tokens;
DROP TABLE sessions;
DROP TABLE users;
