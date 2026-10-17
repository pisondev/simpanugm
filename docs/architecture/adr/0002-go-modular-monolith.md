# ADR-0002 Go modular monolith with a separate worker process

Status: Accepted, 2026-10-17

## Context

The domain has strong consistency needs: a handover changes a claim, two reports, custody,
competing claims, the audit log and notifications, and all of that must commit together.
The team has nine weeks and one server. The lecturer expects a production ready system,
which means operable, observable and testable, not necessarily distributed.

## Decision

One Go binary, `simpan`, organised as modules under `api/internal/<module>/` with explicit
service interfaces and table ownership (overview section 4). It runs as two processes from
the same image: `simpan serve` for HTTP and `simpan worker` for background jobs.

## Consequences

- Use cases are single database transactions. No distributed transactions, no sagas, no
  eventual consistency between our own components.
- HTTP latency is isolated from heavy jobs because they run in another process, while
  sharing all domain code.
- Module boundaries are enforced by convention and review, backed by a lint rule that
  forbids importing another module's `store` package.
- If one module ever needs to scale apart, its service interface is already the seam to
  extract it.

## Alternatives

- **Microservices per module.** Network calls and partial failures inside what is one
  business transaction, a message broker, and far more deployment work. No load that
  justifies it.
- **A single process for HTTP and jobs.** Simpler, but a burst of matching jobs would
  compete with request handling for CPU, and a crash loop in a job would take the API down.
