-- name: GetOfficerSummary :one
SELECT
    (SELECT count(*)::int FROM found_reports fr
     WHERE fr.custody_status = 'in_custody' AND fr.status NOT IN ('returned', 'closed')
       AND (sqlc.narg(post_ids)::uuid[] IS NULL OR fr.holding_post_id = ANY(sqlc.narg(post_ids)::uuid[]))
    ) AS in_custody,
    (SELECT count(*)::int FROM found_reports fr
     WHERE fr.custody_status = 'in_custody' AND fr.status NOT IN ('returned', 'closed')
       AND fr.retention_due_at <= sqlc.arg(due_threshold)::timestamptz
       AND (sqlc.narg(post_ids)::uuid[] IS NULL OR fr.holding_post_id = ANY(sqlc.narg(post_ids)::uuid[]))
    ) AS retention_due,
    (SELECT count(*)::int FROM found_reports fr
     WHERE fr.custody_status = 'with_finder' AND fr.status IN ('reported', 'potential_match')
       AND (sqlc.narg(post_ids)::uuid[] IS NULL OR fr.holding_post_id = ANY(sqlc.narg(post_ids)::uuid[]) OR fr.intended_post_id = ANY(sqlc.narg(post_ids)::uuid[]))
    ) AS with_finder,
    (SELECT count(*)::int FROM claims c
     JOIN found_reports fr ON fr.id = c.found_report_id
     WHERE c.status IN ('submitted', 'under_review')
       AND (sqlc.narg(post_ids)::uuid[] IS NULL OR fr.holding_post_id = ANY(sqlc.narg(post_ids)::uuid[]) OR fr.intended_post_id = ANY(sqlc.narg(post_ids)::uuid[]))
    ) AS claims_pending,
    (SELECT count(*)::int FROM found_reports fr
     WHERE fr.status = 'unclaimed' AND fr.custody_status = 'in_custody'
       AND (sqlc.narg(post_ids)::uuid[] IS NULL OR fr.holding_post_id = ANY(sqlc.narg(post_ids)::uuid[]))
    ) AS unclaimed,
    (SELECT count(*)::int FROM found_reports fr
     WHERE fr.created_at >= sqlc.arg(today_start)::timestamptz
       AND (sqlc.narg(post_ids)::uuid[] IS NULL OR fr.holding_post_id = ANY(sqlc.narg(post_ids)::uuid[]) OR fr.intended_post_id = ANY(sqlc.narg(post_ids)::uuid[]))
    ) AS found_today,
    (SELECT count(DISTINCT fr.id)::int FROM found_reports fr
     JOIN matches m ON m.found_report_id = fr.id AND m.status = 'active' AND m.tier = 'high'
     WHERE fr.status IN ('reported', 'potential_match')
       AND (sqlc.narg(post_ids)::uuid[] IS NULL OR fr.holding_post_id = ANY(sqlc.narg(post_ids)::uuid[]) OR fr.intended_post_id = ANY(sqlc.narg(post_ids)::uuid[]))
       AND NOT EXISTS (
           SELECT 1 FROM claims c
           WHERE c.found_report_id = fr.id AND c.status NOT IN ('rejected', 'withdrawn', 'cancelled')
       )
    ) AS high_match_no_claim;

-- name: ListQueueClaims :many
SELECT c.id AS claim_id, c.code AS claim_code, c.status AS claim_status, c.created_at AS claim_created_at, c.updated_at AS claim_updated_at,
       fr.id AS found_id, fr.code AS found_code, fr.item_name AS found_item_name, fr.custody_status, fr.created_at AS found_created_at,
       fr.received_at, fr.retention_due_at, fr.found_at,
       coalesce(fr.holding_post_id, fr.intended_post_id) AS post_id,
       coalesce(hp.name, ip.name, '')::text AS post_name,
       cat.name AS category_name, cat.slug AS category_slug, coalesce(cat.sensitivity, 'normal')::text AS category_sensitivity,
       coalesce(best_m.score, 0)::float8 AS best_match_score,
       coalesce(thumb.id, '00000000-0000-0000-0000-000000000000'::uuid) AS thumb_media_id
