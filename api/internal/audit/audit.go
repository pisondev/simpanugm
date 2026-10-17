package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/platform/reqctx"
)

type Entry struct {
	Action     string
	EntityType string
	EntityID   uuid.UUID
	Changes    map[string]any
}

// call inside the transaction of the change it describes, FR-AUDIT-01
func Record(ctx context.Context, q *sqlc.Queries, now time.Time, e Entry) error {
	p := access.FromContext(ctx)
	m := reqctx.From(ctx)
	params := sqlc.InsertAuditParams{
		OccurredAt: now,
		Action:     e.Action,
		EntityType: e.EntityType,
	}
	if e.EntityID != uuid.Nil {
		id := e.EntityID
		params.EntityID = &id
	}
	if p != nil {
		id, role := p.UserID, string(p.Role)
		params.ActorID, params.ActorRole = &id, &role
	} else {
		role := "system"
		params.ActorRole = &role
	}
	if m.RequestID != "" {
		params.RequestID = &m.RequestID
	}
	if m.IP.IsValid() {
		ip := m.IP
		params.Ip = &ip
	}
	if m.UserAgent != "" {
		params.UserAgent = &m.UserAgent
	}
	if len(e.Changes) > 0 {
		b, err := json.Marshal(e.Changes)
		if err != nil {
			return fmt.Errorf("audit: marshal changes: %w", err)
		}
		params.Changes = b
	}
	if err := q.InsertAudit(ctx, params); err != nil {
		return fmt.Errorf("audit: insert: %w", err)
	}
	return nil
}

func Change(before, after any) [2]any { return [2]any{before, after} }
