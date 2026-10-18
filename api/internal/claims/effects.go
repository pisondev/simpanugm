package claims

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/custody"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/media"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
	"github.com/pisondev/simpanugm/api/internal/reports"
)

type Service struct {
	pool     *pgxpool.Pool
	q        *sqlc.Queries
	clock    clock.Clock
	settings *settings.Store
	notifier *notify.Notifier
	media    *media.Service
	md       *masterdata.Service
	custody  *custody.Service
	reports  *reports.Service
	pepper   []byte
}

func NewService(pool *pgxpool.Pool, c clock.Clock, st *settings.Store, n *notify.Notifier, m *media.Service, md *masterdata.Service,
	cs *custody.Service, rep *reports.Service, pepper string,
) *Service {
	return &Service{pool: pool, q: sqlc.New(pool), clock: c, settings: st, notifier: n, media: m, md: md, custody: cs, reports: rep, pepper: []byte(pepper)}
}

// a record of a claim's move, by the signed in officer or member, or by the system when a job runs
func (s *Service) history(ctx context.Context, q *sqlc.Queries, entity string, id uuid.UUID, from, to, transition string, reason *string) error {
	params := sqlc.InsertStatusHistoryParams{
		EntityType: entity, EntityID: id, ToStatus: to, Transition: transition, Reason: reason, ActorKind: "system", CreatedAt: s.clock.Now(),
	}
	if from != "" {
		params.FromStatus = &from
	}
	if p := access.FromContext(ctx); p != nil {
		params.ActorID, params.ActorKind = &p.UserID, "user"
	}
	return q.InsertStatusHistory(ctx, params)
}

// validates the move against state machine K and writes its history row
func (s *Service) step(ctx context.Context, q *sqlc.Queries, claim uuid.UUID, from, event, to string, reason *string) error {
	id, err := Claim.Step(from, event, to)
	if err != nil {
		return err
	}
	return s.history(ctx, q, "claim", claim, from, to, id, reason)
}

// the status a lost or found report should show given its claims and candidates, "" when it is past that logic
func desired(current string, open, approved, candidates int32, keepUnclaimed bool) string {
	switch current {
	case "reported", "potential_match", "under_verification", "claimed":
	case "unclaimed":
		if open == 0 && keepUnclaimed {
			return current
		}
	default:
		return current
	}
	switch {
	case approved > 0:
		return "claimed"
	case open > 0:
		return "under_verification"
	case candidates > 0:
		return "potential_match"
	default:
		return "reported"
	}
}

// L4 to L7 and F4 to F7: the report follows its claims. A report that is returned, closed or hidden is left alone
func (s *Service) resync(ctx context.Context, q *sqlc.Queries, lostID, foundID uuid.UUID) error {
	now := s.clock.Now()
	lost, err := q.LockLostStatus(ctx, lostID)
	if err != nil {
		return err
	}
	lc, err := q.OpenClaimCountOfLost(ctx, lostID)
	if err != nil {
		return err
	}
	lcand, err := q.CandidatesOfLost(ctx, lostID)
	if err != nil {
		return err
	}
	if want := desired(lost, lc.Open, lc.Approved, lcand, false); want != lost {
		if tid, ok := reports.Lost.Between(lost, want); ok {
			if err := q.ForceLostStatus(ctx, sqlc.ForceLostStatusParams{ID: lostID, Status: want, LastActivityAt: now}); err != nil {
				return err
			}
			if err := s.history(ctx, q, "lost_report", lostID, lost, want, tid, nil); err != nil {
				return err
			}
		}
	}
	found, err := q.LockFoundStatus(ctx, foundID)
	if err != nil {
		return err
	}
	fc, err := q.OpenClaimCountOfFound(ctx, foundID)
	if err != nil {
		return err
	}
	fcand, err := q.CandidatesOfFound(ctx, foundID)
	if err != nil {
		return err
	}
	status := found.Status
	// F11: an unclaimed item with a new claim goes back to reported first
	if status == "unclaimed" && fc.Open > 0 {
		if err := q.ForceFoundStatus(ctx, sqlc.ForceFoundStatusParams{ID: foundID, Status: "reported"}); err != nil {
			return err
		}
		if err := s.history(ctx, q, "found_report", foundID, "unclaimed", "reported", "F11", nil); err != nil {
			return err
		}
		status = "reported"
	}
	if want := desired(status, fc.Open, fc.Approved, fcand, true); want != status {
		if tid, ok := reports.Found.Between(status, want); ok {
			if err := q.ForceFoundStatus(ctx, sqlc.ForceFoundStatusParams{ID: foundID, Status: want}); err != nil {
				return err
			}
			if err := s.history(ctx, q, "found_report", foundID, status, want, tid, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// the officers of the post that holds the item, or of the post it is on its way to
func (s *Service) officersOf(ctx context.Context, q *sqlc.Queries, holding, intended *uuid.UUID) ([]uuid.UUID, error) {
	var posts []uuid.UUID
	switch {
	case holding != nil:
		posts = append(posts, *holding)
	case intended != nil:
		posts = append(posts, *intended)
	}
	if len(posts) == 0 {
		return nil, nil
	}
	return q.ListUserIDsForPosts(ctx, posts)
}

func (s *Service) tell(ctx context.Context, tx pgx.Tx, event notify.Event, to uuid.UUID, link string, data map[string]string, claim uuid.UUID) error {
	return s.notifier.Send(ctx, tx, notify.Message{Event: event, To: to, Link: link, Data: data, EntityType: "claim", EntityID: &claim})
}

func memberLink(code string) string  { return "/klaim/" + code }
func officerLink(code string) string { return "/petugas/klaim/" + code }

// a pair of lost and found reports moves with its claim, M5 to M7
func (s *Service) pair(ctx context.Context, q *sqlc.Queries, lostID, foundID uuid.UUID, from []string, to string) error {
	return q.SetPairStatus(ctx, sqlc.SetPairStatusParams{LostReportID: lostID, FoundReportID: foundID, Status: to, FromStatuses: from})
}

func (s *Service) approvedFor(ctx context.Context) time.Duration {
	return time.Duration(s.settings.Int(ctx, "claim.approved_timeout_days")) * 24 * time.Hour
}

func (s *Service) infoTimeout(ctx context.Context) time.Duration {
	return time.Duration(s.settings.Int(ctx, "claim.info_timeout_days")) * 24 * time.Hour
}
