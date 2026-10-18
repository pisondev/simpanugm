-- name: ListCategories :many
SELECT * FROM categories WHERE active ORDER BY sort_order, name;

-- name: GetCategory :one
SELECT c.*, coalesce(p.name, c.name)::text AS parent_name, coalesce(p.active, true)::bool AS parent_active
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
WHERE zones.edited_at IS NULL
RETURNING id;

-- name: UpsertLocation :one
INSERT INTO locations (id, zone_id, slug, name, aliases, kind, lat, lng)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (slug) DO UPDATE
SET zone_id = excluded.zone_id, name = excluded.name, aliases = excluded.aliases, kind = excluded.kind,
    lat = excluded.lat, lng = excluded.lng, updated_at = now()
WHERE locations.edited_at IS NULL
RETURNING id;

-- name: UpsertPost :one
INSERT INTO posts (id, slug, name, location_id, phone, opening_hours)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (slug) DO UPDATE
SET name = excluded.name, location_id = excluded.location_id, phone = excluded.phone,
    opening_hours = excluded.opening_hours, updated_at = now()
WHERE posts.edited_at IS NULL
RETURNING id;

-- name: UpsertParentCategory :one
INSERT INTO categories (id, slug, name, verification_questions, clip_prompts, sort_order)
VALUES ($1, $2, $3, '[]', '{}', $4)
ON CONFLICT (slug) WHERE parent_id IS NULL DO UPDATE
SET name = excluded.name, sort_order = excluded.sort_order, updated_at = now()
WHERE categories.edited_at IS NULL
RETURNING id;

