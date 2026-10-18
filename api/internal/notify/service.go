package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/cursor"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/platform/httpx"
)

const (
	heartbeatEvery = 25 * time.Second
	// a stream ends after this long and the browser reconnects, so a session that expired meanwhile stops receiving
	streamLifetime = 10 * time.Minute
	writeTimeout   = 10 * time.Second
)

type Service struct {
	pool  *pgxpool.Pool
	q     *sqlc.Queries
	clock clock.Clock
	hub   *Hub
}

func NewService(pool *pgxpool.Pool, c clock.Clock, hub *Hub) *Service {
	return &Service{pool: pool, q: sqlc.New(pool), clock: c, hub: hub}
}

type ListInput struct {
	UnreadOnly bool
	Limit      int
	Cursor     *cursor.Key
}

// FR-NOTIF-01, the query is scoped to the caller so another user's rows can never appear
func (s *Service) List(ctx context.Context, p *access.Principal, in ListInput) ([]sqlc.Notification, error) {
	params := sqlc.ListNotificationsParams{RecipientID: p.UserID, UnreadOnly: in.UnreadOnly, RowLimit: int32(in.Limit + 1)}
	if in.Cursor != nil {
		params.CursorCreated, params.CursorID = &in.Cursor.At, &in.Cursor.ID
	}
	return s.q.ListNotifications(ctx, params)
}

func (s *Service) UnreadCount(ctx context.Context, p *access.Principal) (int, error) {
	n, err := s.q.CountUnread(ctx, p.UserID)
	return int(n), err
}

func (s *Service) MarkRead(ctx context.Context, p *access.Principal, id uuid.UUID) error {
	if _, err := s.q.GetNotificationCreated(ctx, sqlc.GetNotificationCreatedParams{ID: id, RecipientID: p.UserID}); err != nil {
		if db.IsNoRows(err) {
			return apperr.NotFound()
		}
		return err
	}
	_, err := s.q.MarkRead(ctx, sqlc.MarkReadParams{ID: id, RecipientID: p.UserID, ReadAt: ptr(s.clock.Now())})
	return err
}

func (s *Service) MarkAllRead(ctx context.Context, p *access.Principal) error {
	_, err := s.q.MarkAllRead(ctx, sqlc.MarkAllReadParams{RecipientID: p.UserID, ReadAt: ptr(s.clock.Now())})
	return err
}

type Preference struct {
	Group string
	App   bool
	Email bool
}

// every optional group is listed, a group without a stored row is on for both channels
func (s *Service) Preferences(ctx context.Context, p *access.Principal) ([]Preference, error) {
	rows, err := s.q.ListPreferences(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	out := make([]Preference, 0, len(optionalGroups))
	for _, g := range optionalGroups {
		pref := Preference{Group: g, App: true, Email: true}
		for _, r := range rows {
			if r.EventGroup != g {
				continue
			}
			if r.Channel == "app" {
				pref.App = r.Enabled
			} else {
				pref.Email = r.Enabled
			}
		}
		out = append(out, pref)
	}
	return out, nil
}

func (s *Service) SavePreferences(ctx context.Context, p *access.Principal, prefs []Preference) ([]Preference, error) {
	seen := map[string]bool{}
	for _, pref := range prefs {
		if !slices.Contains(optionalGroups, pref.Group) || seen[pref.Group] {
			return nil, apperr.Validation(apperr.FieldError{Field: "items", Code: "invalid", Message: "Grup notifikasi tidak dikenal atau berulang"})
		}
		seen[pref.Group] = true
	}
	now := s.clock.Now()
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		for _, pref := range prefs {
			for channel, enabled := range map[string]bool{"app": pref.App, "email": pref.Email} {
				if err := q.UpsertPreference(ctx, sqlc.UpsertPreferenceParams{
					UserID: p.UserID, EventGroup: pref.Group, Channel: channel, Enabled: enabled, UpdatedAt: now,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Preferences(ctx, p)
}

type streamEvent struct {
	ID         uuid.UUID  `json:"id"`
	Event      string     `json:"event"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	Link       string     `json:"link"`
	EntityType *string    `json:"entity_type"`
	EntityID   *uuid.UUID `json:"entity_id"`
	ReadAt     *time.Time `json:"read_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// FR-NOTIF-02: Server Sent Events, resumed from Last-Event-ID, heartbeats keep proxies from closing the line
func (s *Service) Stream(w http.ResponseWriter, r *http.Request) {
	p := access.FromContext(r.Context())
	if p == nil {
		httpx.WriteProblem(w, r, apperr.Unauthenticated())
		return
	}
	ctx := r.Context()
	after, err := s.resumePoint(ctx, p.UserID, r.Header.Get("Last-Event-ID"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	wake, cancel := s.hub.Subscribe(p.UserID)
	defer cancel()

	rc := http.NewResponseController(w)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	write := func(chunk string) bool {
		_ = rc.SetWriteDeadline(time.Now().Add(writeTimeout))
		if _, err := fmt.Fprint(w, chunk); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	if !write("retry: 3000\n\n") {
		return
	}
	heartbeat := time.NewTicker(heartbeatEvery)
	defer heartbeat.Stop()
	end := time.NewTimer(streamLifetime)
	defer end.Stop()
	for {
		rows, err := s.q.ListNotificationsAfter(ctx, sqlc.ListNotificationsAfterParams{RecipientID: p.UserID, AfterCreated: after.At, AfterID: after.ID})
		if err != nil {
			if ctx.Err() == nil {
				slog.WarnContext(ctx, "notification stream read failed", "error", err.Error())
			}
			return
		}
		for _, n := range rows {
			payload, _ := json.Marshal(streamEvent{
				ID: n.ID, Event: n.Event, Title: n.Title, Body: n.Body, Link: n.Link,
				EntityType: n.EntityType, EntityID: n.EntityID, ReadAt: n.ReadAt, CreatedAt: n.CreatedAt,
			})
			if !write(fmt.Sprintf("id: %s\nevent: notification\ndata: %s\n\n", n.ID, payload)) {
				return
			}
			after = cursor.Key{At: n.CreatedAt, ID: n.ID}
		}
		if len(rows) == 50 {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-end.C:
			return
		case <-heartbeat.C:
			if !write("event: heartbeat\ndata: {}\n\n") {
				return
			}
		case <-wake:
		}
	}
}

// without a usable Last-Event-ID the stream starts at the newest notification, older ones are in the list endpoint
func (s *Service) resumePoint(ctx context.Context, user uuid.UUID, last string) (cursor.Key, error) {
	if id, err := uuid.Parse(last); err == nil {
		at, err := s.q.GetNotificationCreated(ctx, sqlc.GetNotificationCreatedParams{ID: id, RecipientID: user})
		if err == nil {
			return cursor.Key{At: at, ID: id}, nil
		}
		if !db.IsNoRows(err) {
			return cursor.Key{}, err
		}
	}
	rows, err := s.q.ListNotifications(ctx, sqlc.ListNotificationsParams{RecipientID: user, RowLimit: 1})
	if err != nil {
		return cursor.Key{}, err
	}
	if len(rows) == 0 {
		return cursor.Key{At: time.Unix(0, 0).UTC(), ID: uuid.Nil}, nil
	}
	return cursor.Key{At: rows[0].CreatedAt, ID: rows[0].ID}, nil
}

func ptr[T any](v T) *T { return &v }
