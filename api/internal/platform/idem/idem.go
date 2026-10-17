package idem

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
)

const TTL = 24 * time.Hour

type Key struct {
	UserID   uuid.UUID
	Key      string
	Endpoint string
}

// a non nil id means the same request was already served
func Begin(ctx context.Context, q *sqlc.Queries, k Key, request any, now time.Time) (*uuid.UUID, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("idem: marshal: %w", err)
	}
	sum := sha256.Sum256(raw)
	if err := q.DeleteExpiredIdempotencyKey(ctx, sqlc.DeleteExpiredIdempotencyKeyParams{UserID: k.UserID, Key: k.Key, ExpiresAt: now}); err != nil {
		return nil, fmt.Errorf("idem: purge: %w", err)
	}
	n, err := q.ClaimIdempotencyKey(ctx, sqlc.ClaimIdempotencyKeyParams{
		UserID: k.UserID, Key: k.Key, Endpoint: k.Endpoint, RequestHash: sum[:], CreatedAt: now, ExpiresAt: now.Add(TTL),
	})
	if err != nil {
		return nil, fmt.Errorf("idem: claim: %w", err)
	}
	if n == 1 {
		return nil, nil
	}
	row, err := q.GetIdempotencyKey(ctx, sqlc.GetIdempotencyKeyParams{UserID: k.UserID, Key: k.Key})
	if err != nil {
		return nil, fmt.Errorf("idem: load: %w", err)
	}
	if row.Endpoint != k.Endpoint || !bytes.Equal(row.RequestHash, sum[:]) {
		return nil, apperr.Unprocessable("idempotency_key_reused", "Kunci permintaan sudah dipakai", "Kunci permintaan ini sudah dipakai untuk data lain.")
	}
	if row.ResourceID == nil {
		return nil, apperr.New(http.StatusConflict, "request_in_progress", "Permintaan sedang diproses", "Permintaan yang sama sedang diproses.")
	}
	return row.ResourceID, nil
}

func Complete(ctx context.Context, q *sqlc.Queries, k Key, resource uuid.UUID) error {
	if err := q.SetIdempotencyResource(ctx, sqlc.SetIdempotencyResourceParams{UserID: k.UserID, Key: k.Key, ResourceID: &resource}); err != nil {
		return fmt.Errorf("idem: complete: %w", err)
	}
	return nil
}
