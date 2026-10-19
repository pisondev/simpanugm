# Local Development | SimpanUGM

The whole system runs locally with Docker Compose. Services that you are actively
changing can run on the host with live reload while the rest stay in containers.

> Available since v0.1.0. Targets that belong to later sprints are marked with their sprint.

## 1. Prerequisites

| Tool | Version | Check |
|---|---|---|
| Git | 2.40 or newer | `git --version` |
| Docker Desktop with the WSL 2 backend, or Docker Engine on Linux | Engine 27 or newer, Compose v2 | `docker compose version` |
| Go | 1.26 or newer. Older toolchains must be upgraded | `go version` |
| Node.js | 22 LTS, 22.22.2 or newer, the floor of jsdom 30 | `node --version` |
| pnpm | 10, through Corepack | `corepack enable && pnpm --version` |
| Python | 3.12 | `python --version` |
| uv | latest | `uv --version` |
| make | GNU make 4 | `make --version` |

On Windows, work inside a WSL 2 Ubuntu distribution with the repository cloned into the
Linux file system (`~/code/simpanugm`), not under `/mnt/c`. File watching and Docker bind
mounts are an order of magnitude faster there, and `make` is available out of the box.
Native Windows also works: install GNU make with `winget install ezwinports.make` and run
the targets from Git Bash. The whole Sprint 1 stack was verified that way.

Disk: about 6 GB for images and model weights after the first `make dev`.

## 2. First run

```
git clone https://github.com/pisondev/simpanugm.git
cd simpanugm
make env          # copies deploy/env/*.env.example to deploy/env/*.env with random dev secrets
make dev          # builds and starts the stack, migrates, seeds master data
make seed-demo    # optional: demo accounts and reports, prints the generated passwords
```

Open <http://localhost:8000>. The first start of `ml` downloads the pinned weights into the
`ml-models` volume, which takes a few minutes, and `/readyz` stays red until it finishes.
Report creation works meanwhile, matching catches up.

## 3. Services and ports

| Service | URL on the host | Notes |
|---|---|---|
| Caddy, the app | <http://localhost:8000> | same origin as production, use this, not the ports below |
| web | <http://localhost:3000> | direct, for debugging only, `WEB_HOST_PORT` moves it |
| api | <http://localhost:8090> | direct, `/healthz`, `/readyz` |
| api metrics | <http://localhost:9091/metrics> | |
| ml | <http://localhost:8100/docs> | FastAPI docs, needs the dev token |
| PostgreSQL | `localhost:5433` | 5433 so it does not collide with a PostgreSQL installed on the host, `POSTGRES_HOST_PORT` moves it |
| Mailpit | <http://localhost:8025> | every email sent locally lands here |
| Prometheus | <http://localhost:9090> | profile `observability`, Sprint 4 |
| Grafana | <http://localhost:3001> | profile `observability`, admin password in `deploy/env/grafana.env`, Sprint 4 |

Port 8080 is avoided on purpose because it is commonly taken by other local services.

## 4. Make targets

| Target | Does |
|---|---|
| `make env` | create `deploy/env/*.env` from the templates with random secrets, existing files are kept |
| `make dev` | build and start the stack with dev overrides, migrate, seed with the dev overlay, wait for health |
| `make up`, `make down`, `make ps` | start without building, stop keeping volumes, list services |
| `make reset` | stop and delete volumes, then `make dev` |
| `make logs s=api` | follow logs of one service |
| `make migrate`, `make migrate-down`, `make migrate-status` | apply all, roll back one, show the state |
| `make seed`, `make seed-demo` | master data, plus demo accounts whose passwords are printed once |
| `make admin email=... name=...` | the first administrator, prints the link to set the password |
| `make gen` | regenerate sqlc code, the Go server from `openapi.yaml`, the TypeScript client |
| `make check` | format, lint, types and unit tests for every area, the pre push gate |
| `make test-api`, `make test-api-integration`, `make test-web`, `make test-ml`, `make test-data` | one area |
| `make e2e` | apply `deploy/compose.e2e.yaml` override (`APP_ENV=e2e`, fake clock), restart `api`, then Playwright journeys |
| `make eval-data`, `make eval-check`, `make eval` | evaluation dataset download, checks, run. Sprint 2 |
| `make load` | k6 scenarios against the local stack. Sprint 3 |
| `make obs` | start Prometheus and Grafana. Sprint 4 |

## 5. Working on one service with live reload

| Service | Command | What changes |
|---|---|---|
| web | `docker compose ... stop web`, then `API_INTERNAL_URL=http://localhost:8090 pnpm dev` in `web/` | the dev server on <http://localhost:3000> proxies `/api` to the published api port |
| api | `go run -tags nodynamic ./cmd/simpan serve` in `api/` with the variables of section 6.1 pointing at `localhost` | stop the `api` container first. `make dev-api` with `air` follows in Sprint 2 |
| ml | `uv run uvicorn app.main:create_app --factory --reload --port 8100` in `ml/` | stop the `ml` container first. `make dev-ml` follows in Sprint 2 |

