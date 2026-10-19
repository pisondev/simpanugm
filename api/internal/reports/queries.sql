-- name: NextCodeSequence :one
INSERT INTO code_sequences (prefix, yymm, last_value) VALUES ($1, $2, 1)
ON CONFLICT (prefix, yymm) DO UPDATE SET last_value = code_sequences.last_value + 1
RETURNING last_value;

-- name: InsertLostReport :exec
INSERT INTO lost_reports (
    id, code, reporter_id, created_by, proxy_name, proxy_phone, category_id, item_name, description,
    private_details, colours, brand, lost_from, lost_until, location_unknown, location_note, status,
    last_activity_at, created_at, updated_at, assist_outcome
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14, $15, $16, 'reported',
    $17, $17, $17, sqlc.narg(assist_outcome)
);

-- name: GetLostReport :one
SELECT sqlc.embed(lr),
       c.slug AS category_slug, c.name AS category_name, c.sensitivity AS category_sensitivity,
       coalesce(p.name, c.name)::text AS category_parent_name,
       u.full_name AS reporter_full_name, u.email AS reporter_email, u.phone AS reporter_phone, u.nim_nip AS reporter_nim_nip,
       (SELECT count(*) FROM matches mt WHERE mt.lost_report_id = lr.id AND mt.status = 'active' AND mt.tier IN ('high', 'medium')
        AND mt.rank_for_lost IS NOT NULL AND mt.owner_dismissed_at IS NULL)::int AS candidate_count
FROM lost_reports lr
JOIN categories c ON c.id = lr.category_id
LEFT JOIN categories p ON p.id = c.parent_id
LEFT JOIN users u ON u.id = lr.reporter_id
WHERE lr.id = sqlc.narg(id)::uuid OR lr.code = sqlc.narg(code)::text;

-- name: UpdateLostReport :one
UPDATE lost_reports
SET category_id = sqlc.arg(category_id),
    item_name = sqlc.arg(item_name),
    description = sqlc.arg(description),
    private_details = sqlc.narg(private_details),
    colours = sqlc.arg(colours),
    brand = sqlc.narg(brand),
    lost_from = sqlc.arg(lost_from),
    lost_until = sqlc.arg(lost_until),
    location_unknown = sqlc.arg(location_unknown),
    location_note = sqlc.narg(location_note),
    last_activity_at = sqlc.arg(now)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING version;

-- name: SetLostStatus :one
UPDATE lost_reports
SET status = sqlc.arg(status),
    close_reason = sqlc.narg(close_reason),
    closed_at = sqlc.narg(closed_at),
    last_activity_at = sqlc.arg(now)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING version;

-- name: DeleteLostPlaces :exec
DELETE FROM lost_report_places WHERE lost_report_id = $1;

-- name: InsertLostPlace :exec
INSERT INTO lost_report_places (lost_report_id, position, location_id, zone_id) VALUES ($1, $2, $3, $4);

-- name: ListLostPlaces :many
SELECT pl.position, pl.location_id, l.name AS location_name, z.id AS zone_id, z.name AS zone_name, (pl.zone_id IS NOT NULL)::bool AS zone_only
FROM lost_report_places pl
LEFT JOIN locations l ON l.id = pl.location_id
JOIN zones z ON z.id = coalesce(pl.zone_id, l.zone_id)
WHERE pl.lost_report_id = $1
ORDER BY pl.position;

-- name: ListLostReports :many
SELECT lr.id, lr.code, lr.status, lr.item_name, c.name AS category_name, lr.lost_until, lr.updated_at, lr.created_at,
       coalesce(u.full_name, lr.proxy_name, '')::text AS reporter_name,
       coalesce(thumb.id, '00000000-0000-0000-0000-000000000000'::uuid) AS thumb_media_id,
       (SELECT count(*) FROM matches mt WHERE mt.lost_report_id = lr.id AND mt.status = 'active' AND mt.tier IN ('high', 'medium')
        AND mt.rank_for_lost IS NOT NULL AND mt.owner_dismissed_at IS NULL)::int AS candidate_count
