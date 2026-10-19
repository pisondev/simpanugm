-- name: ListSettings :many
SELECT key, value FROM system_settings ORDER BY key;

-- name: UpsertSetting :exec
INSERT INTO system_settings (key, value, updated_at, updated_by)
VALUES ($1, $2, sqlc.arg(now)::timestamptz, $3)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by;

-- name: ListSettingsDetailed :many
SELECT s.key, s.value, s.updated_at, s.updated_by, u.full_name AS updated_by_name
FROM system_settings s
LEFT JOIN users u ON u.id = s.updated_by
ORDER BY s.key;

