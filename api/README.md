# api

The Go core of SimpanUGM: HTTP API, authentication and authorisation, every state
machine, persistence, background jobs, notifications. One binary, `simpan`, with the
commands `serve`, `worker`, `migrate`, `seed` and `admin`.

Owner: Pison Golda. Status: specified, implementation starts in Sprint 1.

| Read first | For |
|---|---|
| [docs/architecture/overview.md](../docs/architecture/overview.md) | modules and runtime flows |
| [docs/architecture/api.md](../docs/architecture/api.md) | the endpoint catalog that becomes `openapi.yaml` |
| [docs/architecture/data-model.md](../docs/architecture/data-model.md) | every table and migration rule |
| [docs/requirements/state-machines.md](../docs/requirements/state-machines.md) | the transitions each module implements |
| [docs/requirements/permissions.md](../docs/requirements/permissions.md) | authorisation |
| [docs/engineering/repository-structure.md](../docs/engineering/repository-structure.md) | the layout of this folder |

Stack: Go 1.26, chi, oapi-codegen, pgx, sqlc, goose, River, pgvector-go. Full list in
[docs/engineering/tech-stack.md](../docs/engineering/tech-stack.md).
