-- name: InsertAudit :exec
INSERT INTO audit_logs (occurred_at, actor_id, actor_role, action, entity_type, entity_id, changes, request_id, ip, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: ListAuditForEntity :many
SELECT * FROM audit_logs WHERE entity_type = $1 AND entity_id = $2 ORDER BY occurred_at, id;
