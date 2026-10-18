-- name: GetRecipient :one
SELECT id, email, full_name, role, status FROM users WHERE id = $1;

-- name: ListPreferences :many
SELECT event_group, channel, enabled FROM notification_preferences WHERE user_id = $1;

-- name: UpsertPreference :exec
INSERT INTO notification_preferences (user_id, event_group, channel, enabled, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, event_group, channel) DO UPDATE SET enabled = excluded.enabled, updated_at = excluded.updated_at;

-- name: InsertNotification :exec
INSERT INTO notifications (id, recipient_id, event, title, body, link, entity_type, entity_id, email_status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: SignalRecipient :exec
SELECT pg_notify('notifications', sqlc.arg(recipient_id)::text);

-- name: ListNotifications :many
SELECT * FROM notifications
WHERE recipient_id = sqlc.arg(recipient_id)
  AND (NOT sqlc.arg(unread_only)::bool OR read_at IS NULL)
  AND (sqlc.narg(cursor_created)::timestamptz IS NULL
       OR (created_at, id) < (sqlc.narg(cursor_created)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListNotificationsAfter :many
SELECT * FROM notifications
WHERE recipient_id = sqlc.arg(recipient_id)
  AND (created_at, id) > (sqlc.arg(after_created)::timestamptz, sqlc.arg(after_id)::uuid)
ORDER BY created_at, id
LIMIT 50;

-- name: GetNotificationCreated :one
SELECT created_at FROM notifications WHERE id = $1 AND recipient_id = $2;

-- name: CountUnread :one
SELECT count(*)::int FROM notifications WHERE recipient_id = $1 AND read_at IS NULL;

-- name: MarkRead :execrows
UPDATE notifications SET read_at = $3 WHERE id = $1 AND recipient_id = $2 AND read_at IS NULL;

-- name: MarkAllRead :execrows
UPDATE notifications SET read_at = $2 WHERE recipient_id = $1 AND read_at IS NULL;

-- name: SetEmailStatus :exec
UPDATE notifications SET email_status = $2 WHERE id = $1;

-- name: LastNotificationOf :one
SELECT created_at FROM notifications
WHERE recipient_id = $1 AND event = $2 AND entity_id = $3
ORDER BY created_at DESC
LIMIT 1;

-- name: ListDigestRecipients :many
SELECT DISTINCT n.recipient_id, u.email, u.full_name
FROM notifications n
JOIN users u ON u.id = n.recipient_id
WHERE n.created_at >= sqlc.arg(since) AND n.created_at < sqlc.arg(until)
  AND n.event = ANY(sqlc.arg(events)::text[])
  AND u.role IN ('officer', 'admin') AND u.status = 'active';

-- name: ListDigestItems :many
SELECT title, body, link, created_at FROM notifications
WHERE recipient_id = sqlc.arg(recipient_id) AND created_at >= sqlc.arg(since) AND created_at < sqlc.arg(until)
  AND event = ANY(sqlc.arg(events)::text[])
ORDER BY created_at
LIMIT 100;

-- name: DeleteOldNotifications :execrows
DELETE FROM notifications WHERE created_at < $1;

-- name: ListUserIDsForPosts :many
SELECT DISTINCT u.id FROM users u
JOIN officer_posts op ON op.user_id = u.id
WHERE op.post_id = ANY(sqlc.arg(post_ids)::uuid[]) AND u.role IN ('officer', 'admin') AND u.status = 'active';
