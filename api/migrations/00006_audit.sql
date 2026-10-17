-- +goose Up
CREATE TABLE audit_logs (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    actor_id    uuid,
    actor_role  text,
    action      text NOT NULL,
    entity_type text NOT NULL,
    entity_id   uuid,
    changes     jsonb,
    request_id  text,
    ip          inet,
    user_agent  text
);

CREATE INDEX audit_logs_entity ON audit_logs (entity_type, entity_id, occurred_at);
CREATE INDEX audit_logs_actor ON audit_logs (actor_id, occurred_at);
CREATE INDEX audit_logs_occurred ON audit_logs (occurred_at);

-- the application role may only append, FR-AUDIT-02
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'simpan_app') THEN
        REVOKE UPDATE, DELETE, TRUNCATE ON audit_logs FROM simpan_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE audit_logs;
