-- name: DeleteExpiredIdempotencyKey :exec
DELETE FROM idempotency_keys WHERE user_id = $1 AND key = $2 AND expires_at < $3;

-- name: ClaimIdempotencyKey :execrows
INSERT INTO idempotency_keys (user_id, key, endpoint, request_hash, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, key) DO NOTHING;

-- name: GetIdempotencyKey :one
SELECT * FROM idempotency_keys WHERE user_id = $1 AND key = $2;

-- name: SetIdempotencyResource :exec
UPDATE idempotency_keys SET resource_id = $3 WHERE user_id = $1 AND key = $2;

-- name: DeleteExpiredIdempotencyKeys :execrows
DELETE FROM idempotency_keys WHERE expires_at < $1;
