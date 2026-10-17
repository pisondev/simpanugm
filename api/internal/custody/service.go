package custody

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
	"github.com/pisondev/simpanugm/api/internal/platform/textx"
)

type Service struct {
	pool     *pgxpool.Pool
	q        *sqlc.Queries
	clock    clock.Clock
	settings *settings.Store
}

func NewService(pool *pgxpool.Pool, c clock.Clock, st *settings.Store) *Service {
	return &Service{pool: pool, q: sqlc.New(pool), clock: c, settings: st}
}

func NotAssigned() *apperr.Error {
	return apperr.New(403, "post_not_assigned", "Bukan pos kamu", "Kamu tidak ditugaskan di pos ini.")
}

// BR-CUS-03
func (s *Service) CheckPost(ctx context.Context, q *sqlc.Queries, p *access.Principal, post uuid.UUID) (sqlc.GetPostRow, error) {
	row, err := q.GetPost(ctx, post)
	if db.IsNoRows(err) || (err == nil && !row.Active) {
		return row, apperr.Validation(apperr.FieldError{Field: "post_id", Code: "unknown_post", Message: "Pos tidak ditemukan atau tidak aktif"})
	}
	if err != nil {
		return row, err
	}
	if !p.AssignedTo(post) {
		return row, NotAssigned()
	}
	return row, nil
}

func (s *Service) RetentionDue(ctx context.Context, from time.Time, sensitivity string, override *int32) time.Time {
	return from.AddDate(0, 0, s.settings.RetentionDays(ctx, sensitivity, override))
}

type ReceiveInput struct {
	FoundID       uuid.UUID
	Version       int32
	PostID        uuid.UUID
	StorageSlot   string
	ConditionNote *string
	Sensitivity   string
	RetentionDays *int32
}

func ValidateSlot(slot string) error {
	if n := textx.Len(slot); n < 1 || n > 40 {
		return apperr.Validation(apperr.FieldError{Field: "storage_slot", Code: "invalid_length", Message: "Lokasi simpan 1 sampai 40 karakter"})
	}
	return nil
}

// transition C3, runs inside the caller's transaction
func (s *Service) Receive(ctx context.Context, q *sqlc.Queries, p *access.Principal, in ReceiveInput) (time.Time, error) {
	if err := ValidateSlot(in.StorageSlot); err != nil {
		return time.Time{}, err
	}
	if _, err := s.CheckPost(ctx, q, p, in.PostID); err != nil {
		return time.Time{}, err
	}
	now := s.clock.Now()
	due := s.RetentionDue(ctx, now, in.Sensitivity, in.RetentionDays)
	slot := in.StorageSlot
	_, err := q.ReceiveIntoCustody(ctx, sqlc.ReceiveIntoCustodyParams{
		PostID: &in.PostID, StorageSlot: &slot, Now: now, OfficerID: &p.UserID, RetentionDueAt: due, ID: in.FoundID, Version: in.Version,
	})
	if db.IsNoRows(err) {
		return time.Time{}, apperr.StaleVersion()
	}
	if err != nil {
		return time.Time{}, err
	}
	return due, s.Event(ctx, q, in.FoundID, "received", nil, &in.PostID, p.UserID, in.ConditionNote)
}

func (s *Service) Event(ctx context.Context, q *sqlc.Queries, found uuid.UUID, kind string, from, to *uuid.UUID, officer uuid.UUID, note *string) error {
	return q.InsertCustodyEvent(ctx, sqlc.InsertCustodyEventParams{
		ID: ids.New(), FoundReportID: found, Kind: kind, FromPostID: from, ToPostID: to, OfficerID: &officer, Note: textx.Opt(note), CreatedAt: s.clock.Now(),
	})
}

type InventoryItem struct {
	sqlc.ListInventoryRow
	AgeDays int
}

func (s *Service) Inventory(ctx context.Context, post uuid.UUID, sort string) (sqlc.GetPostRow, []InventoryItem, error) {
	row, err := s.q.GetPost(ctx, post)
	if db.IsNoRows(err) {
		return row, nil, apperr.NotFound()
	}
	if err != nil {
		return row, nil, err
	}
	if sort == "" {
		sort = "retention_due_at"
	}
	items, err := s.q.ListInventory(ctx, sqlc.ListInventoryParams{PostID: &post, Sort: sort})
	if err != nil {
		return row, nil, err
	}
	now := s.clock.Now()
	out := make([]InventoryItem, 0, len(items))
	for _, it := range items {
		out = append(out, InventoryItem{ListInventoryRow: it, AgeDays: int(now.Sub(it.ReceivedAt).Hours() / 24)})
	}
	return row, out, nil
}
