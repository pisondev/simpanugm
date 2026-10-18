-- name: InsertMedia :exec
INSERT INTO media (id, uploaded_by, purpose, state, mime, width, height, bytes, sha256, phash, storage_keys, created_at)
VALUES ($1, $2, $3, 'staged', $4, $5, $6, $7, $8, $9, $10, $11);

-- candidate_owner: the viewer owns a lost report for which this found photo is a visible candidate (BR-VIS 5.2)
-- name: GetMediaForAccess :one
SELECT m.*, lr.reporter_id AS lost_reporter_id, fr.finder_id AS found_finder_id,
       coalesce(lc.sensitivity, fc.sensitivity, '')::text AS sensitivity,
       EXISTS (
           SELECT 1 FROM matches mt JOIN lost_reports ol ON ol.id = mt.lost_report_id
           WHERE mt.found_report_id = m.found_report_id AND ol.reporter_id = sqlc.arg(viewer)::uuid
             AND mt.status = 'active' AND mt.tier IN ('high', 'medium') AND mt.rank_for_lost IS NOT NULL AND mt.owner_dismissed_at IS NULL
       )::bool AS candidate_owner
FROM media m
LEFT JOIN lost_reports lr ON lr.id = m.lost_report_id
LEFT JOIN categories lc ON lc.id = lr.category_id
LEFT JOIN found_reports fr ON fr.id = m.found_report_id
LEFT JOIN categories fc ON fc.id = fr.category_id
WHERE m.id = sqlc.arg(id);

-- name: LockMedia :many
SELECT * FROM media WHERE id = ANY(sqlc.arg(ids)::uuid[]) ORDER BY id FOR UPDATE;

-- name: AttachMedia :exec
UPDATE media
SET lost_report_id = sqlc.narg(lost_report_id),
    found_report_id = sqlc.narg(found_report_id),
    state = 'ready',
    position = sqlc.arg(position),
    attached_at = coalesce(attached_at, sqlc.arg(now)::timestamptz)
WHERE id = sqlc.arg(id);

-- name: ListReportMedia :many
SELECT * FROM media
WHERE state = 'ready'
  AND (lost_report_id = sqlc.narg(lost_report_id)::uuid OR found_report_id = sqlc.narg(found_report_id)::uuid)
ORDER BY position, created_at;

-- name: MarkMediaDeleted :exec
UPDATE media SET state = 'deleted', deleted_at = $2 WHERE id = $1;

-- name: ListStagedMediaBefore :many
SELECT id, storage_keys FROM media WHERE state = 'staged' AND created_at < $1 ORDER BY created_at LIMIT 500;
