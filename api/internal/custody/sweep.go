package custody

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
)

type SweepArgs struct{}

func (SweepArgs) Kind() string { return "custody.sweep" }

func (SweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "maintenance", MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByPeriod: 10 * time.Minute}}
}

type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	S *Service
}

func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	unclaimed, warned, overdue, reminders, err := w.S.SweepDue(ctx)
	if unclaimed+warned+overdue+reminders > 0 {
		slog.InfoContext(ctx, "custody swept", "unclaimed", unclaimed, "warned", warned, "overdue", overdue, "reminders", reminders)
	}
	return err
}

func Periodic() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(15*time.Minute), func() (river.JobArgs, *river.InsertOpts) {
			return SweepArgs{}, nil
		}, nil),
	}
}

// BR-CUS-02, BR-CUS-05, BR-FND-03 and US-16
func (s *Service) SweepDue(ctx context.Context) (unclaimed, warned, overdue, reminders int, err error) {
	now := s.clock.Now()

	// BR-CUS-02 retention sweep
	expired, err := s.q.ListRetentionExpired(ctx, now)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	for _, fr := range expired {
		ok, err := s.markUnclaimed(ctx, fr.ID, fr.Version, fr.HoldingPostID, fr.Code)
		if err != nil {
			return unclaimed, warned, overdue, reminders, err
		}
		if ok {
			unclaimed++
		}
	}

	// BR-CUS-05 7-day retention warning
	warnBefore := now.Add(7 * 24 * time.Hour)
	posts, err := s.q.ListRetentionWarningPosts(ctx, sqlc.ListRetentionWarningPostsParams{Now: now, WarnBefore: warnBefore})
	if err != nil {
		return unclaimed, warned, overdue, reminders, err
	}
	for _, p := range posts {
		if p.HoldingPostID == nil {
			continue
		}
		postID := *p.HoldingPostID
		officers, err := s.q.ListUserIDsForPosts(ctx, []uuid.UUID{postID})
		if err != nil {
			return unclaimed, warned, overdue, reminders, err
		}
		for _, off := range officers {
			last, err := s.q.LastNotificationOf(ctx, sqlc.LastNotificationOfParams{RecipientID: off, Event: string(notify.CustodyRetentionAlert), EntityID: &postID})
			if err == nil && !last.Before(now.Add(-24*time.Hour)) {
				continue
			}
			err = db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
				return s.notifier.Send(ctx, tx, notify.Message{
					Event:      notify.CustodyRetentionAlert,
					To:         off,
					Link:       "/petugas/inventaris",
					Data:       map[string]string{"count": strconv.Itoa(int(p.ItemCount))},
					EntityType: "post",
					EntityID:   &postID,
				})
			})
			if err != nil {
				return unclaimed, warned, overdue, reminders, err
			}
			warned++
		}
	}

	// BR-FND-03 24h reminder
	due24, err := s.q.ListFinderCustodyDue24h(ctx, now.Add(-24*time.Hour))
	if err != nil {
		return unclaimed, warned, overdue, reminders, err
	}
	for _, fr := range due24 {
		if fr.FinderID == nil {
			continue
		}
		err = db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
			return s.notifier.Send(ctx, tx, notify.Message{
				Event:      notify.FoundReminder24h,
				To:         *fr.FinderID,
				Link:       "/laporan/temuan/" + fr.Code,
				Data:       map[string]string{"item_name": fr.ItemName, "post_name": fr.PostName},
				EntityType: "found_report",
				EntityID:   &fr.ID,
			})
		})
		if err != nil {
			return unclaimed, warned, overdue, reminders, err
		}
		reminders++
	}

	// BR-FND-03 48h reminder and officer alert
	due48, err := s.q.ListFinderCustodyDue48h(ctx, now.Add(-48*time.Hour))
	if err != nil {
		return unclaimed, warned, overdue, reminders, err
	}
	for _, fr := range due48 {
		err = db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
			if fr.FinderID != nil {
				if err := s.notifier.Send(ctx, tx, notify.Message{
					Event:      notify.FoundReminder48h,
					To:         *fr.FinderID,
					Link:       "/laporan/temuan/" + fr.Code,
					Data:       map[string]string{"post_name": fr.PostName},
					EntityType: "found_report",
					EntityID:   &fr.ID,
				}); err != nil {
					return err
				}
			}
			if fr.IntendedPostID != nil {
				q := s.q.WithTx(tx)
				officers, err := q.ListUserIDsForPosts(ctx, []uuid.UUID{*fr.IntendedPostID})
				if err != nil {
					return err
				}
				for _, off := range officers {
					if err := s.notifier.Send(ctx, tx, notify.Message{
						Event:      notify.FoundOverdue,
						To:         off,
						Link:       "/petugas/temuan/" + fr.Code,
						Data:       map[string]string{"found_code": fr.Code},
						EntityType: "found_report",
						EntityID:   &fr.ID,
					}); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return unclaimed, warned, overdue, reminders, err
		}
		reminders++
	}

	// BR-FND-03 72h mark overdue
	due72, err := s.q.ListFinderCustodyDue72h(ctx, now.Add(-72*time.Hour))
	if err != nil {
		return unclaimed, warned, overdue, reminders, err
	}
	for _, fr := range due72 {
		err = db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
			q := s.q.WithTx(tx)
			_, err := q.MarkFoundCustodyOverdue(ctx, sqlc.MarkFoundCustodyOverdueParams{ID: fr.ID, Version: fr.Version, Now: now})
			if db.IsNoRows(err) {
				return nil
			}
			return err
		})
		if err != nil {
			return unclaimed, warned, overdue, reminders, err
		}
		overdue++
	}

	return unclaimed, warned, overdue, reminders, nil
}

func (s *Service) markUnclaimed(ctx context.Context, id uuid.UUID, version int32, postID *uuid.UUID, code string) (bool, error) {
	done := false
	now := s.clock.Now()
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		_, err := q.MarkFoundUnclaimed(ctx, sqlc.MarkFoundUnclaimedParams{ID: id, Version: version, Now: now})
		if db.IsNoRows(err) {
			return nil
		}
		if err != nil {
			return err
		}
		from := "reported"
		if err := q.InsertStatusHistory(ctx, sqlc.InsertStatusHistoryParams{
			EntityType: "found_report", EntityID: id, FromStatus: &from, ToStatus: "unclaimed",
			Transition: "F10", ActorKind: "system", CreatedAt: now,
		}); err != nil {
			return err
		}
		done = true
		if postID != nil {
			officers, err := q.ListUserIDsForPosts(ctx, []uuid.UUID{*postID})
			if err != nil {
				return err
			}
			for _, off := range officers {
				if err := s.notifier.Send(ctx, tx, notify.Message{
					Event:      notify.CustodyUnclaimed,
					To:         off,
					Link:       "/petugas/temuan/" + code,
					Data:       map[string]string{"found_code": code},
					EntityType: "found_report",
					EntityID:   &id,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return done, err
}
