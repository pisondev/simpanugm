-- name: ActiveVersion :one
SELECT id, name, identity, config FROM model_versions WHERE component = $1 AND status = 'active';

-- name: VersionByName :one
SELECT id, component, name, status FROM model_versions WHERE name = $1;

-- name: InsertVersion :exec
INSERT INTO model_versions (id, component, name, identity, config, config_sha256, status, activated_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (name) DO NOTHING;

-- name: LostForMatching :many
SELECT lr.id, lr.code, lr.status, lr.hidden_at, lr.category_id, c.parent_id, c.name AS category_name, pc.name AS parent_name,
       c.sensitivity, lr.item_name, lr.description, lr.private_details, lr.colours, lr.brand,
       lr.lost_from, lr.lost_until, lr.location_unknown, lr.updated_at, lr.reporter_id,
       coalesce(u.full_name, lr.proxy_name, '')::text AS owner_name,
       EXISTS (SELECT 1 FROM media m WHERE m.lost_report_id = lr.id AND m.state = 'ready')::bool AS has_photos
FROM lost_reports lr
JOIN categories c ON c.id = lr.category_id
JOIN categories pc ON pc.id = c.parent_id
LEFT JOIN users u ON u.id = lr.reporter_id
WHERE lr.id = ANY(sqlc.arg(ids)::uuid[]);

-- name: LostPlacesForMatching :many
SELECT p.lost_report_id, p.position, (p.zone_id IS NOT NULL)::bool AS zone_only,
       coalesce(l.lat, z.centroid_lat)::float8 AS lat, coalesce(l.lng, z.centroid_lng)::float8 AS lng,
       coalesce(l.name, z.name)::text AS name
FROM lost_report_places p
LEFT JOIN locations l ON l.id = p.location_id
LEFT JOIN zones z ON z.id = p.zone_id
WHERE p.lost_report_id = ANY(sqlc.arg(ids)::uuid[])
ORDER BY p.lost_report_id, p.position;

-- name: FoundForMatching :many
SELECT fr.id, fr.code, fr.status, fr.hidden_at, fr.category_id, c.parent_id, c.name AS category_name, pc.name AS parent_name,
       c.sensitivity, fr.item_name, fr.description, fr.hidden_details, fr.colours, fr.brand, fr.found_at, fr.updated_at,
       l.lat, l.lng, l.name AS location_name, fr.finder_id, fr.holding_post_id, fr.intended_post_id,
       coalesce((SELECT string_agg(ia.visible_text, ' ') FROM image_attributes ia JOIN media m ON m.id = ia.media_id
                 WHERE m.found_report_id = fr.id AND m.state = 'ready'), '')::text AS visible_text,
       EXISTS (SELECT 1 FROM media m WHERE m.found_report_id = fr.id AND m.state = 'ready')::bool AS has_photos
FROM found_reports fr
JOIN categories c ON c.id = fr.category_id
JOIN categories pc ON pc.id = c.parent_id
JOIN locations l ON l.id = fr.found_location_id
WHERE fr.id = ANY(sqlc.arg(ids)::uuid[]);

-- name: FoundPoolStructured :many
SELECT fr.id FROM found_reports fr
WHERE fr.hidden_at IS NULL AND fr.status IN ('reported', 'potential_match', 'under_verification', 'unclaimed')
  AND fr.found_at BETWEEN sqlc.arg(window_start)::timestamptz AND sqlc.arg(window_end)::timestamptz
  AND fr.category_id = sqlc.arg(category_id)
ORDER BY abs(extract(epoch FROM fr.found_at - sqlc.arg(anchor)::timestamptz)), fr.id
LIMIT sqlc.arg(k)::int;

-- name: LostPoolStructured :many
SELECT lr.id FROM lost_reports lr
WHERE lr.hidden_at IS NULL AND lr.status IN ('reported', 'potential_match', 'under_verification')
  AND lr.lost_from - make_interval(hours => sqlc.arg(before_h)::int) <= sqlc.arg(found_at)::timestamptz
  AND sqlc.arg(found_at)::timestamptz <= lr.lost_until + make_interval(days => sqlc.arg(after_d)::int)
  AND lr.category_id = sqlc.arg(category_id)
ORDER BY abs(extract(epoch FROM lr.lost_until - sqlc.arg(found_at)::timestamptz)), lr.id
LIMIT sqlc.arg(k)::int;

-- name: FoundNearestImages :many
-- the inner ORDER BY with LIMIT walks the HNSW index, the outer query keeps one row per report
SELECT nearest.report_id FROM (
    SELECT ie.report_id, ie.embedding <=> sqlc.arg(query)::text::vector AS distance
    FROM image_embeddings ie
    JOIN found_reports fr ON fr.id = ie.report_id
    WHERE ie.report_type = 'found' AND ie.model_version_id = sqlc.arg(version_id)
      AND fr.hidden_at IS NULL AND fr.status IN ('reported', 'potential_match', 'under_verification', 'unclaimed')
      AND fr.found_at BETWEEN sqlc.arg(window_start)::timestamptz AND sqlc.arg(window_end)::timestamptz
    ORDER BY distance
    LIMIT sqlc.arg(k)::int * 3
) nearest
GROUP BY nearest.report_id
ORDER BY min(nearest.distance)
LIMIT sqlc.arg(k)::int;

-- name: FoundNearestE5 :many
SELECT t.report_id FROM text_embeddings_e5 t
JOIN found_reports fr ON fr.id = t.report_id
WHERE t.report_type = 'found' AND t.model_version_id = sqlc.arg(version_id)
  AND fr.hidden_at IS NULL AND fr.status IN ('reported', 'potential_match', 'under_verification', 'unclaimed')
  AND fr.found_at BETWEEN sqlc.arg(window_start)::timestamptz AND sqlc.arg(window_end)::timestamptz
ORDER BY t.embedding <=> sqlc.arg(query)::text::vector
LIMIT sqlc.arg(k)::int;

-- name: LostNearestImages :many
SELECT nearest.report_id FROM (
    SELECT ie.report_id, ie.embedding <=> sqlc.arg(query)::text::vector AS distance
    FROM image_embeddings ie
    JOIN lost_reports lr ON lr.id = ie.report_id
    WHERE ie.report_type = 'lost' AND ie.model_version_id = sqlc.arg(version_id)
      AND lr.hidden_at IS NULL AND lr.status IN ('reported', 'potential_match', 'under_verification')
      AND lr.lost_from - make_interval(hours => sqlc.arg(before_h)::int) <= sqlc.arg(found_at)::timestamptz
      AND sqlc.arg(found_at)::timestamptz <= lr.lost_until + make_interval(days => sqlc.arg(after_d)::int)
    ORDER BY distance
    LIMIT sqlc.arg(k)::int * 3
) nearest
GROUP BY nearest.report_id
ORDER BY min(nearest.distance)
LIMIT sqlc.arg(k)::int;

-- name: LostNearestClipText :many
SELECT t.report_id FROM text_embeddings_clip t
JOIN lost_reports lr ON lr.id = t.report_id
WHERE t.report_type = 'lost' AND t.model_version_id = sqlc.arg(version_id)
  AND lr.hidden_at IS NULL AND lr.status IN ('reported', 'potential_match', 'under_verification')
  AND lr.lost_from - make_interval(hours => sqlc.arg(before_h)::int) <= sqlc.arg(found_at)::timestamptz
  AND sqlc.arg(found_at)::timestamptz <= lr.lost_until + make_interval(days => sqlc.arg(after_d)::int)
ORDER BY t.embedding <=> sqlc.arg(query)::text::vector
LIMIT sqlc.arg(k)::int;

-- name: LostNearestE5 :many
SELECT t.report_id FROM text_embeddings_e5 t
JOIN lost_reports lr ON lr.id = t.report_id
WHERE t.report_type = 'lost' AND t.model_version_id = sqlc.arg(version_id)
  AND lr.hidden_at IS NULL AND lr.status IN ('reported', 'potential_match', 'under_verification')
  AND lr.lost_from - make_interval(hours => sqlc.arg(before_h)::int) <= sqlc.arg(found_at)::timestamptz
  AND sqlc.arg(found_at)::timestamptz <= lr.lost_until + make_interval(days => sqlc.arg(after_d)::int)
ORDER BY t.embedding <=> sqlc.arg(query)::text::vector
LIMIT sqlc.arg(k)::int;

-- name: ImageVectors :many
SELECT media_id, embedding::text AS embedding FROM image_embeddings
WHERE report_type = $1 AND report_id = $2 AND model_version_id = $3;

-- name: ClipTextVector :one
SELECT embedding::text AS embedding FROM text_embeddings_clip WHERE report_type = $1 AND report_id = $2 AND model_version_id = $3;

-- name: E5Vector :one
SELECT embedding::text AS embedding FROM text_embeddings_e5 WHERE report_type = $1 AND report_id = $2 AND model_version_id = $3;

-- name: ReportsWithImageEmbeddings :many
SELECT DISTINCT report_id FROM image_embeddings
WHERE report_type = sqlc.arg(report_type) AND report_id = ANY(sqlc.arg(ids)::uuid[]) AND model_version_id = sqlc.arg(version_id);

-- name: ImageCosines :many
SELECT b.report_id AS candidate_id, max(1 - (a.embedding <=> b.embedding))::float8 AS cosine
FROM image_embeddings a
JOIN image_embeddings b ON b.report_type = sqlc.arg(candidate_type) AND b.report_id = ANY(sqlc.arg(candidate_ids)::uuid[])
                        AND b.model_version_id = a.model_version_id
WHERE a.report_type = sqlc.arg(source_type) AND a.report_id = sqlc.arg(source_id) AND a.model_version_id = sqlc.arg(version_id)
GROUP BY b.report_id;

-- name: CrossCosines :many
SELECT t.report_id AS text_report_id, ie.report_id AS image_report_id, max(1 - (t.embedding <=> ie.embedding))::float8 AS cosine
FROM text_embeddings_clip t
JOIN image_embeddings ie ON ie.report_type = sqlc.arg(image_type) AND ie.report_id = ANY(sqlc.arg(image_ids)::uuid[])
                         AND ie.model_version_id = sqlc.arg(image_version_id)
WHERE t.report_type = sqlc.arg(text_type) AND t.report_id = ANY(sqlc.arg(text_ids)::uuid[]) AND t.model_version_id = sqlc.arg(text_version_id)
GROUP BY t.report_id, ie.report_id;

-- name: E5Cosines :many
SELECT b.report_id AS candidate_id, (1 - (a.embedding <=> b.embedding))::float8 AS cosine
FROM text_embeddings_e5 a
JOIN text_embeddings_e5 b ON b.report_type = sqlc.arg(candidate_type) AND b.report_id = ANY(sqlc.arg(candidate_ids)::uuid[])
                          AND b.model_version_id = a.model_version_id
WHERE a.report_type = sqlc.arg(source_type) AND a.report_id = sqlc.arg(source_id) AND a.model_version_id = sqlc.arg(version_id);

-- name: CompatibilityFor :many
SELECT category_a, category_b, factor::float8 AS factor FROM category_compatibility WHERE (category_a = $1 OR category_b = $1) AND active;

-- name: PairForUpdate :one
SELECT id, status, tier FROM matches WHERE lost_report_id = $1 AND found_report_id = $2 FOR UPDATE;

-- name: InsertMatch :exec
INSERT INTO matches (id, lost_report_id, found_report_id, visual_mode, s_visual, s_semantic, s_attr, s_text, s_location, s_time,
                     category_factor, overall, tier, flags, explanation, status, fusion_version_id, component_versions,
                     first_scored_at, scored_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, 'superseded', $16, $17, $18, $18)
ON CONFLICT (lost_report_id, found_report_id) DO NOTHING;

-- name: UpdateMatchScores :exec
UPDATE matches
SET visual_mode = $2, s_visual = $3, s_semantic = $4, s_attr = $5, s_text = $6, s_location = $7, s_time = $8,
    category_factor = $9, overall = $10, tier = $11, flags = $12, explanation = $13, fusion_version_id = $14,
    component_versions = $15, scored_at = $16, version = version + 1
WHERE id = $1 AND status IN ('active', 'superseded');

-- ranks count only pairs whose counterpart is still open, a closed or hidden counterpart loses its rank here
-- name: RerankLost :exec
WITH ranked AS (
    SELECT mm.id, row_number() OVER (ORDER BY mm.overall DESC, mm.first_scored_at, mm.id) AS r
    FROM matches mm JOIN found_reports f ON f.id = mm.found_report_id
    WHERE mm.lost_report_id = sqlc.arg(report_id) AND mm.status IN ('active', 'superseded') AND mm.tier IS NOT NULL
      AND mm.owner_dismissed_at IS NULL AND f.hidden_at IS NULL AND f.status IN ('reported', 'potential_match', 'under_verification', 'unclaimed')
)
UPDATE matches m
SET rank_for_lost = (SELECT CASE WHEN ranked.r <= sqlc.arg(top_k)::int THEN ranked.r::smallint END FROM ranked WHERE ranked.id = m.id)
WHERE m.lost_report_id = sqlc.arg(report_id) AND m.status IN ('active', 'superseded');

-- name: RerankFound :exec
WITH ranked AS (
    SELECT mm.id, row_number() OVER (ORDER BY mm.overall DESC, mm.first_scored_at, mm.id) AS r
    FROM matches mm JOIN lost_reports l ON l.id = mm.lost_report_id
    WHERE mm.found_report_id = sqlc.arg(report_id) AND mm.status IN ('active', 'superseded') AND mm.tier IS NOT NULL
      AND l.hidden_at IS NULL AND l.status IN ('reported', 'potential_match', 'under_verification')
)
UPDATE matches m
SET rank_for_found = (SELECT CASE WHEN ranked.r <= sqlc.arg(top_k)::int THEN ranked.r::smallint END FROM ranked WHERE ranked.id = m.id)
WHERE m.found_report_id = sqlc.arg(report_id) AND m.status IN ('active', 'superseded');

-- name: ClearRanksBelowFloor :exec
UPDATE matches SET rank_for_lost = NULL, rank_for_found = NULL
WHERE (lost_report_id = ANY(sqlc.arg(lost_ids)::uuid[]) OR found_report_id = ANY(sqlc.arg(found_ids)::uuid[]))
  AND status IN ('active', 'superseded') AND tier IS NULL;

-- name: SettleStatuses :many
UPDATE matches
SET status = CASE WHEN rank_for_lost IS NOT NULL OR rank_for_found IS NOT NULL THEN 'active' ELSE 'superseded' END
WHERE (lost_report_id = ANY(sqlc.arg(lost_ids)::uuid[]) OR found_report_id = ANY(sqlc.arg(found_ids)::uuid[]))
  AND status IN ('active', 'superseded')
  AND status <> CASE WHEN rank_for_lost IS NOT NULL OR rank_for_found IS NOT NULL THEN 'active' ELSE 'superseded' END
RETURNING id, lost_report_id, found_report_id, status;

-- name: SupersedeForLost :many
UPDATE matches SET status = 'superseded', rank_for_lost = NULL, rank_for_found = NULL
WHERE lost_report_id = $1 AND status = 'active'
RETURNING found_report_id;

-- name: SupersedeForFound :many
UPDATE matches SET status = 'superseded', rank_for_lost = NULL, rank_for_found = NULL
WHERE found_report_id = $1 AND status = 'active'
RETURNING lost_report_id;

-- what the owner and the post see, the same rule decides L2, L3, F2 and F3
-- name: VisibleCandidatesOfLost :one
SELECT count(*)::int FROM matches
WHERE lost_report_id = $1 AND status = 'active' AND tier IN ('high', 'medium') AND rank_for_lost IS NOT NULL AND owner_dismissed_at IS NULL;

-- name: VisibleCandidatesOfFound :one
SELECT count(*)::int FROM matches
WHERE found_report_id = $1 AND status = 'active' AND tier IN ('high', 'medium') AND rank_for_found IS NOT NULL;

-- name: LostStatusForUpdate :one
SELECT status FROM lost_reports WHERE id = $1 FOR UPDATE;

-- name: FoundStatusForUpdate :one
SELECT status FROM found_reports WHERE id = $1 FOR UPDATE;

-- name: SetLostStatusBySystem :exec
UPDATE lost_reports SET status = $2 WHERE id = $1;

-- name: SetFoundStatusBySystem :exec
UPDATE found_reports SET status = $2 WHERE id = $1;

-- name: HighMatchesOfFound :many
SELECT m.id, lr.code AS lost_code FROM matches m
JOIN lost_reports lr ON lr.id = m.lost_report_id
WHERE m.found_report_id = $1 AND m.status = 'active' AND m.tier = 'high';

-- candidates the owner has not been told about yet, or has only been told about as medium
-- name: UnnotifiedCandidates :many
SELECT id, tier::text AS tier FROM matches
WHERE lost_report_id = $1 AND status = 'active' AND tier IN ('high', 'medium') AND rank_for_lost IS NOT NULL
  AND owner_dismissed_at IS NULL AND (notified_tier IS NULL OR (notified_tier = 'medium' AND tier = 'high'));

-- name: MarkNotified :exec
UPDATE matches SET notified_tier = tier WHERE id = ANY(sqlc.arg(ids)::uuid[]) AND tier IN ('high', 'medium');

-- changes whenever an embedding of the report is stored, so a scoring that raced an embedding job runs again
-- name: EmbeddingFingerprint :one
SELECT coalesce(md5(string_agg(s.x, ',' ORDER BY s.x)), '')::text AS fingerprint FROM (
    SELECT ie.media_id::text || ie.model_version_id::text AS x FROM image_embeddings ie
    WHERE ie.report_type = sqlc.arg(report_type)::text AND ie.report_id = sqlc.arg(report_id)::uuid
    UNION ALL
    SELECT 'e5' || te.model_version_id::text || encode(te.text_sha256, 'hex') FROM text_embeddings_e5 te
    WHERE te.report_type = sqlc.arg(report_type)::text AND te.report_id = sqlc.arg(report_id)::uuid
    UNION ALL
    SELECT 'clip' || tc.model_version_id::text || encode(tc.text_sha256, 'hex') FROM text_embeddings_clip tc
    WHERE tc.report_type = sqlc.arg(report_type)::text AND tc.report_id = sqlc.arg(report_id)::uuid
) s;

-- name: LastMatchNotification :one
SELECT coalesce(max(created_at), 'epoch'::timestamptz)::timestamptz AS at FROM notifications
WHERE recipient_id = $1 AND entity_id = $2 AND event IN ('match.high_for_owner', 'match.medium_digest_for_owner');

-- name: NotificationExists :one
SELECT EXISTS (SELECT 1 FROM notifications WHERE recipient_id = $1 AND event = $2 AND entity_id = $3)::bool;

-- name: OpenReportsForSweep :many
SELECT 'lost'::text AS report_type, id FROM lost_reports WHERE status IN ('reported', 'potential_match', 'under_verification') AND hidden_at IS NULL
UNION ALL
SELECT 'found'::text, id FROM found_reports WHERE status IN ('reported', 'potential_match', 'under_verification', 'unclaimed') AND hidden_at IS NULL;

-- name: ReportMediaForEmbedding :many
SELECT m.id, m.storage_keys FROM media m
WHERE m.state = 'ready' AND ((sqlc.arg(report_type)::text = 'lost' AND m.lost_report_id = sqlc.arg(report_id))
                             OR (sqlc.arg(report_type)::text = 'found' AND m.found_report_id = sqlc.arg(report_id)))
ORDER BY m.position;

-- name: MediaMissingEmbedding :many
SELECT m.id FROM media m
WHERE m.state = 'ready' AND ((sqlc.arg(report_type)::text = 'lost' AND m.lost_report_id = sqlc.arg(report_id))
                             OR (sqlc.arg(report_type)::text = 'found' AND m.found_report_id = sqlc.arg(report_id)))
  AND NOT EXISTS (SELECT 1 FROM image_embeddings ie WHERE ie.media_id = m.id AND ie.model_version_id = sqlc.arg(version_id));

-- name: GetMediaForEmbedding :one
SELECT id, state, storage_keys, lost_report_id, found_report_id FROM media WHERE id = $1;

-- name: UpsertImageEmbedding :exec
INSERT INTO image_embeddings (media_id, model_version_id, report_type, report_id, embedding, created_at)
VALUES ($1, $2, $3, $4, sqlc.arg(embedding)::text::vector, $5)
ON CONFLICT (media_id, model_version_id) DO UPDATE SET embedding = excluded.embedding;

-- name: TextHashE5 :one
SELECT text_sha256 FROM text_embeddings_e5 WHERE report_type = $1 AND report_id = $2 AND model_version_id = $3;

-- name: UpsertTextE5 :exec
INSERT INTO text_embeddings_e5 (report_type, report_id, model_version_id, embedding, text_sha256, created_at, updated_at)
VALUES ($1, $2, $3, sqlc.arg(embedding)::text::vector, $4, $5, $5)
ON CONFLICT (report_type, report_id, model_version_id) DO UPDATE SET embedding = excluded.embedding, text_sha256 = excluded.text_sha256, updated_at = excluded.updated_at;

-- name: UpsertTextClip :exec
INSERT INTO text_embeddings_clip (report_type, report_id, model_version_id, embedding, text_sha256, created_at, updated_at)
VALUES ($1, $2, $3, sqlc.arg(embedding)::text::vector, $4, $5, $5)
ON CONFLICT (report_type, report_id, model_version_id) DO UPDATE SET embedding = excluded.embedding, text_sha256 = excluded.text_sha256, updated_at = excluded.updated_at;

-- name: UpsertImageAttributes :exec
INSERT INTO image_attributes (media_id, versions, category_scores, colours, visible_text, brands, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (media_id) DO UPDATE SET versions = excluded.versions, category_scores = excluded.category_scores,
    colours = excluded.colours, visible_text = excluded.visible_text, brands = excluded.brands, created_at = excluded.created_at;

-- name: ListSynonyms :many
SELECT term, expansion FROM synonyms;

-- name: CategoryPromptSet :many
SELECT slug, clip_prompts FROM categories WHERE parent_id IS NOT NULL AND active ORDER BY slug;

-- name: ExistingFoundOfLost :many
SELECT found_report_id FROM matches WHERE lost_report_id = $1 AND status IN ('active', 'superseded');

-- name: ExistingLostOfFound :many
SELECT lost_report_id FROM matches WHERE found_report_id = $1 AND status IN ('active', 'superseded');

-- name: MatchStates :many
SELECT id, lost_report_id, found_report_id, status, tier, rank_for_lost, rank_for_found FROM matches WHERE id = ANY(sqlc.arg(ids)::uuid[]);

-- name: LostRefForMatches :one
SELECT id, code, reporter_id, status, hidden_at FROM lost_reports
WHERE id = sqlc.narg(id)::uuid OR code = sqlc.narg(code)::text;

-- name: FoundRefForMatches :one
SELECT id, code, finder_id, status, hidden_at FROM found_reports
WHERE id = sqlc.narg(id)::uuid OR code = sqlc.narg(code)::text;

-- one shape for every projection, the service filters by role
-- name: MatchViews :many
SELECT m.id, m.lost_report_id, m.found_report_id, m.visual_mode, m.s_visual, m.s_semantic, m.s_attr, m.s_text, m.s_location,
       m.s_time, m.category_factor, m.overall, m.tier, m.rank_for_lost, m.rank_for_found, m.flags, m.explanation, m.status,
       m.owner_dismissed_at, m.component_versions, m.first_scored_at, m.scored_at,
       (m.pinned_at IS NOT NULL)::bool AS pinned, m.pinned_at,
       fr.code AS found_code, fr.item_name AS found_item_name, fr.description AS found_description, fr.colours AS found_colours,
       fr.brand AS found_brand, fr.found_at, fr.custody_status, fr.finder_id, fr.hidden_details,
       fc.id AS found_category_id, fc.slug AS found_category_slug, fc.name AS found_category_name,
       coalesce(fp.name, fc.name)::text AS found_category_parent, fc.sensitivity AS found_sensitivity,
       l.id AS location_id, l.name AS location_name, z.id AS zone_id, z.name AS zone_name,
       hp.id AS holding_post_id, hp.name AS holding_post_name,
       lr.code AS lost_code, lr.item_name AS lost_item_name, lr.reporter_id, lr.lost_from, lr.lost_until,
       lc.id AS lost_category_id, lc.slug AS lost_category_slug, lc.name AS lost_category_name,
       coalesce(lp.name, lc.name)::text AS lost_category_parent, lc.sensitivity AS lost_sensitivity,
       coalesce(u.full_name, lr.proxy_name, '')::text AS owner_name
FROM matches m
JOIN found_reports fr ON fr.id = m.found_report_id
JOIN categories fc ON fc.id = fr.category_id
LEFT JOIN categories fp ON fp.id = fc.parent_id
JOIN locations l ON l.id = fr.found_location_id
JOIN zones z ON z.id = l.zone_id
LEFT JOIN posts hp ON hp.id = fr.holding_post_id
JOIN lost_reports lr ON lr.id = m.lost_report_id
JOIN categories lc ON lc.id = lr.category_id
LEFT JOIN categories lp ON lp.id = lc.parent_id
LEFT JOIN users u ON u.id = lr.reporter_id
WHERE (sqlc.narg(match_id)::uuid IS NULL OR m.id = sqlc.narg(match_id)::uuid)
  AND (sqlc.narg(lost_id)::uuid IS NULL OR m.lost_report_id = sqlc.narg(lost_id)::uuid)
  AND (sqlc.narg(found_id)::uuid IS NULL OR m.found_report_id = sqlc.narg(found_id)::uuid)
  AND m.status = ANY(sqlc.arg(statuses)::text[])
ORDER BY (m.pinned_at IS NOT NULL) DESC, m.overall DESC, m.first_scored_at, m.id
LIMIT 50;

-- name: CandidatePhotos :many
SELECT id, found_report_id, position FROM media
WHERE found_report_id = ANY(sqlc.arg(ids)::uuid[]) AND state = 'ready'
ORDER BY found_report_id, position;

-- name: MatchForAction :one
SELECT m.id, m.lost_report_id, m.found_report_id, m.status, m.tier, m.rank_for_lost, m.owner_dismissed_at, lr.reporter_id
FROM matches m JOIN lost_reports lr ON lr.id = m.lost_report_id
WHERE m.id = $1
FOR UPDATE OF m;

-- name: InsertMatchFeedback :exec
INSERT INTO match_feedback (id, match_id, actor_id, actor_role, verdict, reason, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: OwnerDismissMatch :exec
UPDATE matches SET owner_dismissed_at = $2, version = version + 1 WHERE id = $1 AND owner_dismissed_at IS NULL;

-- M4
-- name: OfficerDismissMatch :exec
UPDATE matches SET status = 'dismissed', rank_for_lost = NULL, rank_for_found = NULL, version = version + 1
WHERE id = $1 AND status = 'active';

-- name: PinMatch :one
UPDATE matches
SET pinned_by = sqlc.narg(pinned_by)::uuid,
    pinned_at = sqlc.narg(pinned_at)::timestamptz,
    version = version + 1
WHERE id = $1
RETURNING id, (pinned_at IS NOT NULL)::bool AS pinned, pinned_at;

-- AI-05: what the finder may learn, the item name and category of the lost report and nothing else (BR-VIS 5.1)
-- name: FinderHint :many
SELECT lr.item_name, c.name AS category_name FROM matches m
JOIN lost_reports lr ON lr.id = m.lost_report_id
JOIN categories c ON c.id = lr.category_id
WHERE m.found_report_id = $1 AND m.status = 'active' AND m.tier = 'high' AND m.rank_for_found IS NOT NULL
ORDER BY m.overall DESC, m.id
LIMIT 3;
