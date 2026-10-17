-- name: SearchReports :many
SELECT r.type, r.id, r.code, r.item_name, r.category_name, r.status, r.custody_status, r.place_name,
       r.occurred_at, r.person_name, r.relevance, r.created_at
FROM (
    SELECT 'lost'::text AS type, lr.id, lr.code, lr.item_name, c.name AS category_name, lr.status,
           NULL::text AS custody_status,
           coalesce((SELECT coalesce(l.name, z.name)
              FROM lost_report_places pl
              LEFT JOIN locations l ON l.id = pl.location_id
              LEFT JOIN zones z ON z.id = pl.zone_id
             WHERE pl.lost_report_id = lr.id
             ORDER BY pl.position LIMIT 1), '')::text AS place_name,
           lr.lost_until AS occurred_at,
           coalesce(u.full_name, lr.proxy_name, '')::text AS person_name,
           greatest(word_similarity(search_key(sqlc.arg(term)::text), search_key(lr.item_name || ' ' || lr.description)),
                    word_similarity(search_key(sqlc.arg(term)::text), search_key(coalesce(u.full_name, lr.proxy_name, ''))))::float8 AS relevance,
           lr.created_at
    FROM lost_reports lr
    JOIN categories c ON c.id = lr.category_id
    LEFT JOIN users u ON u.id = lr.reporter_id
    WHERE sqlc.arg(include_lost)::bool
      AND lr.hidden_at IS NULL
      AND (sqlc.narg(code)::text IS NULL OR lr.code = sqlc.narg(code)::text)
      AND sqlc.narg(custody)::text IS NULL
      AND sqlc.narg(post_id)::uuid IS NULL
      AND (sqlc.narg(category_id)::uuid IS NULL OR c.id = sqlc.narg(category_id)::uuid OR c.parent_id = sqlc.narg(category_id)::uuid)
      AND (sqlc.narg(zone_id)::uuid IS NULL OR EXISTS (
            SELECT 1 FROM lost_report_places pl LEFT JOIN locations l ON l.id = pl.location_id
             WHERE pl.lost_report_id = lr.id AND coalesce(pl.zone_id, l.zone_id) = sqlc.narg(zone_id)::uuid))
      AND (sqlc.narg(location_id)::uuid IS NULL OR EXISTS (
            SELECT 1 FROM lost_report_places pl WHERE pl.lost_report_id = lr.id AND pl.location_id = sqlc.narg(location_id)::uuid))
      AND (sqlc.narg(from_time)::timestamptz IS NULL OR lr.lost_until >= sqlc.narg(from_time)::timestamptz)
      AND (sqlc.narg(to_time)::timestamptz IS NULL OR lr.lost_from < sqlc.narg(to_time)::timestamptz)
      AND (cardinality(sqlc.arg(statuses)::text[]) = 0 OR lr.status = ANY(sqlc.arg(statuses)::text[]))
    UNION ALL
    SELECT 'found'::text, fr.id, fr.code, fr.item_name, c.name, fr.status,
           fr.custody_status,
           l.name,
           fr.found_at,
           coalesce(u.full_name, fr.proxy_finder_name, '')::text,
           greatest(word_similarity(search_key(sqlc.arg(term)::text), search_key(fr.item_name || ' ' || fr.description)),
                    word_similarity(search_key(sqlc.arg(term)::text), search_key(coalesce(u.full_name, fr.proxy_finder_name, ''))))::float8,
           fr.created_at
    FROM found_reports fr
    JOIN categories c ON c.id = fr.category_id
    JOIN locations l ON l.id = fr.found_location_id
    LEFT JOIN users u ON u.id = fr.finder_id
    WHERE sqlc.arg(include_found)::bool
      AND fr.hidden_at IS NULL
      AND (sqlc.narg(code)::text IS NULL OR fr.code = sqlc.narg(code)::text)
      AND (sqlc.narg(custody)::text IS NULL OR fr.custody_status = sqlc.narg(custody)::text)
      AND (sqlc.narg(post_id)::uuid IS NULL OR fr.holding_post_id = sqlc.narg(post_id)::uuid OR fr.intended_post_id = sqlc.narg(post_id)::uuid)
      AND (sqlc.narg(category_id)::uuid IS NULL OR c.id = sqlc.narg(category_id)::uuid OR c.parent_id = sqlc.narg(category_id)::uuid)
      AND (sqlc.narg(zone_id)::uuid IS NULL OR l.zone_id = sqlc.narg(zone_id)::uuid)
      AND (sqlc.narg(location_id)::uuid IS NULL OR fr.found_location_id = sqlc.narg(location_id)::uuid)
      AND (sqlc.narg(from_time)::timestamptz IS NULL OR fr.found_at >= sqlc.narg(from_time)::timestamptz)
      AND (sqlc.narg(to_time)::timestamptz IS NULL OR fr.found_at < sqlc.narg(to_time)::timestamptz)
      AND (cardinality(sqlc.arg(statuses)::text[]) = 0 OR fr.status = ANY(sqlc.arg(statuses)::text[]))
) r
WHERE sqlc.arg(term)::text = '' OR r.relevance >= 0.3
ORDER BY CASE WHEN sqlc.arg(term)::text = '' THEN 0 ELSE r.relevance END DESC, r.created_at DESC, r.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);
