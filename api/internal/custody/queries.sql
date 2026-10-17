-- name: ReceiveIntoCustody :one
UPDATE found_reports
SET custody_status = 'in_custody',
    holding_post_id = sqlc.arg(post_id),
    storage_slot = sqlc.arg(storage_slot),
    received_at = sqlc.arg(now)::timestamptz,
    received_by = sqlc.arg(officer_id),
    retention_due_at = sqlc.arg(retention_due_at)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND custody_status = 'with_finder'
RETURNING version;

-- name: InsertCustodyEvent :exec
INSERT INTO custody_events (id, found_report_id, kind, from_post_id, to_post_id, officer_id, note, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListInventory :many
SELECT fr.id, fr.code, fr.item_name, c.name AS category_name, coalesce(c.sensitivity, 'normal')::text AS sensitivity,
       coalesce(fr.storage_slot, '')::text AS storage_slot, fr.received_at::timestamptz AS received_at,
       fr.retention_due_at::timestamptz AS retention_due_at, fr.status,
       coalesce(thumb.id, '00000000-0000-0000-0000-000000000000'::uuid) AS thumb_media_id
FROM found_reports fr
JOIN categories c ON c.id = fr.category_id
LEFT JOIN LATERAL (
    SELECT m.id FROM media m WHERE m.found_report_id = fr.id AND m.state = 'ready' ORDER BY m.position LIMIT 1
) thumb ON true
WHERE fr.holding_post_id = sqlc.arg(post_id) AND fr.custody_status = 'in_custody'
ORDER BY
  CASE WHEN sqlc.arg(sort)::text = 'retention_due_at' THEN fr.retention_due_at END ASC,
  fr.received_at DESC
LIMIT 500;

-- name: IsOfficerOfPost :one
SELECT EXISTS (SELECT 1 FROM officer_posts WHERE user_id = $1 AND post_id = $2);
