# deploy

Everything needed to run SimpanUGM on one host: Docker Compose files for development and
production, the Caddy configuration, environment templates, Prometheus and Grafana
provisioning, and the backup scripts.

Owner: Pison Golda. Status: specified, files land in Sprint 1 (development stack) and
Sprint 4 (production, monitoring, backups).

| Read first | For |
|---|---|
| [docs/engineering/local-development.md](../docs/engineering/local-development.md) | ports, environment variables, make targets |
| [docs/operations/deployment.md](../docs/operations/deployment.md) | environments, pipeline, server layout, Caddy |
| [docs/operations/observability.md](../docs/operations/observability.md) | metrics, dashboards, alerts |
| [docs/operations/runbook.md](../docs/operations/runbook.md) | backups, restore, incidents |

Real `.env` files never enter git. Only `*.env.example` templates do.
