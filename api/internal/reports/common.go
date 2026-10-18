package reports

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/custody"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/matching/jobargs"
	"github.com/pisondev/simpanugm/api/internal/media"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
	"github.com/pisondev/simpanugm/api/internal/platform/textx"
)

const (
	maxOpenLost       = 5
	maxFoundPerWeek   = 10
	futureTolerance   = 5 * time.Minute
	lostWindowMax     = 7 * 24 * time.Hour
	lostLookback      = 90 * 24 * time.Hour
	foundLookback     = 30 * 24 * time.Hour
	closeNoteMinChars = 5
)

var codePattern = regexp.MustCompile(`^(LST|FND|CLM)-[0-9]{4}-[0-9]{4,5}$`)

type Service struct {
	pool     *pgxpool.Pool
	q        *sqlc.Queries
	clock    clock.Clock
	media    *media.Service
	md       *masterdata.Service
	custody  *custody.Service
	settings *settings.Store
	notifier *notify.Notifier
	matching *jobargs.Enqueuer
}

func NewService(pool *pgxpool.Pool, c clock.Clock, m *media.Service, md *masterdata.Service, cs *custody.Service, st *settings.Store, n *notify.Notifier, enq *jobargs.Enqueuer) *Service {
	return &Service{pool: pool, q: sqlc.New(pool), clock: c, media: m, md: md, custody: cs, settings: st, notifier: n, matching: enq}
}

// photos in the new set that the report did not carry before, only those need embedding
func addedMedia(before []sqlc.Medium, after []uuid.UUID) []uuid.UUID {
	had := map[uuid.UUID]bool{}
	for _, m := range before {
		had[m.ID] = true
	}
	var out []uuid.UUID
	for _, id := range after {
		if !had[id] {
			out = append(out, id)
		}
	}
	return out
}

func IsCode(s string) bool { return codePattern.MatchString(s) }

func refOf(idOrCode string) (*uuid.UUID, *string, error) {
	if id, err := uuid.Parse(idOrCode); err == nil {
		return &id, nil, nil
	}
	if IsCode(idOrCode) {
		return nil, &idOrCode, nil
	}
	return nil, nil, apperr.NotFound()
}

// codes take the WIB month, BR-ID-01
func NextCode(ctx context.Context, q *sqlc.Queries, prefix string, now time.Time) (string, error) {
	yymm := now.In(clock.WIB()).Format("0601")
	n, err := q.NextCodeSequence(ctx, sqlc.NextCodeSequenceParams{Prefix: prefix, Yymm: yymm})
	if err != nil {
		return "", fmt.Errorf("reports: code sequence: %w", err)
	}
	return fmt.Sprintf("%s-%s-%04d", prefix, yymm, n), nil
}

func (s *Service) history(ctx context.Context, q *sqlc.Queries, entity string, id uuid.UUID, from, to, transition string, reason *string) error {
	p := access.FromContext(ctx)
	params := sqlc.InsertStatusHistoryParams{
		EntityType: entity, EntityID: id, ToStatus: to, Transition: transition, Reason: reason, ActorKind: "system", CreatedAt: s.clock.Now(),
	}
	if from != "" {
		params.FromStatus = &from
	}
	if p != nil {
		params.ActorID, params.ActorKind = &p.UserID, "user"
	}
	return q.InsertStatusHistory(ctx, params)
}

type textRule struct {
	field    string
	value    *string
	min, max int
	label    string
}

func checkText(v *apperr.Collector, r textRule) {
	if r.value == nil {
		if r.min > 0 {
			v.Add(r.field, "required", r.label+" wajib diisi")
		}
		return
	}
	n := textx.Len(*r.value)
	switch {
	case n < r.min:
		v.Add(r.field, "too_short", fmt.Sprintf("%s minimal %d karakter, saat ini %d", r.label, r.min, n))
	case n > r.max:
		v.Add(r.field, "too_long", fmt.Sprintf("%s maksimal %d karakter", r.label, r.max))
	}
}

func (s *Service) checkColours(ctx context.Context, q *sqlc.Queries, v *apperr.Collector, colours []string) error {
	if len(colours) > 2 {
		v.Add("colours", "too_many", "Pilih maksimal 2 warna")
		return nil
	}
	palette, err := s.md.ColourNames(ctx, q)
	if err != nil {
		return err
	}
	for _, c := range colours {
		if !palette[c] {
			v.Add("colours", "unknown", "Warna tidak dikenal: "+c)
		}
	}
	if len(colours) == 2 && colours[0] == colours[1] {
		v.Add("colours", "duplicate", "Warna tidak boleh sama")
	}
	return nil
}

func (s *Service) leaf(ctx context.Context, q *sqlc.Queries, v *apperr.Collector, id uuid.UUID) (masterdata.LeafCategory, error) {
	c, ok, err := s.md.Leaf(ctx, q, id)
	if err != nil {
		return c, err
	}
	if !ok {
		v.Add("category_id", "unknown", "Pilih kategori yang tersedia")
	}
	return c, nil
}

func validStatuses(statuses []string, allowed []string) error {
	for _, st := range statuses {
		if !slices.Contains(allowed, st) {
			return apperr.Validation(apperr.FieldError{Field: "status", Code: "unknown", Message: "Status tidak dikenal: " + st})
		}
	}
	return nil
}

var (
	lostStatuses  = []string{"reported", "potential_match", "under_verification", "claimed", "returned", "closed"}
	foundStatuses = []string{"reported", "potential_match", "under_verification", "claimed", "returned", "unclaimed", "closed"}
)

func parseVersion(raw string) (int32, error) {
	var n int32
	trimmed := raw
	if len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
		trimmed = trimmed[1 : len(trimmed)-1]
	}
	if _, err := fmt.Sscanf(trimmed, "%d", &n); err != nil || n < 1 {
		return 0, apperr.BadRequest("Header If-Match tidak valid.")
	}
	return n, nil
}
