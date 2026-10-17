-- name: ListSettings :many
SELECT key, value FROM system_settings ORDER BY key;