## 6. Environment variables

Every variable is read once at start. A missing required variable stops the service with
a message naming it. Templates live in `deploy/env/*.env.example`.

### 6.1 `api` and `worker`

| Variable | Required | Example | Meaning |
|---|---|---|---|
| `APP_ENV` | yes | `development` | `development`, `test`, `e2e`, `staging`, `production` |
| `HTTP_ADDR` | no | `:8090` | listen address of `simpan serve` |
| `METRICS_ADDR` | no | `:9091` | Prometheus endpoint |
| `PUBLIC_BASE_URL` | yes | `http://localhost:8000` | origin for links in emails and the CSRF origin check |
| `DATABASE_URL` | yes | `postgres://simpan_app:...@postgres:5432/simpan` | application role |
| `DATABASE_MIGRATOR_URL` | for `migrate` | `postgres://simpan_owner:...@postgres:5432/simpan` | owner role |
| `DATABASE_MAX_CONNS` | no | `20` | pool size per process |
| `STORAGE_DRIVER` | yes | `fs` | `fs` or `s3` |
| `STORAGE_FS_ROOT` | with `fs` | `/var/lib/simpan/media` | mounted volume |
| `S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`, `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY` | with `s3` | | S3 compatible store |
| `ML_BASE_URL` | yes | `http://ml:8100` | inference service |
| `ML_INTERNAL_TOKEN` | yes | 32 random bytes, base64 | shared with `ml` |
| `SMTP_HOST`, `SMTP_PORT` | yes | `mailpit`, `1025` | |
| `SMTP_USERNAME`, `SMTP_PASSWORD` | no | | |
| `SMTP_TLS` | no | `starttls` | `starttls`, `tls`, `none` |
| `MAIL_FROM` | yes | `SimpanUGM <no-reply@simpanugm.my.id>` | |
| `SESSION_COOKIE_SECURE` | no | `true` | false only in `development` over plain HTTP |
| `HANDOVER_PEPPER` | yes | 32 random bytes, base64 | BR-HND-01 |
| `WORKER_QUEUES` | no | `default=10,ml=2,email=2,maintenance=1` | River concurrency |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, `error` |
| `FEATURE_SEMANTIC_SEARCH` | no | `false` | AI-13 |
| `FEATURE_FAKE_CLOCK` | no | `false` | allowed only when `APP_ENV=e2e` |

### 6.2 `web`

| Variable | Required | Example | Meaning |
|---|---|---|---|
| `API_INTERNAL_URL` | yes | `http://api:8090` | server side calls |
| `NEXT_PUBLIC_BASE_URL` | yes | `http://localhost:8000` | canonical links |
| `NEXT_PUBLIC_MAP_TILE_URL` | no | `https://tile.openstreetmap.org/{z}/{x}/{y}.png` | map tiles |
| `LOG_LEVEL` | no | `info` | |

### 6.3 `ml`

| Variable | Required | Example | Meaning |
|---|---|---|---|
| `ML_INTERNAL_TOKEN` | yes | same as `api` | |
| `ML_MODELS_LOCK` | no | `/app/models.lock.json` | |
| `HF_HOME` | no | `/models` | weights cache on the `ml-models` volume |
| `ML_TORCH_THREADS` | no | `2` | `torch.set_num_threads` |
| `ML_MAX_IMAGE_BYTES` | no | `10485760` | |
| `ML_OCR_ENABLED` | no | `true` | switch OCR off on weak machines |
| `LOG_LEVEL` | no | `info` | |

## 7. Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Sign in works on `localhost:3000` but cookies vanish | use <http://localhost:8000>, the same origin setup depends on Caddy |
| `ml` stays unready | weights still downloading, `make logs s=ml`. Behind a proxy set `HTTPS_PROXY` in `deploy/env/ml.env` |
| Times look seven hours off | something formatted UTC without the zone. Display must go through `lib/format.ts`, which pins `Asia/Jakarta` |
| `make gen` produces a diff in CI | someone edited generated code or forgot to regenerate after changing `openapi.yaml` or a `queries.sql` |
| Port 5433, 3000 or 8000 in use | set `POSTGRES_HOST_PORT`, `WEB_HOST_PORT` or `HTTP_PORT` in `deploy/env/compose.env` |
| `exec /simpan: no such file or directory` | the image was built without `-tags nodynamic`, so the binary links libc. Rebuild with the Dockerfile as committed |
| Registration answers "Terlalu banyak percobaan" while testing | BR-SEC-01 allows 5 registrations per hour per IP and everything local shares one IP. `make e2e` restarts `api`, which resets the in memory limiters |
| `pnpm test` fails to start with an undici error | Node.js is older than 22.22.2 |
