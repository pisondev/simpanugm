# Observability | SimpanUGM

What the system tells us about itself, and what we promise about it.

## 1. Logs

JSON lines on standard output from every service, collected by Docker's log driver.

| Field | Present in | Meaning |
|---|---|---|
| `ts` | all | RFC 3339 with milliseconds, UTC |
| `level` | all | `debug`, `info`, `warn`, `error` |
| `service` | all | `web`, `api`, `worker`, `ml`, `caddy` |
| `version` | all | release version and commit |
| `request_id` | requests and jobs | from `X-Request-Id`, copied into job metadata |
| `user_id`, `role` | authenticated requests | ids only, never names or emails |
| `route`, `method`, `status`, `duration_ms` | access logs | route pattern, not the raw path, so codes in URLs are not logged |
| `job_kind`, `job_id`, `attempt` | worker | |
| `error`, `error_code` | failures | technical cause and problem code |
| `msg` | all | short, constant text per event, so logs can be counted |

Redacted keys: `password`, `token`, `code`, `cookie`, `authorization`, `hidden_details`,
`private_details`, `answers`, `statement`, `description`. A unit test feeds each logger a
record with these keys and asserts the output contains none of the values.

Retention on the host: 5 files of 20 MB per container. Logs are for debugging, the audit
log is the record.

## 2. Metrics

Prometheus scrapes every 15 s on the internal network.

### 2.1 `api`

| Metric | Type | Labels |
|---|---|---|
| `http_requests_total` | counter | `route`, `method`, `status` |
| `http_request_duration_seconds` | histogram | `route`, `method` |
| `http_inflight_requests` | gauge | |
| `sse_connections` | gauge | |
| `auth_login_total` | counter | `result` (`ok`, `bad_credentials`, `locked`, `rate_limited`) |
| `rate_limited_total` | counter | `limit` |
| `db_pool_acquired_conns`, `db_pool_idle_conns` | gauge | |
| `domain_transitions_total` | counter | `machine`, `transition` |
| `ml_request_duration_seconds` | histogram | `endpoint`, `status` |

### 2.2 `worker`

| Metric | Type | Labels |
|---|---|---|
| `river_jobs_total` | counter | `kind`, `state` (`completed`, `retryable`, `discarded`) |
| `river_job_duration_seconds` | histogram | `kind` |
| `river_queue_depth` | gauge | `queue` |
| `match_pipeline_seconds` | histogram | from report creation to stored candidates, NFR-PERF-03 |
| `match_candidates_stored` | histogram | per `match.report` run |
| `match_top1_score` | histogram | best score per new report, feeds AI-12 |
| `emails_sent_total` | counter | `event`, `result` |

### 2.3 `ml`

| Metric | Type | Labels |
|---|---|---|
| `http_request_duration_seconds` | histogram | `handler`, `status` |
| `ml_request_duration_seconds` | histogram | `handler`, `status` |
| `ml_inference_seconds` | histogram | `component` |
| `ml_models_loaded` | gauge | `component`, `revision` |
| `process_resident_memory_bytes` | gauge | |

PostgreSQL metrics come from `postgres_exporter` in the observability profile: connections,
cache hit ratio, slow queries through `pg_stat_statements`, database size.

## 3. Dashboards

Provisioned from `deploy/grafana/dashboards/` as JSON in git.

| Dashboard | Panels |
|---|---|
| Overview | request rate, error rate, p50 and p95 latency per route group, SSE connections, sign in results |
| Jobs and matching | queue depth per queue, job outcomes, match pipeline p50 and p95, inference latency per component, top 1 score histogram |
| Database | connections, slow queries, cache hit ratio, size, replication of backups |
| Host | CPU, memory, disk, container restarts |

The AI quality view for administrators lives in the product (AI-12), not in Grafana,
because it needs domain data and role based access.

## 4. Health checks

| Service | `/healthz` | `/readyz` |
|---|---|---|
| `api` | process up | database reachable, migrations at the expected version, settings loaded, `ml` model versions match the active rows |
| `worker` | process up | database reachable, River client running and leader election healthy |
| `ml` | process up | every model loaded and warmed |
| `web` | process up | `api` `/healthz` reachable |

Compose uses `/readyz` as the container health check. Caddy only routes to healthy
upstreams.

## 5. Service level objectives

Measured monthly in production.

| SLO | Objective | Indicator |
|---|---|---|
| Availability | 99.5 percent | share of 1 minute windows where the synthetic check of `/api/healthz` through Caddy and the landing page both succeed |
| API latency | 95 percent of read requests under 300 ms | `http_request_duration_seconds` for `GET` routes |
| Matching freshness | 95 percent of new reports have candidates within 60 s | `match_pipeline_seconds` |
| Email delivery | 95 percent sent within 2 minutes of the event | notification `created_at` to `emails_sent_total` timing |

Error budget for availability: about 3.6 hours a month. When half of it is spent, feature
work pauses for reliability work until the next month.

## 6. Alerts

Prometheus rules in `deploy/prometheus/alerts.yml`, delivered to the team chat through
Grafana alerting.

| Alert | Condition | Severity |
|---|---|---|
| `SiteDown` | synthetic check failing for 2 minutes | page |
| `HighErrorRate` | 5xx above 2 percent of requests for 5 minutes | page |
| `SlowAPI` | p95 read latency above 600 ms for 10 minutes | warn |
| `MatchingStalled` | `ml` queue depth above 50 for 10 minutes, or no completed `match.report` for 30 minutes while reports were created | page |
| `MLNotReady` | `ml` `/readyz` failing for 5 minutes | warn |
| `JobsDiscarded` | any discarded job in 15 minutes | warn |
| `DiskFilling` | disk above 80 percent | warn, above 90 percent page |
| `BackupMissing` | no successful backup in 26 hours | page |
| `CertExpiry` | certificate expiring within 14 days | warn |

"Page" means a message that mentions the member on call for the week (team section 5).
