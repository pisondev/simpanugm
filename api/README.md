# api

The Go core of SimpanUGM: HTTP API, authentication and authorisation, the state machines,
persistence, background jobs and notifications. One binary, `simpan`:

| Command | Does |
|---|---|
| `simpan serve` | HTTP API on `HTTP_ADDR`, metrics and health on `METRICS_ADDR` |
| `simpan worker` | River jobs: email, maintenance purge |
| `simpan migrate [up\|down\|status]` | goose migrations, then River's own, as the owner role |
| `simpan seed [--dir data/seed] [--overlay deploy/dev-seed] [--demo]` | master data, optional development overlay and demo accounts, both refused in production |
| `simpan admin create --email e --name n` | first administrator, prints the set password link |
| `simpan healthcheck URL` | exit 0 when URL answers 200, for container health checks |

Owner: Pison Golda. Read first: [api.md](../docs/architecture/api.md),
[data-model.md](../docs/architecture/data-model.md),
[state-machines.md](../docs/requirements/state-machines.md),
[permissions.md](../docs/requirements/permissions.md).

## Implemented in v0.1

Accounts (registration on UGM domains, verification, sign in, sessions, password reset,
invitations, user administration), master data with ETags, photo upload with metadata
stripping and four variants, lost and found reports with idempotent creation, editing,
closing, history and audit, the public found listing, proxy reports for walk ins, custody
intake with retention, post inventory, and officer search.

## Layout

```
cmd/simpan/          commands
internal/
  platform/          config, db, httpx, storage, mail, ratelimit, settings, idem, cursor, logx, metrics
  access/            roles, permissions, the rule of every operation
  auth/ masterdata/ media/ reports/ custody/ search/ audit/ notify/ maintenance/ privacy/
  server/            router, contract middleware, integration tests
  openapi/           generated from openapi.yaml, never edited
  db/sqlc/           generated from migrations and queries.sql files, never edited
  testenv/           PostgreSQL in a container with the real init script, for integration tests
migrations/          numbered goose migrations, embedded in the binary
openapi.yaml         the public contract
```

## Working on it

Requires Go 1.26 or newer and Docker.

```
go build ./...
go test ./...                                   # unit tests
go test -tags integration ./internal/server/    # starts pgvector/pgvector:pg18 through testcontainers
golangci-lint run ./...
go generate ./internal/openapi/                 # after editing openapi.yaml
docker run --rm -v "$PWD:/src" -w /src sqlc/sqlc:1.31.1 generate   # after editing SQL
```

The integration tests run the same `deploy/postgres/init.sql` as Compose, apply every
migration as the owner role, seed `data/seed` with the `deploy/dev-seed` overlay, then
exercise the real router over HTTP as the application role.

The image is built from the repository root so the seed files can be copied in:
`docker build -f api/Dockerfile .`

Build and test with `-tags nodynamic`. Without it `gen2brain/webp` links libc through
purego to look for a system libwebp, the binary becomes dynamic and fails to start on
`distroless/static`. With it every environment uses the same WebAssembly encoder.
