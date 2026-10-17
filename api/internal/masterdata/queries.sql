-- name: ListCategories :many
SELECT * FROM categories WHERE active ORDER BY sort_order, name;

-- name: GetCategory :one
SELECT c.*, coalesce(p.name, c.name)::text AS parent_name
FROM categories c
LEFT JOIN categories p ON p.id = c.parent_id
WHERE c.id = $1;

-- name: ListZones :many
SELECT * FROM zones WHERE active ORDER BY name;

-- name: ListLocations :many
SELECT l.*, z.name AS zone_name
FROM locations l
JOIN zones z ON z.id = l.zone_id
WHERE l.active
  AND (sqlc.narg(zone_id)::uuid IS NULL OR l.zone_id = sqlc.narg(zone_id)::uuid)
  AND (sqlc.narg(q)::text IS NULL
       OR search_key_with_aliases(l.name, l.aliases) LIKE '%' || search_key(sqlc.narg(q)::text) || '%'
       OR word_similarity(search_key(sqlc.narg(q)::text), search_key_with_aliases(l.name, l.aliases)) >= 0.4)
ORDER BY
  CASE WHEN sqlc.narg(q)::text IS NULL THEN 0
       WHEN search_key(l.name) = search_key(sqlc.narg(q)::text)
         OR search_key(sqlc.narg(q)::text) = ANY (SELECT search_key(a) FROM unnest(l.aliases) a) THEN 2
       ELSE 1 END DESC,
  CASE WHEN sqlc.narg(q)::text IS NULL THEN 0
       ELSE word_similarity(search_key(sqlc.narg(q)::text), search_key_with_aliases(l.name, l.aliases)) END DESC,
  CASE WHEN sqlc.narg(q)::text IS NULL THEN 0
       ELSE similarity(search_key(sqlc.narg(q)::text), search_key(l.name)) END DESC,
  l.name
LIMIT sqlc.arg(row_limit);

-- name: GetLocation :one
SELECT l.*, z.name AS zone_name
FROM locations l
JOIN zones z ON z.id = l.zone_id
WHERE l.id = $1;

-- name: GetZone :one
SELECT * FROM zones WHERE id = $1;

-- name: ListPosts :many
SELECT p.*, l.name AS location_name, l.lat, l.lng, l.zone_id, z.name AS zone_name
FROM posts p
JOIN locations l ON l.id = p.location_id
JOIN zones z ON z.id = l.zone_id
WHERE p.active
ORDER BY p.name;

-- name: GetPost :one
SELECT p.*, l.name AS location_name, l.lat, l.lng, l.zone_id, z.name AS zone_name
FROM posts p
JOIN locations l ON l.id = p.location_id
JOIN zones z ON z.id = l.zone_id
WHERE p.id = $1;

-- name: ListColours :many
SELECT * FROM colours ORDER BY sort_order, name;

-- name: UpsertZone :one
INSERT INTO zones (id, slug, name, centroid_lat, centroid_lng)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (slug) DO UPDATE
SET name = excluded.name, centroid_lat = excluded.centroid_lat, centroid_lng = excluded.centroid_lng, updated_at = now()
RETURNING id;

-- name: UpsertLocation :one
INSERT INTO locations (id, zone_id, slug, name, aliases, kind, lat, lng)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (slug) DO UPDATE
SET zone_id = excluded.zone_id, name = excluded.name, aliases = excluded.aliases, kind = excluded.kind,
    lat = excluded.lat, lng = excluded.lng, updated_at = now()
RETURNING id;

-- name: UpsertPost :one
INSERT INTO posts (id, slug, name, location_id, phone, opening_hours)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (slug) DO UPDATE
SET name = excluded.name, location_id = excluded.location_id, phone = excluded.phone,
    opening_hours = excluded.opening_hours, updated_at = now()
RETURNING id;

-- name: UpsertParentCategory :one
INSERT INTO categories (id, slug, name, verification_questions, clip_prompts, sort_order)
VALUES ($1, $2, $3, '[]', '{}', $4)
ON CONFLICT (slug) WHERE parent_id IS NULL DO UPDATE
SET name = excluded.name, sort_order = excluded.sort_order, updated_at = now()
RETURNING id;

-- name: UpsertLeafCategory :one
INSERT INTO categories (id, parent_id, slug, name, sensitivity, retention_days, verification_questions, clip_prompts, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (slug) WHERE parent_id IS NOT NULL DO UPDATE
SET parent_id = excluded.parent_id, name = excluded.name, sensitivity = excluded.sensitivity,
    retention_days = excluded.retention_days, verification_questions = excluded.verification_questions,
    clip_prompts = excluded.clip_prompts, sort_order = excluded.sort_order, updated_at = now()
RETURNING id;

-- name: UpsertCompatibility :exec
INSERT INTO category_compatibility (category_a, category_b, factor)
VALUES ($1, $2, $3)
ON CONFLICT (category_a, category_b) DO UPDATE SET factor = excluded.factor;

-- name: UpsertColour :exec
INSERT INTO colours (name, hex, sort_order) VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET hex = excluded.hex, sort_order = excluded.sort_order;

-- name: UpsertBrand :exec
INSERT INTO brands (name, aliases) VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET aliases = excluded.aliases;

-- name: UpsertSynonym :exec
INSERT INTO synonyms (term, expansion) VALUES ($1, $2)
ON CONFLICT (term) DO UPDATE SET expansion = excluded.expansion;

-- name: InsertSettingIfMissing :exec
INSERT INTO system_settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO NOTHING;
