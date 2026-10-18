package claims

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
)

// US-29: claims that stall do not hold an item forever
type ExpireArgs struct{}

func (ExpireArgs) Kind() string { return "claims.expire" }

func (ExpireArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "maintenance", MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByPeriod: 10 * time.Minute}}
}

type ExpireWorker struct {
	river.WorkerDefaults[ExpireArgs]
	S *Service
}

func (w *ExpireWorker) Work(ctx context.Context, _ *river.Job[ExpireArgs]) error {
	expired, warned, err := w.S.ExpireDue(ctx)
	if expired+warned > 0 {
		slog.InfoContext(ctx, "claims swept", "expired", expired, "warned", warned)
	}
	return err
}

func Periodic() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(15*time.Minute), func() (river.JobArgs, *river.InsertOpts) {
			return ExpireArgs{}, nil
		}, nil),
	}
}

// the one day warning of BR-CLM-06 first, then the expiry of K5 and K12
func (s *Service) ExpireDue(ctx context.Context) (expired, warned int, err error) {
	now := s.clock.Now()
	soon, err := s.q.ClaimsExpiringSoon(ctx, sqlc.ClaimsExpiringSoonParams{Until: now.Add(24 * time.Hour), Now: now})
	if err != nil {
		return 0, 0, err
	}
	for _, id := range soon {
		ok, err := s.warn(ctx, id)
		if err != nil {
			return expired, warned, err
		}
		if ok {
			warned++
		}
	}
	due, err := s.q.ClaimsDueToExpire(ctx, sqlc.ClaimsDueToExpireParams{InfoBefore: now.Add(-s.infoTimeout(ctx)), Now: now})
	if err != nil {
		return expired, warned, err
	}
	for _, id := range due {
		ok, err := s.expire(ctx, id)
		if err != nil {
			return expired, warned, err
		}
		if ok {
			expired++
		}
	}
	return expired, warned, nil
}

func (s *Service) warn(ctx context.Context, id uuid.UUID) (bool, error) {
	done := false
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		c, err := q.ClaimForUpdate(ctx, sqlc.ClaimForUpdateParams{ID: &id})
		if err != nil || c.Status != "approved" || c.ExpiringNotifiedAt != nil {
			return err
		}
		v, err := q.ClaimView(ctx, sqlc.ClaimViewParams{ID: &id})
		if err != nil {
			return err
		}
		if err := q.MarkClaimExpiringNotified(ctx, sqlc.MarkClaimExpiringNotifiedParams{ID: id, ExpiringNotifiedAt: ptrTime(s.clock.Now())}); err != nil {
			return err
		}
		done = true
		if v.ClaimantID == nil {
			return nil
		}
		post := ""
		if v.HoldingPostName != nil {
			post = *v.HoldingPostName
		}
		return s.tell(ctx, tx, notify.ClaimExpiring, *v.ClaimantID, memberLink(c.Code), map[string]string{"claim_code": c.Code, "post_name": post}, id)
	})
	return done, err
}

// K5 for a claim waiting on its claimant, K12 for an approval nobody collected
func (s *Service) expire(ctx context.Context, id uuid.UUID) (bool, error) {
	done := false
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		c, err := q.ClaimForUpdate(ctx, sqlc.ClaimForUpdateParams{ID: &id})
		if err != nil {
			return err
		}
		now := s.clock.Now()
		switch {
		case c.Status == "info_requested" && c.InfoRequestedAt != nil && !c.InfoRequestedAt.After(now.Add(-s.infoTimeout(ctx))):
			if err := s.step(ctx, q, id, c.Status, "info_timeout", "expired", nil); err != nil {
				return err
			}
		case c.Status == "approved" && c.ApprovedExpiresAt != nil && !c.ApprovedExpiresAt.After(now):
			if err := s.step(ctx, q, id, c.Status, "approval_timeout", "expired", nil); err != nil {
				return err
			}
		default:
			return nil
		}
		approved := c.Status == "approved"
		if err := q.EndClaim(ctx, sqlc.EndClaimParams{ID: id, Status: "expired", UpdatedAt: now}); err != nil {
			return err
		}
		if approved {
			if err := s.releaseApproval(ctx, tx, q, c); err != nil {
				return err
			}
		} else if err := s.resync(ctx, q, c.LostReportID, c.FoundReportID); err != nil {
			return err
		}
		done = true
		if c.ClaimantID != nil {
			if err := s.tell(ctx, tx, notify.ClaimExpired, *c.ClaimantID, memberLink(c.Code), map[string]string{"claim_code": c.Code}, id); err != nil {
				return err
			}
		}
		return s.audit(ctx, q, "claim.expire", c, map[string]any{"from": c.Status})
	})
	return done, err
}
