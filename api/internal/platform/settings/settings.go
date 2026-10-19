package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/geo"
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
	"claim.reject_flag_count":         3,
	"claim.reject_flag_window_days":   30,
	"claim.info_timeout_days":         7,
	"claim.approved_timeout_days":     7,
	"notify.match_throttle_hours":     6,
	"notify.email_quiet_hours":        map[string]string{"from": "22:00", "to": "06:00"},
	"analytics.min_cell_count":        3,
}

type Store struct {
	pool     *pgxpool.Pool
	q        *sqlc.Queries
	now      func() time.Time
	mu       sync.Mutex
	values   map[string]json.RawMessage
	loadedAt time.Time
}

func New(pool *pgxpool.Pool, q *sqlc.Queries, now func() time.Time) *Store {
	return &Store{pool: pool, q: q, now: now}
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

// the campus.bounding_box setting, false when it is missing or malformed
func (s *Store) CampusBox(ctx context.Context) (geo.Box, bool) {
	var b geo.Box
	v, ok := s.raw(ctx, "campus.bounding_box")
	if !ok || json.Unmarshal(v, &b) != nil || b.MinLat >= b.MaxLat || b.MinLng >= b.MaxLng {
		return geo.Box{}, false
	}
	return b, true
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

func (s *Store) LostStaleDays(ctx context.Context) int {
	return s.Int(ctx, "lost.stale_days")
}

func (s *Store) LostExpiryGraceDays(ctx context.Context) int {
	return s.Int(ctx, "lost.expiry_grace_days")
}

func (s *Store) LostReopenDays(ctx context.Context) int {
	return s.Int(ctx, "lost.reopen_days")
}

func Defaults() map[string]any { return defaults }

func (s *Store) ListDetailed(ctx context.Context) ([]sqlc.ListSettingsDetailedRow, error) {
	rows, err := s.q.ListSettingsDetailed(ctx)
	if err != nil {
		return nil, fmt.Errorf("settings: list detailed: %w", err)
	}
	existing := make(map[string]bool, len(rows))
	for _, r := range rows {
		existing[r.Key] = true
	}
	for k, defVal := range defaults {
		if !existing[k] {
			raw, _ := json.Marshal(defVal)
			rows = append(rows, sqlc.ListSettingsDetailedRow{
				Key:   k,
				Value: raw,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key })
	return rows, nil
}

func (s *Store) Update(ctx context.Context, entries []openapi.UpdateSettingEntry, actorID *uuid.UUID) ([]sqlc.ListSettingsDetailedRow, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("settings: database pool not configured")
	}
	for _, e := range entries {
		if err := ValidateSetting(e.Key, e.Value); err != nil {
			return nil, err
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("settings: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)
	now := s.now()
	changes := make(map[string]any, len(entries))

	for _, e := range entries {
		raw, err := json.Marshal(e.Value)
		if err != nil {
			return nil, apperr.Unprocessable("invalid_setting", "Pengaturan tidak valid", "Gagal memproses nilai pengaturan.")
		}
		oldRaw, _ := s.raw(ctx, e.Key)
		var oldVal any
		if len(oldRaw) > 0 {
			_ = json.Unmarshal(oldRaw, &oldVal)
		} else {
			oldVal = defaults[e.Key]
		}
		changes[e.Key] = audit.Change(oldVal, e.Value)

		if err := qtx.UpsertSetting(ctx, sqlc.UpsertSettingParams{
			Key:       e.Key,
			Value:     raw,
			Now:       now,
			UpdatedBy: actorID,
		}); err != nil {
			return nil, fmt.Errorf("settings: upsert %s: %w", e.Key, err)
		}
	}

	if err := audit.Record(ctx, qtx, now, audit.Entry{
		Action:     "settings.update",
		EntityType: "system_settings",
		Changes:    changes,
	}); err != nil {
		return nil, fmt.Errorf("settings: audit record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("settings: commit: %w", err)
	}

	s.Invalidate()
	return s.ListDetailed(ctx)
}

func ValidateSetting(key string, val any) error {
	switch key {
	case "auth.allowed_email_domains":
		domains, ok := toStringSlice(val)
		if !ok || len(domains) == 0 {
			return apperr.Unprocessable("invalid_setting", "Pengaturan tidak valid", "auth.allowed_email_domains harus berisi minimal satu domain.")
		}
		for _, d := range domains {
			if strings.TrimSpace(d) == "" || !strings.Contains(d, ".") {
				return apperr.Unprocessable("invalid_setting", "Pengaturan tidak valid", fmt.Sprintf("Domain %q tidak valid.", d))
			}
		}
		return nil
	case "lost.stale_days":
		return validateIntRange(key, val, 14, 180)
	case "lost.expiry_grace_days":
		return validateIntRange(key, val, 3, 30)
	case "lost.reopen_days":
		return validateIntRange(key, val, 7, 90)
	case "found.with_finder_max_hours":
		return validateIntRange(key, val, 12, 72)
	case "custody.retention_days.normal":
		return validateIntRange(key, val, 7, 120)
	case "custody.retention_days.valuable":
		return validateIntRange(key, val, 14, 180)
	case "custody.retention_days.identity":
		return validateIntRange(key, val, 7, 90)
	case "claim.max_open":
		return validateIntRange(key, val, 1, 10)
	case "claim.reject_flag_count":
		return validateIntRange(key, val, 2, 10)
	case "claim.reject_flag_window_days":
		return validateIntRange(key, val, 7, 90)
	case "claim.info_timeout_days":
		return validateIntRange(key, val, 2, 30)
	case "claim.approved_timeout_days":
		return validateIntRange(key, val, 2, 30)
	case "notify.match_throttle_hours":
		return validateIntRange(key, val, 1, 24)
	case "notify.email_quiet_hours":
		m, ok := toMap(val)
		if !ok {
			return apperr.Unprocessable("invalid_setting", "Pengaturan tidak valid", "notify.email_quiet_hours harus berupa objek dengan properti from dan to.")
		}
		timePattern := regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
		fromStr, _ := m["from"].(string)
		toStr, _ := m["to"].(string)
		if !timePattern.MatchString(fromStr) || !timePattern.MatchString(toStr) {
			return apperr.Unprocessable("invalid_setting", "Pengaturan tidak valid", "notify.email_quiet_hours from dan to harus berformat HH:MM.")
		}
		return nil
	case "analytics.min_cell_count":
		return validateIntRange(key, val, 3, 20)
	default:
		return apperr.Unprocessable("unknown_setting", "Pengaturan tidak dikenal", fmt.Sprintf("Kunci pengaturan %q tidak dikenali.", key))
	}
}

func validateIntRange(key string, val any, min, max int) error {
	n, ok := toInt(val)
	if !ok {
		return apperr.Unprocessable("invalid_setting", "Pengaturan tidak valid", fmt.Sprintf("%s harus berupa angka bulat.", key))
	}
	if n < min || n > max {
		return apperr.Unprocessable("invalid_setting", "Pengaturan tidak valid", fmt.Sprintf("%s harus bernilai antara %d dan %d.", key, min, max))
	}
	return nil
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		if n == float64(int(n)) {
			return int(n), true
		}
		return 0, false
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}

func toStringSlice(v any) ([]string, bool) {
	switch s := v.(type) {
	case []string:
		return s, true
	case []any:
		out := make([]string, 0, len(s))
		for _, item := range s {
			str, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, str)
		}
		return out, true
	default:
		return nil, false
	}
}

func toMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[string]string:
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[k] = val
		}
		return out, true
	default:
		return nil, false
	}
}
