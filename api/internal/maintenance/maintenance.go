package maintenance

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/auth"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/media"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
)

// BR-RET-06
const notificationRetentionDays = 180

type PurgeArgs struct{}

func (PurgeArgs) Kind() string { return "maintenance.purge" }

func (PurgeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "maintenance", MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Minute}}
}

// BR-MED-02, BR-ACC-02, BR-RET-06 and expired idempotency keys
type PurgeWorker struct {
	river.WorkerDefaults[PurgeArgs]
	Media *media.Service
	Auth  *auth.Service
	Q     *sqlc.Queries
	Clock clock.Clock
}

func (w *PurgeWorker) Work(ctx context.Context, _ *river.Job[PurgeArgs]) error {
	staged, err := w.Media.PurgeStaged(ctx)
	if err != nil {
		return err
	}
	keys, err := w.Q.DeleteExpiredIdempotencyKeys(ctx, w.Clock.Now())
	if err != nil {
		return err
	}
	users, sessions, err := w.Auth.Purge(ctx)
	if err != nil {
		return err
	}
	notifications, err := w.Q.DeleteOldNotifications(ctx, w.Clock.Now().AddDate(0, 0, -notificationRetentionDays))
	if err != nil {
		return err
	}
	summary := map[string]any{"staged_media": staged, "idempotency_keys": keys, "unverified_users": users, "expired_sessions": sessions, "notifications": notifications}
	slog.InfoContext(ctx, "maintenance purge done", "summary", summary)
	return audit.Record(ctx, w.Q, w.Clock.Now(), audit.Entry{Action: "maintenance.purge", EntityType: "system", Changes: summary})
}

func Periodic() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(time.Hour), func() (river.JobArgs, *river.InsertOpts) {
			return PurgeArgs{}, nil
		}, &river.PeriodicJobOpts{RunOnStart: true}),
	}
}