FROM claims c
JOIN found_reports fr ON fr.id = c.found_report_id
JOIN categories cat ON cat.id = fr.category_id
LEFT JOIN posts hp ON hp.id = fr.holding_post_id
LEFT JOIN posts ip ON ip.id = fr.intended_post_id
LEFT JOIN LATERAL (
    SELECT max(m.overall)::float8 AS score
    FROM matches m
    WHERE m.found_report_id = fr.id AND m.status = 'active'
) best_m ON true
LEFT JOIN LATERAL (
    SELECT m.id FROM media m WHERE m.found_report_id = fr.id AND m.state = 'ready' ORDER BY m.position LIMIT 1
) thumb ON true
WHERE c.status IN ('submitted', 'under_review')
  AND (sqlc.narg(post_ids)::uuid[] IS NULL OR fr.holding_post_id = ANY(sqlc.narg(post_ids)::uuid[]) OR fr.intended_post_id = ANY(sqlc.narg(post_ids)::uuid[]))
  AND (sqlc.narg(specific_post_id)::uuid IS NULL OR fr.holding_post_id = sqlc.narg(specific_post_id)::uuid OR fr.intended_post_id = sqlc.narg(specific_post_id)::uuid);

-- name: ListQueueFoundReports :many
SELECT fr.id AS found_id, fr.code AS found_code, fr.item_name AS found_item_name, fr.custody_status, fr.status AS found_status,
       fr.close_reason, fr.created_at AS found_created_at, fr.updated_at AS found_updated_at,
       fr.received_at, fr.retention_due_at, fr.overdue_flagged_at, fr.found_at,
       coalesce(fr.holding_post_id, fr.intended_post_id) AS post_id,
       coalesce(hp.name, ip.name, '')::text AS post_name,
       cat.name AS category_name, cat.slug AS category_slug, coalesce(cat.sensitivity, 'normal')::text AS category_sensitivity,
       coalesce(best_m.score, 0)::float8 AS best_match_score,
       coalesce(best_m.has_high, false)::boolean AS has_high_match,
       coalesce(claim_info.has_active_claim, false)::boolean AS has_active_claim,
       coalesce(thumb.id, '00000000-0000-0000-0000-000000000000'::uuid) AS thumb_media_id
FROM found_reports fr
JOIN categories cat ON cat.id = fr.category_id
LEFT JOIN posts hp ON hp.id = fr.holding_post_id
LEFT JOIN posts ip ON ip.id = fr.intended_post_id
LEFT JOIN LATERAL (
    SELECT max(m.overall)::float8 AS score,
           bool_or(m.tier = 'high') AS has_high
    FROM matches m
    WHERE m.found_report_id = fr.id AND m.status = 'active'
) best_m ON true
LEFT JOIN LATERAL (
    SELECT bool_or(c.status NOT IN ('rejected', 'withdrawn', 'cancelled')) AS has_active_claim
    FROM claims c
    WHERE c.found_report_id = fr.id
) claim_info ON true
LEFT JOIN LATERAL (
    SELECT m.id FROM media m WHERE m.found_report_id = fr.id AND m.state = 'ready' ORDER BY m.position LIMIT 1
) thumb ON true
WHERE (
    (fr.status IN ('reported', 'potential_match') AND coalesce(best_m.has_high, false) AND NOT coalesce(claim_info.has_active_claim, false))
    OR
    (fr.custody_status = 'with_finder' AND fr.status IN ('reported', 'potential_match') AND (fr.created_at <= sqlc.arg(overdue_threshold)::timestamptz OR fr.overdue_flagged_at IS NOT NULL))
    OR
    (fr.custody_status = 'in_custody' AND fr.status NOT IN ('returned', 'unclaimed', 'closed') AND fr.retention_due_at <= sqlc.arg(due_threshold)::timestamptz AND NOT coalesce(claim_info.has_active_claim, false))
    OR
    (fr.status = 'unclaimed' AND fr.custody_status = 'in_custody')
    OR
    (fr.status = 'closed' AND fr.close_reason = 'returned_directly')
)
  AND (sqlc.narg(post_ids)::uuid[] IS NULL OR fr.holding_post_id = ANY(sqlc.narg(post_ids)::uuid[]) OR fr.intended_post_id = ANY(sqlc.narg(post_ids)::uuid[]))
  AND (sqlc.narg(specific_post_id)::uuid IS NULL OR fr.holding_post_id = sqlc.narg(specific_post_id)::uuid OR fr.intended_post_id = sqlc.narg(specific_post_id)::uuid);

-- name: ListDuplicateLinks :many
SELECT dl.id, dl.report_type, dl.report_a, dl.report_b, dl.score::float8 AS score,
       dl.status, dl.reviewed_by, dl.reviewed_at, dl.created_at
FROM duplicate_links dl
WHERE (sqlc.narg(status)::text IS NULL OR dl.status = sqlc.narg(status))
  AND (sqlc.narg(report_type)::text IS NULL OR dl.report_type = sqlc.narg(report_type))