FROM lost_reports lr
JOIN categories c ON c.id = lr.category_id
LEFT JOIN users u ON u.id = lr.reporter_id
LEFT JOIN LATERAL (
    SELECT m.id FROM media m WHERE m.lost_report_id = lr.id AND m.state = 'ready' ORDER BY m.position LIMIT 1
) thumb ON true
WHERE lr.hidden_at IS NULL
  AND (sqlc.narg(reporter_id)::uuid IS NULL OR lr.reporter_id = sqlc.narg(reporter_id)::uuid)
  AND (cardinality(sqlc.arg(statuses)::text[]) = 0 OR lr.status = ANY(sqlc.arg(statuses)::text[]))
  AND (sqlc.narg(cursor_created)::timestamptz IS NULL
       OR (lr.created_at, lr.id) < (sqlc.narg(cursor_created)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY lr.created_at DESC, lr.id DESC
LIMIT sqlc.arg(row_limit);

-- name: CountOpenLostByReporter :one
SELECT count(*)::int FROM lost_reports WHERE reporter_id = $1 AND status <> 'closed';

-- name: InsertFoundReport :exec
INSERT INTO found_reports (
    id, code, finder_id, created_by, proxy_finder_name, category_id, item_name, description, hidden_details,
    colours, brand, found_at, found_location_id, location_note, custody_status, custody_keep, intended_post_id,
    holding_post_id, storage_slot, received_at, received_by, retention_due_at, status, created_at, updated_at, assist_outcome
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14, $15, $16, $17,
    $18, $19, $20, $21, $22, 'reported', $23, $23, sqlc.narg(assist_outcome)
);

-- name: GetFoundReport :one
SELECT sqlc.embed(fr),
       c.slug AS category_slug, c.name AS category_name, c.sensitivity AS category_sensitivity,
       coalesce(p.name, c.name)::text AS category_parent_name,
       l.name AS location_name, z.id AS zone_id, z.name AS zone_name,
       ip.name AS intended_post_name, hp.name AS holding_post_name,
       u.full_name AS finder_full_name, u.email AS finder_email, u.phone AS finder_phone, u.nim_nip AS finder_nim_nip
FROM found_reports fr
JOIN categories c ON c.id = fr.category_id
LEFT JOIN categories p ON p.id = c.parent_id
JOIN locations l ON l.id = fr.found_location_id
JOIN zones z ON z.id = l.zone_id
LEFT JOIN posts ip ON ip.id = fr.intended_post_id
LEFT JOIN posts hp ON hp.id = fr.holding_post_id
LEFT JOIN users u ON u.id = fr.finder_id
WHERE fr.id = sqlc.narg(id)::uuid OR fr.code = sqlc.narg(code)::text;

-- name: UpdateFoundReport :one
UPDATE found_reports
SET category_id = sqlc.arg(category_id),
    item_name = sqlc.arg(item_name),
    description = sqlc.arg(description),
    hidden_details = sqlc.narg(hidden_details),
    colours = sqlc.arg(colours),
    brand = sqlc.narg(brand),
    found_at = sqlc.arg(found_at),
    found_location_id = sqlc.arg(found_location_id),
    location_note = sqlc.narg(location_note),
    custody_keep = sqlc.arg(custody_keep),
    intended_post_id = sqlc.narg(intended_post_id),
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING version;

-- name: CloseFoundReport :one
UPDATE found_reports
SET status = 'closed',
    close_reason = sqlc.arg(close_reason),
    close_note = sqlc.arg(close_note),
    closed_at = sqlc.arg(now)::timestamptz,
    custody_status = sqlc.arg(custody_status),
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING version;

-- name: ListFoundReports :many
SELECT fr.id, fr.code, fr.status, fr.item_name, c.name AS category_name, fr.found_at, fr.custody_status,
       l.name AS location_name, hp.name AS holding_post_name, fr.updated_at, fr.created_at,
       coalesce(u.full_name, fr.proxy_finder_name, '')::text AS finder_name,
       coalesce(thumb.id, '00000000-0000-0000-0000-000000000000'::uuid) AS thumb_media_id
FROM found_reports fr
JOIN categories c ON c.id = fr.category_id
JOIN locations l ON l.id = fr.found_location_id
LEFT JOIN posts hp ON hp.id = fr.holding_post_id
LEFT JOIN users u ON u.id = fr.finder_id
LEFT JOIN LATERAL (
    SELECT m.id FROM media m WHERE m.found_report_id = fr.id AND m.state = 'ready' ORDER BY m.position LIMIT 1
) thumb ON true
WHERE fr.hidden_at IS NULL
  AND (sqlc.narg(finder_id)::uuid IS NULL OR fr.finder_id = sqlc.narg(finder_id)::uuid)
  AND (cardinality(sqlc.arg(statuses)::text[]) = 0 OR fr.status = ANY(sqlc.arg(statuses)::text[]))
  AND (sqlc.narg(cursor_created)::timestamptz IS NULL
       OR (fr.created_at, fr.id) < (sqlc.narg(cursor_created)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY fr.created_at DESC, fr.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListPublicFoundReports :many
SELECT fr.id, c.id AS category_id, c.name AS category_name, fr.item_name, z.id AS zone_id, z.name AS zone_name,
       fr.found_at, fr.custody_status, hp.id AS holding_post_id, hp.name AS holding_post_name
FROM found_reports fr
JOIN categories c ON c.id = fr.category_id
JOIN locations l ON l.id = fr.found_location_id
JOIN zones z ON z.id = l.zone_id
LEFT JOIN posts hp ON hp.id = fr.holding_post_id
WHERE fr.hidden_at IS NULL
  AND fr.status IN ('reported', 'potential_match', 'under_verification', 'unclaimed')
  AND (sqlc.narg(category_id)::uuid IS NULL OR c.id = sqlc.narg(category_id)::uuid OR c.parent_id = sqlc.narg(category_id)::uuid)
  AND (sqlc.narg(zone_id)::uuid IS NULL OR z.id = sqlc.narg(zone_id)::uuid)
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR fr.found_at >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR fr.found_at < sqlc.narg(to_time)::timestamptz)
  AND (sqlc.narg(cursor_found)::timestamptz IS NULL
       OR (fr.found_at, fr.id) < (sqlc.narg(cursor_found)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY fr.found_at DESC, fr.id DESC
LIMIT sqlc.arg(row_limit);

-- name: CountFoundByFinderSince :one
SELECT count(*)::int FROM found_reports WHERE finder_id = $1 AND created_at > $2;

-- name: InsertStatusHistory :exec
INSERT INTO status_history (entity_type, entity_id, from_status, to_status, transition, reason, actor_id, actor_kind, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ListStatusHistory :many
SELECT h.entity_type, h.from_status, h.to_status, h.transition, h.reason, h.actor_kind, h.created_at, u.full_name AS actor_name
FROM status_history h
LEFT JOIN users u ON u.id = h.actor_id
WHERE h.entity_id = sqlc.arg(entity_id) AND h.entity_type = ANY(sqlc.arg(entity_types)::text[])
ORDER BY h.created_at, h.id;

-- name: ReopenLostReport :one
UPDATE lost_reports
SET status = 'reported',
    close_reason = NULL,
    closed_at = NULL,
    last_activity_at = sqlc.arg(now)::timestamptz,
    stale_reminded_at = NULL,
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status = 'closed'
RETURNING version;

-- name: KeepOpenLostReport :one
UPDATE lost_reports
SET last_activity_at = sqlc.arg(now)::timestamptz,
    stale_reminded_at = NULL,
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status IN ('reported', 'potential_match', 'under_verification')
RETURNING version;

-- name: ConfirmLostReceipt :one
UPDATE lost_reports
SET status = 'closed',
    close_reason = 'returned',
    closed_at = sqlc.arg(now)::timestamptz,
    last_activity_at = sqlc.arg(now)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status = 'returned'
RETURNING version;

-- name: ListStaleLostReports :many
SELECT lr.id, lr.code, lr.reporter_id, lr.item_name
FROM lost_reports lr
WHERE lr.status IN ('reported', 'potential_match')
  AND lr.stale_reminded_at IS NULL
  AND lr.last_activity_at <= sqlc.arg(stale_before)::timestamptz
  AND lr.reporter_id IS NOT NULL;

-- name: MarkLostStaleReminded :exec
UPDATE lost_reports
SET stale_reminded_at = sqlc.arg(now)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = $1;

-- name: ListExpiredLostReports :many
SELECT lr.id, lr.code, lr.reporter_id, lr.version
FROM lost_reports lr
WHERE lr.status IN ('reported', 'potential_match')
  AND lr.stale_reminded_at IS NOT NULL
  AND lr.stale_reminded_at <= sqlc.arg(expired_before)::timestamptz
  AND NOT EXISTS (
      SELECT 1 FROM claims c WHERE c.lost_report_id = lr.id AND c.status IN ('submitted', 'under_review', 'info_requested', 'approved', 'on_hold')
  );

-- name: ExpireLostReport :one
UPDATE lost_reports
SET status = 'closed',
    close_reason = 'expired',
    closed_at = sqlc.arg(now)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status IN ('reported', 'potential_match')
RETURNING version;
