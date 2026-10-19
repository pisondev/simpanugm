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

-- name: TransferCustody :one
UPDATE found_reports
SET holding_post_id = sqlc.arg(to_post_id),
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND custody_status = 'in_custody'
RETURNING version;

-- name: ExtendRetention :one
UPDATE found_reports
SET retention_due_at = retention_due_at + (sqlc.arg(days)::int * interval '1 day'),
    retention_extended_at = sqlc.arg(now)::timestamptz,
    retention_extension_reason = sqlc.arg(reason),
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND custody_status = 'in_custody' AND retention_extended_at IS NULL
RETURNING version;

-- name: InsertDisposition :exec
INSERT INTO dispositions (found_report_id, kind, recipient, note, officer_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DisposeCustody :one
UPDATE found_reports
SET status = 'closed',
    close_reason = 'disposed',
    close_note = sqlc.arg(note),
    closed_at = sqlc.arg(now)::timestamptz,
    custody_status = 'disposed',
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status = 'unclaimed' AND custody_status = 'in_custody'
RETURNING version;

-- name: RevertUnclaimed :one
UPDATE found_reports
SET status = 'reported',
    retention_due_at = sqlc.arg(retention_due_at)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status = 'unclaimed' AND custody_status = 'in_custody'
RETURNING version;

-- name: ListRetentionExpired :many
SELECT fr.id, fr.code, fr.version, fr.holding_post_id
FROM found_reports fr
WHERE fr.custody_status = 'in_custody'
  AND fr.status IN ('reported', 'potential_match')
  AND fr.retention_due_at <= sqlc.arg(now)::timestamptz
  AND NOT EXISTS (
      SELECT 1 FROM claims c WHERE c.found_report_id = fr.id AND c.status IN ('submitted', 'under_review', 'info_requested', 'approved', 'on_hold')
  );

-- name: MarkFoundUnclaimed :one
UPDATE found_reports
SET status = 'unclaimed',
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status IN ('reported', 'potential_match') AND custody_status = 'in_custody'
RETURNING version;

-- name: ListRetentionWarningPosts :many
SELECT fr.holding_post_id, count(*)::int AS item_count
FROM found_reports fr
WHERE fr.custody_status = 'in_custody'
  AND fr.status IN ('reported', 'potential_match')
  AND fr.holding_post_id IS NOT NULL
  AND fr.retention_due_at > sqlc.arg(now)::timestamptz
  AND fr.retention_due_at <= sqlc.arg(warn_before)::timestamptz
GROUP BY fr.holding_post_id;

-- name: ListFinderCustodyDue24h :many
SELECT fr.id, fr.code, fr.finder_id, fr.item_name, coalesce(ip.name, '')::text AS post_name
FROM found_reports fr
LEFT JOIN posts ip ON ip.id = fr.intended_post_id
WHERE fr.custody_status = 'with_finder'
  AND fr.status IN ('reported', 'potential_match', 'under_verification')
  AND fr.finder_id IS NOT NULL
  AND fr.created_at <= sqlc.arg(before_24h)::timestamptz
  AND NOT EXISTS (
      SELECT 1 FROM notifications n WHERE n.entity_type = 'found_report' AND n.entity_id = fr.id AND n.event = 'found.custody_reminder_24h'
  );

-- name: ListFinderCustodyDue48h :many
SELECT fr.id, fr.code, fr.finder_id, fr.intended_post_id, fr.item_name, coalesce(ip.name, '')::text AS post_name
FROM found_reports fr
LEFT JOIN posts ip ON ip.id = fr.intended_post_id
WHERE fr.custody_status = 'with_finder'
  AND fr.status IN ('reported', 'potential_match', 'under_verification')
  AND fr.created_at <= sqlc.arg(before_48h)::timestamptz
  AND NOT EXISTS (
      SELECT 1 FROM notifications n WHERE n.entity_type = 'found_report' AND n.entity_id = fr.id AND n.event = 'found.custody_reminder_48h'
  );

-- name: ListFinderCustodyDue72h :many
SELECT fr.id, fr.code, fr.intended_post_id, fr.version
FROM found_reports fr
WHERE fr.custody_status = 'with_finder'
  AND fr.status IN ('reported', 'potential_match', 'under_verification')
  AND fr.overdue_flagged_at IS NULL
  AND fr.created_at <= sqlc.arg(before_72h)::timestamptz;

-- name: MarkFoundCustodyOverdue :one
UPDATE found_reports
SET overdue_flagged_at = sqlc.arg(now)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND overdue_flagged_at IS NULL
RETURNING version;