ORDER BY dl.status = 'suggested' DESC, dl.score DESC, dl.created_at DESC
LIMIT sqlc.arg(lim)::int;

-- name: GetDuplicateLink :one
SELECT dl.id, dl.report_type, dl.report_a, dl.report_b, dl.score::float8 AS score,
       dl.status, dl.reviewed_by, dl.reviewed_at, dl.created_at
FROM duplicate_links dl
WHERE dl.id = sqlc.arg(id);

-- name: UpsertDuplicateLink :one
INSERT INTO duplicate_links (id, report_type, report_a, report_b, score, status, reviewed_by, reviewed_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(report_type), sqlc.arg(report_a), sqlc.arg(report_b), sqlc.arg(score), sqlc.arg(status), sqlc.narg(reviewed_by), sqlc.narg(reviewed_at), sqlc.arg(created_at))
ON CONFLICT (report_type, LEAST(report_a, report_b), GREATEST(report_a, report_b))
DO UPDATE SET score = EXCLUDED.score,
              status = CASE WHEN duplicate_links.status = 'confirmed' THEN duplicate_links.status ELSE EXCLUDED.status END,
              reviewed_by = coalesce(EXCLUDED.reviewed_by, duplicate_links.reviewed_by),
              reviewed_at = coalesce(EXCLUDED.reviewed_at, duplicate_links.reviewed_at)
RETURNING id, report_type, report_a, report_b, score::float8 AS score, status, reviewed_by, reviewed_at, created_at;

-- name: UpdateDuplicateLinkStatus :one
UPDATE duplicate_links
SET status = sqlc.arg(status),
    reviewed_by = sqlc.arg(reviewed_by),
    reviewed_at = sqlc.arg(reviewed_at)
WHERE id = sqlc.arg(id)
RETURNING id, report_type, report_a, report_b, score::float8 AS score, status, reviewed_by, reviewed_at, created_at;

-- name: GetFoundReportSummary :one
SELECT fr.id, fr.code, fr.item_name, fr.status, fr.found_at,
       cat.name AS category_name, loc.name AS location_name,
       coalesce(thumb.id, '00000000-0000-0000-0000-000000000000'::uuid) AS thumb_media_id
FROM found_reports fr
JOIN categories cat ON cat.id = fr.category_id
JOIN locations loc ON loc.id = fr.found_location_id
LEFT JOIN LATERAL (
    SELECT m.id FROM media m WHERE m.found_report_id = fr.id AND m.state = 'ready' ORDER BY m.position LIMIT 1
) thumb ON true
WHERE fr.id = sqlc.arg(id);

-- name: GetLostReportSummary :one
SELECT lr.id, lr.code, lr.item_name, lr.status, lr.lost_from AS occurred_at,
       cat.name AS category_name,
       coalesce(loc.name, '')::text AS location_name,
       coalesce(thumb.id, '00000000-0000-0000-0000-000000000000'::uuid) AS thumb_media_id
FROM lost_reports lr
JOIN categories cat ON cat.id = lr.category_id
LEFT JOIN LATERAL (
    SELECT l.name FROM lost_report_places lrp
    JOIN locations l ON l.id = lrp.location_id
    WHERE lrp.lost_report_id = lr.id
    ORDER BY lrp.position LIMIT 1
) loc ON true
LEFT JOIN LATERAL (
    SELECT m.id FROM media m WHERE m.lost_report_id = lr.id AND m.state = 'ready' ORDER BY m.position LIMIT 1
) thumb ON true
WHERE lr.id = sqlc.arg(id);

-- name: MergeCloseFoundReport :exec
UPDATE found_reports
SET status = 'closed',
    close_reason = 'duplicate',
    close_note = 'Merged into duplicate survivor',
    closed_at = sqlc.arg(closed_at)::timestamptz,
    duplicate_of = sqlc.arg(survivor_id),
    custody_status = 'cancelled',
    version = version + 1,
    updated_at = sqlc.arg(closed_at)::timestamptz
WHERE id = sqlc.arg(duplicate_id);

-- name: MergeCloseLostReport :exec
UPDATE lost_reports
SET status = 'closed',
    close_reason = 'duplicate',
    closed_at = sqlc.arg(closed_at)::timestamptz,
    duplicate_of = sqlc.arg(survivor_id),
    version = version + 1,
    updated_at = sqlc.arg(closed_at)::timestamptz
WHERE id = sqlc.arg(duplicate_id);

-- name: MoveFoundMediaToSurvivor :exec
UPDATE media
SET found_report_id = sqlc.arg(survivor_id),
    position = media.position + coalesce((SELECT max(m2.position) FROM media m2 WHERE m2.found_report_id = sqlc.arg(survivor_id)), 0)::smallint + 1
