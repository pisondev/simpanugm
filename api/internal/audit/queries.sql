-- name: InsertAudit :exec
INSERT INTO audit_logs (occurred_at, actor_id, actor_role, action, entity_type, entity_id, changes, request_id, ip, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: ListAuditForEntity :many
SELECT * FROM audit_logs WHERE entity_type = $1 AND entity_id = $2 ORDER BY occurred_at, id;

-- name: ListAuditLogs :many
SELECT al.id, al.occurred_at, al.actor_id, al.actor_role, al.action, al.entity_type, al.entity_id,
       al.changes, al.request_id, al.ip, al.user_agent, u.full_name AS actor_name
FROM audit_logs al
LEFT JOIN users u ON u.id = al.actor_id
WHERE (sqlc.narg(actor_id)::uuid IS NULL OR al.actor_id = sqlc.narg(actor_id)::uuid)
  AND (sqlc.narg(action)::text IS NULL OR al.action = sqlc.narg(action)::text)
  AND (sqlc.narg(entity_type)::text IS NULL OR al.entity_type = sqlc.narg(entity_type)::text)
  AND (sqlc.narg(entity_id)::uuid IS NULL OR al.entity_id = sqlc.narg(entity_id)::uuid)
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR al.occurred_at >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR al.occurred_at <= sqlc.narg(to_time)::timestamptz)
  AND (sqlc.narg(cursor_time)::timestamptz IS NULL OR (al.occurred_at, al.id) < (sqlc.narg(cursor_time)::timestamptz, sqlc.narg(cursor_id)::bigint))
ORDER BY al.occurred_at DESC, al.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ExportAuditLogs :many
SELECT al.id, al.occurred_at, al.actor_id, al.actor_role, al.action, al.entity_type, al.entity_id,
       al.changes, al.request_id, al.ip, al.user_agent, u.full_name AS actor_name
FROM audit_logs al
LEFT JOIN users u ON u.id = al.actor_id
WHERE (sqlc.narg(actor_id)::uuid IS NULL OR al.actor_id = sqlc.narg(actor_id)::uuid)
  AND (sqlc.narg(action)::text IS NULL OR al.action = sqlc.narg(action)::text)
  AND (sqlc.narg(entity_type)::text IS NULL OR al.entity_type = sqlc.narg(entity_type)::text)
  AND (sqlc.narg(entity_id)::uuid IS NULL OR al.entity_id = sqlc.narg(entity_id)::uuid)
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR al.occurred_at >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR al.occurred_at <= sqlc.narg(to_time)::timestamptz)
ORDER BY al.occurred_at DESC, al.id DESC
LIMIT sqlc.arg(row_limit);
