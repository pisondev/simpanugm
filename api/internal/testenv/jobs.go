package testenv

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// Drain works the due jobs of one kind queued since the Env was built with w, oldest first, the way a River worker would, and records each outcome in
// river_job so a later drain sees only what is due again. It returns one result per job, nil for a completed job
func Drain[T river.JobArgs](e *Env, w river.Worker[T]) []error {
	e.T.Helper()
	ctx := context.Background()
	var zero T
	kind := zero.Kind()
	rows, err := e.Pool.Query(ctx, `SELECT id, args, attempt, created_at FROM river_job
		WHERE kind = $1 AND created_at >= $2 AND (state = 'available' OR (state IN ('scheduled', 'retryable') AND scheduled_at <= now()))
		ORDER BY id`, kind, e.Started)
	if err != nil {
		e.T.Fatalf("testenv: due %s jobs: %v", kind, err)
	}
	type due struct {
		id      int64
		args    []byte
		attempt int
		created time.Time
	}
	var jobs []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.args, &d.attempt, &d.created); err != nil {
			e.T.Fatalf("testenv: scan job: %v", err)
		}
		jobs = append(jobs, d)
	}
	rows.Close()
	out := make([]error, 0, len(jobs))
	for _, d := range jobs {
		var args T
		if err := json.Unmarshal(d.args, &args); err != nil {
			e.T.Fatalf("testenv: decode %s args: %v", kind, err)
		}
		job := &river.Job[T]{JobRow: &rivertype.JobRow{ID: d.id, Kind: kind, Attempt: d.attempt + 1, CreatedAt: d.created, State: rivertype.JobStateRunning}, Args: args}
		werr := w.Work(ctx, job)
		var snooze *river.JobSnoozeError
		var cancel *river.JobCancelError
		var q string
		params := []any{d.id}
		switch {
		case werr == nil:
			q = `UPDATE river_job SET state = 'completed', attempt = attempt + 1, finalized_at = now() WHERE id = $1`
		case errors.As(werr, &snooze):
			q = `UPDATE river_job SET state = 'scheduled', scheduled_at = now() + make_interval(secs => $2) WHERE id = $1`
			params = append(params, snooze.Duration.Seconds())
		case errors.As(werr, &cancel):
			q = `UPDATE river_job SET state = 'cancelled', attempt = attempt + 1, finalized_at = now() WHERE id = $1`
		default:
			q = `UPDATE river_job SET state = 'retryable', attempt = attempt + 1, scheduled_at = now() WHERE id = $1`
		}
		if _, err := e.Pool.Exec(ctx, q, params...); err != nil {
			e.T.Fatalf("testenv: record %s outcome: %v", kind, err)
		}
		out = append(out, werr)
	}
	return out
}

// how many jobs of a kind wait in a state, for assertions on what an action queued
func (e *Env) JobCount(kind string, states ...string) int {
	e.T.Helper()
	if len(states) == 0 {
		states = []string{"available", "scheduled", "retryable"}
	}
	var n int
	if err := e.Pool.QueryRow(context.Background(), "SELECT count(*) FROM river_job WHERE kind = $1 AND state = ANY($2) AND created_at >= $3", kind, states, e.Started).Scan(&n); err != nil {
		e.T.Fatalf("testenv: count %s: %v", kind, err)
	}
	return n
}