-- name: UpsertLeafCategory :one
INSERT INTO categories (id, parent_id, slug, name, sensitivity, retention_days, verification_questions, clip_prompts, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (slug) WHERE parent_id IS NOT NULL DO UPDATE
SET parent_id = excluded.parent_id, name = excluded.name, sensitivity = excluded.sensitivity,
    retention_days = excluded.retention_days, verification_questions = excluded.verification_questions,
    clip_prompts = excluded.clip_prompts, sort_order = excluded.sort_order, updated_at = now()
WHERE categories.edited_at IS NULL
RETURNING id;

-- name: UpsertCompatibility :exec
INSERT INTO category_compatibility (category_a, category_b, factor)
VALUES ($1, $2, $3)
ON CONFLICT (category_a, category_b) DO UPDATE SET factor = excluded.factor
WHERE category_compatibility.edited_at IS NULL;

-- an upsert that left an administrator's row alone returns nothing, the seed then looks the id up
-- name: ZoneIDBySlug :one
SELECT id FROM zones WHERE slug = $1;

-- name: LocationIDBySlug :one
SELECT id FROM locations WHERE slug = $1;

-- name: ParentCategoryIDBySlug :one
SELECT id FROM categories WHERE slug = $1 AND parent_id IS NULL;

-- name: LeafCategoryIDBySlug :one
SELECT id FROM categories WHERE slug = $1 AND parent_id IS NOT NULL;

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

-- US-07: administrators see every row, active or not

-- name: AdminListCategories :many
SELECT c.*, coalesce(p.name, '')::text AS parent_name
FROM categories c
LEFT JOIN categories p ON p.id = c.parent_id
ORDER BY coalesce(p.sort_order, c.sort_order), coalesce(p.name, c.name), c.parent_id NULLS FIRST, c.sort_order, c.name;

-- name: InsertCategory :exec
INSERT INTO categories (id, parent_id, slug, name, sensitivity, retention_days, verification_questions, clip_prompts, sort_order,
                        active, created_at, updated_at, edited_at)
VALUES (sqlc.arg(id), sqlc.narg(parent_id), sqlc.arg(slug), sqlc.arg(name), sqlc.narg(sensitivity), sqlc.narg(retention_days),
        sqlc.arg(verification_questions), sqlc.arg(clip_prompts), sqlc.arg(sort_order), true, sqlc.arg(now), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateCategory :exec
UPDATE categories
SET name = sqlc.arg(name), sensitivity = sqlc.narg(sensitivity), retention_days = sqlc.narg(retention_days),
    verification_questions = sqlc.arg(verification_questions), clip_prompts = sqlc.arg(clip_prompts),
    sort_order = sqlc.arg(sort_order), active = sqlc.arg(active), updated_at = sqlc.arg(now), edited_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: CategoryForUpdate :one
SELECT * FROM categories WHERE id = $1 FOR UPDATE;

-- name: AdminListZones :many
SELECT * FROM zones ORDER BY name;

-- name: InsertZone :exec
INSERT INTO zones (id, slug, name, centroid_lat, centroid_lng, active, created_at, updated_at, edited_at)
VALUES (sqlc.arg(id), sqlc.arg(slug), sqlc.arg(name), sqlc.arg(centroid_lat), sqlc.arg(centroid_lng), true, sqlc.arg(now), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateZone :exec
UPDATE zones
SET name = sqlc.arg(name), centroid_lat = sqlc.arg(centroid_lat), centroid_lng = sqlc.arg(centroid_lng), active = sqlc.arg(active),
    updated_at = sqlc.arg(now), edited_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ZoneForUpdate :one
SELECT * FROM zones WHERE id = $1 FOR UPDATE;

-- name: AdminListLocations :many
SELECT l.*, z.name AS zone_name
FROM locations l
JOIN zones z ON z.id = l.zone_id
WHERE (sqlc.narg(zone_id)::uuid IS NULL OR l.zone_id = sqlc.narg(zone_id)::uuid)
  AND (sqlc.narg(q)::text IS NULL OR search_key_with_aliases(l.name, l.aliases) LIKE '%' || search_key(sqlc.narg(q)::text) || '%')
ORDER BY z.name, l.name
LIMIT 500;

-- name: InsertLocation :exec
INSERT INTO locations (id, zone_id, slug, name, aliases, kind, lat, lng, active, created_at, updated_at, edited_at)
VALUES (sqlc.arg(id), sqlc.arg(zone_id), sqlc.arg(slug), sqlc.arg(name), sqlc.arg(aliases), sqlc.arg(kind), sqlc.arg(lat), sqlc.arg(lng),
        true, sqlc.arg(now), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateLocation :exec
UPDATE locations
SET zone_id = sqlc.arg(zone_id), name = sqlc.arg(name), aliases = sqlc.arg(aliases), kind = sqlc.arg(kind), lat = sqlc.arg(lat),
    lng = sqlc.arg(lng), active = sqlc.arg(active), updated_at = sqlc.arg(now), edited_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: LocationForUpdate :one
SELECT * FROM locations WHERE id = $1 FOR UPDATE;

-- name: AdminListPosts :many
SELECT p.*, l.name AS location_name, z.name AS zone_name
FROM posts p
JOIN locations l ON l.id = p.location_id
JOIN zones z ON z.id = l.zone_id
ORDER BY p.name;

-- name: InsertPost :exec
INSERT INTO posts (id, slug, name, location_id, phone, opening_hours, active, created_at, updated_at, edited_at)
VALUES (sqlc.arg(id), sqlc.arg(slug), sqlc.arg(name), sqlc.arg(location_id), sqlc.narg(phone), sqlc.arg(opening_hours), true,
        sqlc.arg(now), sqlc.arg(now), sqlc.arg(now));

-- name: UpdatePost :exec
UPDATE posts
SET name = sqlc.arg(name), location_id = sqlc.arg(location_id), phone = sqlc.narg(phone), opening_hours = sqlc.arg(opening_hours),
    active = sqlc.arg(active), updated_at = sqlc.arg(now), edited_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: PostForUpdate :one
SELECT * FROM posts WHERE id = $1 FOR UPDATE;

-- name: AdminListCompatibility :many
SELECT cc.category_a, cc.category_b, cc.factor::float8 AS factor, cc.active, a.name AS name_a, b.name AS name_b
FROM category_compatibility cc
JOIN categories a ON a.id = cc.category_a
JOIN categories b ON b.id = cc.category_b
ORDER BY a.name, b.name;

-- name: CompatibilityForUpdate :one
SELECT category_a, category_b, factor::float8 AS factor, active FROM category_compatibility
WHERE category_a = $1 AND category_b = $2 FOR UPDATE;

-- name: PutCompatibility :exec
INSERT INTO category_compatibility (category_a, category_b, factor, active, edited_at)
VALUES (sqlc.arg(category_a), sqlc.arg(category_b), sqlc.arg(factor), sqlc.arg(active), sqlc.arg(now))
ON CONFLICT (category_a, category_b) DO UPDATE
SET factor = excluded.factor, active = excluded.active, edited_at = excluded.edited_at;
