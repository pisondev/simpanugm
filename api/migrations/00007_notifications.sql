-- +goose Up
CREATE TABLE notifications (
    id           uuid PRIMARY KEY,
    recipient_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    event        text NOT NULL CHECK (event ~ '^[a-z]+\.[a-z_0-9]+$'),
    title        text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    body         text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 1000),
    link         text NOT NULL CHECK (link LIKE '/%'),
    entity_type  text,
    entity_id    uuid,
    read_at      timestamptz,
    email_status text NOT NULL DEFAULT 'none' CHECK (email_status IN ('none', 'pending', 'sent', 'failed', 'skipped')),
    created_at   timestamptz NOT NULL
);

CREATE INDEX notifications_recipient_newest ON notifications (recipient_id, created_at DESC, id DESC);
CREATE INDEX notifications_unread ON notifications (recipient_id) WHERE read_at IS NULL;
CREATE INDEX notifications_created ON notifications (created_at);

CREATE TABLE notification_preferences (
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    event_group text NOT NULL CHECK (event_group IN ('match', 'found', 'claim')),
    channel     text NOT NULL CHECK (channel IN ('app', 'email')),
    enabled     boolean NOT NULL,
    updated_at  timestamptz NOT NULL,
    PRIMARY KEY (user_id, event_group, channel)
);

-- +goose Down
DROP TABLE notification_preferences;
DROP TABLE notifications;
