-- name: CreateUser :one
INSERT INTO users (id, email, full_name, nim_nip, phone, password_hash, role, status, privacy_version, privacy_accepted_at, invited_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: MarkEmailVerified :exec
UPDATE users
SET email_verified_at = coalesce(email_verified_at, sqlc.arg(now)::timestamptz),
    status = CASE WHEN status = 'pending_verification' THEN 'active' ELSE status END,
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id);

-- name: ActivateInvitedUser :exec
UPDATE users
SET password_hash = sqlc.arg(password_hash),
    status = 'active',
    email_verified_at = coalesce(email_verified_at, sqlc.arg(now)::timestamptz),
    version = version + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND status = 'pending_verification';

-- name: UpdatePasswordHash :exec
UPDATE users SET password_hash = $2, version = version + 1, updated_at = $3 WHERE id = $1;

-- name: SetLastLogin :exec
UPDATE users SET last_login_at = $2 WHERE id = $1;

-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, token_hash, created_at, last_seen_at, expires_at, idle_expires_at, ip, user_agent)
VALUES ($1, $2, $3, $4, $4, $5, $6, $7, $8);

-- name: GetSessionByTokenHash :one
SELECT s.id AS session_id, s.expires_at, s.idle_expires_at, s.last_seen_at, s.revoked_at,
       u.id AS user_id, u.email, u.full_name, u.role, u.status, u.email_verified_at, u.claim_restricted_at
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1;

-- name: TouchSession :exec
UPDATE sessions
SET last_seen_at = sqlc.arg(now)::timestamptz,
    expires_at = sqlc.arg(expires_at)::timestamptz,
    idle_expires_at = sqlc.narg(idle_expires_at)::timestamptz
WHERE id = sqlc.arg(id);

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeUserSessions :execrows
UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL;

-- name: CreateAuthToken :exec
INSERT INTO auth_tokens (id, user_id, purpose, token_hash, new_email, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetUsableAuthToken :one
SELECT * FROM auth_tokens
WHERE token_hash = sqlc.arg(token_hash)
  AND purpose = sqlc.arg(purpose)
  AND used_at IS NULL
  AND expires_at > sqlc.arg(now)::timestamptz
FOR UPDATE;

-- name: MarkAuthTokenUsed :exec
UPDATE auth_tokens SET used_at = $2 WHERE id = $1;

-- name: InvalidateUserTokens :exec
UPDATE auth_tokens SET used_at = sqlc.arg(now)::timestamptz
WHERE user_id = sqlc.arg(user_id) AND purpose = sqlc.arg(purpose) AND used_at IS NULL;

-- name: TokenIssueStats :one
SELECT count(*)::int AS issued_today,
       coalesce(max(created_at), '-infinity'::timestamptz)::timestamptz AS last_issued_at
FROM auth_tokens
WHERE user_id = sqlc.arg(user_id) AND purpose = sqlc.arg(purpose) AND created_at > sqlc.arg(since)::timestamptz;

-- name: ListUserPostIDs :many
SELECT post_id FROM officer_posts WHERE user_id = $1 ORDER BY post_id;

-- name: DeleteOfficerPosts :exec
DELETE FROM officer_posts WHERE user_id = $1;

-- name: InsertOfficerPost :exec
INSERT INTO officer_posts (user_id, post_id, assigned_at, assigned_by) VALUES ($1, $2, $3, $4);

-- name: CountActivePostsByIDs :one
SELECT count(*)::int FROM posts WHERE id = ANY(sqlc.arg(ids)::uuid[]) AND active;

-- name: UpdateUserRole :exec
UPDATE users SET role = $2, version = version + 1, updated_at = $3 WHERE id = $1;

-- name: CountActiveAdmins :one
SELECT count(*)::int FROM users WHERE role = 'admin' AND status = 'active';

-- name: ListUsers :many
SELECT * FROM users
WHERE status <> 'deleted'
  AND (sqlc.narg(role)::text IS NULL OR role = sqlc.narg(role)::text)
  AND (sqlc.narg(q)::text IS NULL
       OR email ILIKE '%' || sqlc.narg(q)::text || '%'
       OR search_key(full_name) LIKE '%' || search_key(sqlc.narg(q)::text) || '%')
  AND (sqlc.narg(cursor_created)::timestamptz IS NULL
       OR (created_at, id) < (sqlc.narg(cursor_created)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListPostIDsForUsers :many
SELECT user_id, post_id FROM officer_posts WHERE user_id = ANY(sqlc.arg(user_ids)::uuid[]);

-- name: DeleteUnverifiedUsersBefore :execrows
DELETE FROM users
WHERE users.status = 'pending_verification'
  AND users.role = 'member'
  AND users.email_verified_at IS NULL
  AND users.created_at < sqlc.arg(before)::timestamptz
  AND NOT EXISTS (SELECT 1 FROM lost_reports WHERE reporter_id = users.id)
  AND NOT EXISTS (SELECT 1 FROM found_reports WHERE finder_id = users.id)
  AND NOT EXISTS (SELECT 1 FROM media WHERE uploaded_by = users.id);

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at < $1;
