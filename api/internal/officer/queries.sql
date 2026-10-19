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
