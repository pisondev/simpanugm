package reports

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
)

type SweepArgs struct{}

func (SweepArgs) Kind() string { return "reports.sweep" }

func (SweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "maintenance", MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByPeriod: 10 * time.Minute}}
}

type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	S *Service
}

func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	reminded, expired, err := w.S.SweepDue(ctx)
	if reminded+expired > 0 {
		slog.InfoContext(ctx, "lost reports swept", "reminded", reminded, "expired", expired)
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

// BR-LST-02 stale reminder and BR-LST-03 expiry
func (s *Service) SweepDue(ctx context.Context) (reminded, expired int, err error) {
	now := s.clock.Now()

	// BR-LST-02 stale reminder
	staleDays := s.settings.LostStaleDays(ctx)
	staleBefore := now.Add(-time.Duration(staleDays) * 24 * time.Hour)
	stale, err := s.q.ListStaleLostReports(ctx, staleBefore)
	if err != nil {
		return 0, 0, err
	}
	for _, lr := range stale {
		ok, err := s.remindStale(ctx, lr.ID, lr.Code, lr.ItemName, lr.ReporterID)
		if err != nil {
			return reminded, expired, err
		}
		if ok {
			reminded++
		}
	}

	// BR-LST-03 expiry after grace period without answer
	graceDays := s.settings.LostExpiryGraceDays(ctx)
	expiredBefore := now.Add(-time.Duration(graceDays) * 24 * time.Hour)
	expList, err := s.q.ListExpiredLostReports(ctx, expiredBefore)
	if err != nil {
		return reminded, expired, err
	}
	for _, lr := range expList {
		ok, err := s.expireReport(ctx, lr.ID, lr.Version, lr.Code, lr.ReporterID)
		if err != nil {
			return reminded, expired, err
		}
		if ok {
			expired++
		}
	}

	return reminded, expired, nil
}

func (s *Service) remindStale(ctx context.Context, id uuid.UUID, code, itemName string, reporterID *uuid.UUID) (bool, error) {
	if reporterID == nil {
		return false, nil
	}
	now := s.clock.Now()
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.MarkLostStaleReminded(ctx, sqlc.MarkLostStaleRemindedParams{ID: id, Now: now}); err != nil {
			return err
		}
		return s.notifier.Send(ctx, tx, notify.Message{
			Event:      notify.LostStaleReminder,
			To:         *reporterID,
			Link:       "/laporan/hilang/" + code,
			Data:       map[string]string{"item_name": itemName},
			EntityType: "lost_report",
			EntityID:   &id,
		})
	})
	return err == nil, err
}

func (s *Service) expireReport(ctx context.Context, id uuid.UUID, version int32, code string, reporterID *uuid.UUID) (bool, error) {
	done := false
	now := s.clock.Now()
	err := db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		_, err := q.ExpireLostReport(ctx, sqlc.ExpireLostReportParams{ID: id, Version: version, Now: now})
		if db.IsNoRows(err) {
			return nil
		}
		if err != nil {
			return err
		}
		from := "reported"
		reason := "expired"
		if err := q.InsertStatusHistory(ctx, sqlc.InsertStatusHistoryParams{
			EntityType: "lost_report", EntityID: id, FromStatus: &from, ToStatus: "closed",
			Transition: "L11", Reason: &reason, ActorKind: "system", CreatedAt: now,
		}); err != nil {
			return err
		}
		if err := s.matching.ReportClosed(ctx, tx, "lost", id); err != nil {
			return err
		}
		if err := audit.Record(ctx, q, now, audit.Entry{
			Action:     "lost_report.expire",
			EntityType: "lost_report",
			EntityID:   id,
			Changes:    map[string]any{"status": audit.Change("reported", "closed"), "close_reason": "expired"},
		}); err != nil {
			return err
		}
		done = true
		if reporterID != nil {
			return s.notifier.Send(ctx, tx, notify.Message{
				Event:      notify.LostExpired,
				To:         *reporterID,
				Link:       "/laporan/hilang/" + code,
				Data:       map[string]string{"lost_code": code},
				EntityType: "lost_report",
				EntityID:   &id,
			})
		}
		return nil
	})
	return done, err
}