WHERE media.found_report_id = sqlc.arg(duplicate_id)
  AND NOT EXISTS (
      SELECT 1 FROM media existing
      WHERE existing.found_report_id = sqlc.arg(survivor_id)
        AND existing.sha256 = media.sha256
        AND existing.state <> 'deleted'
  );

-- name: DeleteUnmovedFoundMedia :exec
UPDATE media
SET state = 'deleted', deleted_at = sqlc.arg(now)::timestamptz
WHERE media.found_report_id = sqlc.arg(duplicate_id) AND media.state <> 'deleted';

-- name: MoveLostMediaToSurvivor :exec
UPDATE media
SET lost_report_id = sqlc.arg(survivor_id),
    position = media.position + coalesce((SELECT max(m2.position) FROM media m2 WHERE m2.lost_report_id = sqlc.arg(survivor_id)), 0)::smallint + 1
WHERE media.lost_report_id = sqlc.arg(duplicate_id)
  AND NOT EXISTS (
      SELECT 1 FROM media existing
      WHERE existing.lost_report_id = sqlc.arg(survivor_id)
        AND existing.sha256 = media.sha256
        AND existing.state <> 'deleted'
  );

-- name: DeleteUnmovedLostMedia :exec
UPDATE media
SET state = 'deleted', deleted_at = sqlc.arg(now)::timestamptz
WHERE media.lost_report_id = sqlc.arg(duplicate_id) AND media.state <> 'deleted';

-- name: MoveFoundClaimsToSurvivor :exec
UPDATE claims
SET found_report_id = sqlc.arg(survivor_id)
WHERE claims.found_report_id = sqlc.arg(duplicate_id)
  AND NOT EXISTS (
      SELECT 1 FROM claims existing
      WHERE existing.found_report_id = sqlc.arg(survivor_id)
        AND existing.claimant_id IS NOT NULL
        AND existing.claimant_id = claims.claimant_id
        AND existing.status IN ('submitted', 'under_review', 'info_requested', 'approved', 'on_hold')
  );

-- name: RejectConflictingFoundClaims :exec
UPDATE claims
SET status = 'rejected',
    decision_reason = 'duplicate_claim',
    decision_note = 'Laporan temuan digabungkan dengan laporan lain yang sudah Anda klaim',
    decided_at = sqlc.arg(now)::timestamptz
WHERE claims.found_report_id = sqlc.arg(duplicate_id)
  AND claims.status IN ('submitted', 'under_review', 'info_requested', 'on_hold');

-- name: MoveLostClaimsToSurvivor :exec
UPDATE claims
SET lost_report_id = sqlc.arg(survivor_id)
WHERE lost_report_id = sqlc.arg(duplicate_id);

-- name: SupersedeFoundMatches :exec
UPDATE matches
SET status = 'superseded'
WHERE found_report_id = sqlc.arg(duplicate_id);

-- name: SupersedeLostMatches :exec
UPDATE matches
SET status = 'superseded'
WHERE lost_report_id = sqlc.arg(duplicate_id);

-- name: ListDuplicateFoundCandidates :many
SELECT fr.id, fr.code, fr.item_name, fr.description, fr.found_at,
       loc.lat, loc.lng, loc.name AS location_name, cat.name AS category_name,
       coalesce(thumb.id, '00000000-0000-0000-0000-000000000000'::uuid) AS thumb_media_id
FROM found_reports fr
JOIN categories cat ON cat.id = fr.category_id
JOIN locations loc ON loc.id = fr.found_location_id
LEFT JOIN LATERAL (
    SELECT m.id FROM media m WHERE m.found_report_id = fr.id AND m.state = 'ready' ORDER BY m.position LIMIT 1
) thumb ON true
WHERE fr.status IN ('reported', 'potential_match')
  AND (cat.parent_id = sqlc.arg(parent_category_id) OR cat.id = sqlc.arg(parent_category_id))
  AND abs(extract(epoch from (fr.found_at - sqlc.arg(found_at)::timestamptz))) <= 72 * 3600
ORDER BY fr.found_at DESC;

-- name: ListFoundMediaPhashes :many
SELECT m.found_report_id, m.phash
FROM media m
WHERE m.found_report_id = ANY(sqlc.arg(report_ids)::uuid[])
  AND m.state = 'ready';

-- name: ListStagedMediaPhashes :many
SELECT m.id, m.phash
FROM media m
WHERE m.id = ANY(sqlc.arg(media_ids)::uuid[])
  AND m.state IN ('staged', 'ready');

