# Local Development | SimpanUGM

The whole system runs locally with Docker Compose. Services that you are actively
changing can run on the host with live reload while the rest stay in containers.

> The commands below become available in Sprint 1, when `Makefile`, `deploy/compose.yaml`
> and the service skeletons land.

## 1. Prerequisites

| Tool | Version | Check |
|---|---|---|
| Git | 2.40 or newer | `git --version` |
| Docker Desktop with the WSL 2 backend, or Docker Engine on Linux | Engine 27 or newer, Compose v2 | `docker compose version` |
| Go | 1.26 or newer. Older toolchains must be upgraded | `go version` |
| Node.js | 22 LTS | `node --version` |
| pnpm | 10, through Corepack | `corepack enable && pnpm --version` |
| Python | 3.12 | `python --version` |
| uv | latest | `uv --version` |
| make | GNU make 4 | `make --version` |

On Windows, work inside a WSL 2 Ubuntu distribution with the repository cloned into the
Linux file system (`~/code/simpanugm`), not under `/mnt/c`. File watching and Docker bind
mounts are an order of magnitude faster there, and `make` is available out of the box.

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
| web | <http://localhost:3000> | direct, for debugging only |
| api | <http://localhost:8090> | direct, `/healthz`, `/readyz` |
| api metrics | <http://localhost:9091/metrics> | |
| ml | <http://localhost:8100/docs> | FastAPI docs, needs the dev token |
| PostgreSQL | `localhost:5433` | 5433 so it does not collide with a PostgreSQL installed on the host |
| Mailpit | <http://localhost:8025> | every email sent locally lands here |
| Prometheus | <http://localhost:9090> | profile `observability` |
| Grafana | <http://localhost:3001> | profile `observability`, admin password in `deploy/env/grafana.env` |

Port 8080 is avoided on purpose because it is commonly taken by other local services.

## 4. Make targets

| Target | Does |
|---|---|
| `make env` | create local env files with random secrets |
| `make dev` | compose up with dev overrides, migrate, seed |
| `make down` | stop the stack, keep volumes |
| `make reset` | stop and delete volumes, then `make dev` |
| `make logs s=api` | follow logs of one service |
| `make migrate`, `make migrate-down` | apply or roll back one migration |
| `make seed`, `make seed-demo` | master data, demo data |
| `make gen` | regenerate sqlc code, Go server from `openapi.yaml`, TypeScript client, `ml/openapi.json` |
| `make check` | lint, typecheck and unit tests for every area, the pre push gate |
| `make test-api`, `make test-web`, `make test-ml`, `make test-data` | one area including integration tests |
| `make e2e` | Playwright against a fresh stack |
| `make eval-data`, `make eval-check`, `make eval` | evaluation dataset download, checks, run |
| `make load` | k6 scenarios against the local stack |
| `make obs` | start Prometheus and Grafana |

## 5. Working on one service with live reload

| Service | Command | What changes |
|---|---|---|
| api | `make dev-api` | stops the `api` and `worker` containers and runs both on the host with `air` against the containerised database |
| web | `make dev-web` | stops the `web` container and runs `pnpm dev` on the host. Caddy proxies to the host |
| ml | `make dev-ml` | stops the `ml` container and runs Uvicorn with reload on the host, reusing the downloaded weights |

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
| Port 5433 in use | set `POSTGRES_HOST_PORT` in `deploy/env/compose.env` |
