package settings

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
)

const refreshEvery = 60 * time.Second

var defaults = map[string]any{
	"auth.allowed_email_domains":      []string{"mail.ugm.ac.id", "ugm.ac.id"},
	"lost.stale_days":                 60,
	"lost.expiry_grace_days":          14,
	"lost.reopen_days":                30,
	"found.with_finder_max_hours":     48,
	"custody.retention_days.normal":   30,
	"custody.retention_days.valuable": 60,
	"custody.retention_days.identity": 30,
	"claim.max_open":                  3,
	"notify.match_throttle_hours":     6,
	"analytics.min_cell_count":        3,
}

type Store struct {
	q        *sqlc.Queries
	now      func() time.Time
	mu       sync.Mutex
	values   map[string]json.RawMessage
	loadedAt time.Time
}

func New(q *sqlc.Queries, now func() time.Time) *Store {
	return &Store{q: q, now: now}
}

func (s *Store) raw(ctx context.Context, key string) (json.RawMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil || s.now().Sub(s.loadedAt) > refreshEvery {
		rows, err := s.q.ListSettings(ctx)
		if err != nil {
			slog.WarnContext(ctx, "settings refresh failed, keeping cached values", "error", err.Error())
		} else {
			s.values = make(map[string]json.RawMessage, len(rows))
			for _, r := range rows {
				s.values[r.Key] = r.Value
			}
			s.loadedAt = s.now()
		}
	}
	v, ok := s.values[key]
	return v, ok
}

func (s *Store) Invalidate() {
	s.mu.Lock()
	s.values = nil
	s.mu.Unlock()
}

func (s *Store) Int(ctx context.Context, key string) int {
	if v, ok := s.raw(ctx, key); ok {
		var n int
		if json.Unmarshal(v, &n) == nil {
			return n
		}
	}
	n, _ := defaults[key].(int)
	return n
}

func (s *Store) Strings(ctx context.Context, key string) []string {
	if v, ok := s.raw(ctx, key); ok {
		var out []string
		if json.Unmarshal(v, &out) == nil {
			return out
		}
	}
	out, _ := defaults[key].([]string)
	return out
}

type QuietHours struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// BR-NTF-03, both ends are WIB wall clock times
func (s *Store) QuietHours(ctx context.Context) QuietHours {
	if v, ok := s.raw(ctx, "notify.email_quiet_hours"); ok {
		var q QuietHours
		if json.Unmarshal(v, &q) == nil && q.From != "" && q.To != "" {
			return q
		}
	}
	return QuietHours{From: "22:00", To: "06:00"}
}

func (s *Store) AllowedEmailDomains(ctx context.Context) []string {
	return s.Strings(ctx, "auth.allowed_email_domains")
}

func (s *Store) RetentionDays(ctx context.Context, sensitivity string, categoryOverride *int32) int {
	if categoryOverride != nil && *categoryOverride > 0 {
		return int(*categoryOverride)
	}
	return s.Int(ctx, "custody.retention_days."+sensitivity)
}

func Defaults() map[string]any { return defaults }
