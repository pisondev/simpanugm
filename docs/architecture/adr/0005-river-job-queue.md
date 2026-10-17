# ADR-0005 PostgreSQL backed job queue with River

Status: Accepted, 2026-10-17

## Context

Background work: image variants, inference calls, matching, notifications, emails,
reminders at computed times, retention, nightly sweeps. Jobs must be enqueued atomically
with the change that causes them, retried with backoff, scheduled for a future time, made
unique, and inspected when they fail.

## Decision

Use River (`github.com/riverqueue/river`) with the pgx driver. Jobs are inserted with
`InsertTx` inside the domain transaction. Queues `default`, `ml`, `email`, `maintenance`
with fixed concurrency. Periodic jobs for sweeps and retention. Unique jobs for
`match.report` per report.

## Consequences

- No outbox table and no broker: the job exists if and only if the transaction committed.
- The queue lives in the same database and the same backup.
- Throughput is bounded by PostgreSQL, which is far above our need of a few thousand jobs
  a day.
- Job handlers must be idempotent because delivery is at least once (NFR-AVAIL-03).

## Alternatives

- **Redis with Asynq.** Another stateful service, and enqueueing is no longer atomic with
  the database write.
- **A hand written queue with `SELECT ... FOR UPDATE SKIP LOCKED`.** Small to start, but
  retries, scheduling, uniqueness, periodic jobs and a failure view would all be ours to
  build and test.
