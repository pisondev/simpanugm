# deploy

Everything needed to run SimpanUGM on one host: Docker Compose files for development and
production, the Caddy configuration, environment templates, Prometheus and Grafana
provisioning, and the backup scripts.

Owner: Pison Golda. Status: the development stack runs, production overrides, monitoring
and backups land in Sprint 4.

```
make env    # deploy/env/*.env with random secrets
make dev    # build, migrate, seed, start, wait for health, then open http://localhost:8000
```

| File | Holds |
|---|---|
| `compose.yaml` | postgres, migrate, seed, api, worker, ml, web, caddy, with health checks and start order |
| `compose.dev.yaml` | published ports, Mailpit, the `dev-seed` overlay |
| `caddy/Caddyfile` | one origin: `/api/*` to api, the rest to web, request ids, security headers, body limits |
| `postgres/` | roles, database and extensions on first start, also used by the integration tests |
| `dev-seed/` | starter zones, landmarks and posts until the reviewed OpenStreetMap seed lands in `data/seed` |
| `env/*.env.example` | every variable of docs/engineering/local-development.md section 6 |

| Read first | For |
|---|---|
| [docs/engineering/local-development.md](../docs/engineering/local-development.md) | ports, environment variables, make targets |
| [docs/operations/deployment.md](../docs/operations/deployment.md) | environments, pipeline, server layout, Caddy |
| [docs/operations/observability.md](../docs/operations/observability.md) | metrics, dashboards, alerts |
| [docs/operations/runbook.md](../docs/operations/runbook.md) | backups, restore, incidents |

Real `.env` files never enter git. Only `*.env.example` templates do.
