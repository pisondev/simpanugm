# Architecture Overview | SimpanUGM

| | |
|---|---|
| Owner | Pison Golda |
| Style | Modular monolith in Go, a Next.js frontend, a stateless Python inference service, one PostgreSQL |
| Decisions | [adr/](adr/) |

## 1. Quality goals that shaped it

| Priority | Goal | Consequence |
|---|---|---|
| 1 | Correct and accountable handovers | All state in one transactional database. State machines enforced in one service with optimistic locking |
| 2 | Privacy of people and items | Authorisation and field projection centralised in `api`. Media never public. Inference on our own server |
| 3 | Buildable by three people in nine weeks | One deployable backend, one database for relational data, vectors and jobs. No message broker, no separate vector store |
| 4 | Operable on one small server | CPU models, Docker Compose, a single reverse proxy with automatic TLS |
| 5 | AI results that can be explained and reproduced | Scores computed in Go from stored components under a versioned configuration |

## 2. System context

```mermaid
flowchart TB
    owner([Owner, member])
    finder([Finder, member])
    officer([Officer at a post])
    admin([Administrator])
    sys[SimpanUGM]
    smtp[(SMTP provider)]
    hf[(Hugging Face Hub<br/>weights, first start only)]
    osm[(OpenStreetMap tiles)]

    owner -->|reports loss, reviews candidates, claims| sys
    finder -->|reports found item| sys
    officer -->|receives, verifies, hands over| sys
    admin -->|manages, analyses, monitors| sys
    sys -->|notification email| smtp
    sys -->|downloads pinned weights| hf
    owner -. map tiles in the browser .-> osm
    admin -. map tiles in the browser .-> osm
```

No other system holds or receives SimpanUGM data. Map tiles are fetched by the browser
and carry no report data.

## 3. Containers

```mermaid
flowchart LR
    browser[Browser]
    subgraph host[Single host, Docker Compose]
        caddy[caddy<br/>TLS, routing, compression]
        web[web<br/>Next.js 16, Node 22]
        api[api<br/>Go: simpan serve]
        worker[worker<br/>Go: simpan worker]
        ml[ml<br/>FastAPI, PyTorch CPU]
        pg[(postgres<br/>PostgreSQL 18 + pgvector)]
        media[(media volume<br/>or S3 bucket)]
        prom[prometheus]
        graf[grafana]
    end
    browser -->|HTTPS| caddy
    caddy -->|/| web
    caddy -->|/api/*| api
    web -->|SSR fetch, internal| api
    api --> pg
    worker --> pg
    api --> media
    worker --> media
    worker -->|embeddings, analysis| ml
    api -->|assistant, sync| ml
    worker -->|SMTP| smtpx[(SMTP)]
    prom --> api
    prom --> worker
    prom --> ml
    graf --> prom
```

| Container | Responsibility | Scales by |
|---|---|---|
| `caddy` | TLS termination with automatic certificates, HTTP/2 and HTTP/3, routing `/api/*` to `api` and the rest to `web`, gzip and zstd, security headers, request id generation | single instance |
| `web` | Every page, server side rendering, the route guards, the typed API client. No database access, no secrets beyond the internal API URL | stateless, more replicas |
| `api` | HTTP API, authentication, authorisation, domain logic, state machines, field projection, SSE, enqueueing jobs | stateless, more replicas |
| `worker` | River job processing: media variants, inference calls, matching, notifications, email, periodic jobs | more replicas, River coordinates |
| `ml` | Embeddings, image analysis, clustering. No database, no storage, no state beyond loaded weights | one per host, CPU bound |
| `postgres` | Relational data, pgvector indexes, River job tables, `LISTEN` and `NOTIFY`, materialised views for analytics | vertical |
| media | Photo variants and evidence behind the storage interface | volume or bucket |
| `prometheus`, `grafana` | Metrics and dashboards | single instance |

`api` and `worker` are one Go binary with two commands, so domain code is shared and
always deployed in lockstep. Other commands: `simpan migrate`, `simpan seed`,
`simpan admin create`.

## 4. Components inside `api`

Each module owns its tables and exposes a service interface. Modules call each other's
services, never each other's queries. A module may read another module's tables only
through a read model the owner publishes (for example `search` reads a view that
`reports` defines).

```mermaid
flowchart TB
    subgraph http[HTTP layer]
        mw[middleware:<br/>request id, logging, recover,<br/>session, CSRF origin check,<br/>rate limit, metrics]
        handlers[handlers per module<br/>generated from openapi.yaml]
    end
    subgraph domain[Modules]
        auth[auth]
        access[access]
        md[masterdata]
        reports[reports<br/>lost, found]
        mediam[media]
        matching[matching]
        assist[assist]
        claims[claims]
        custody[custody]
        notify[notify]
        search[search]
        officer[officer]
        admin[admin]
        audit[audit]
        analytics[analytics]
    end
    subgraph platform[Platform]
        db[db: pgx pool, sqlc, tx]
        jobs[jobs: River client]
        storage[storage: fs, s3]
        mlc[mlclient]
        mail[mail: SMTP]
        cfg[config, clock, ids]
    end
    mw --> handlers --> domain
    domain --> platform
```

| Module | Owns | Notes |
|---|---|---|
| `auth` | users, sessions, auth tokens | argon2id, session cookies, invitations |
| `access` | role to permission map, scope checks | pure functions, no tables |
| `masterdata` | categories, compatibility, zones, locations, posts, officer assignments, synonyms, brands | seed command |
| `reports` | lost and found reports, places, status history, code sequences | state machines L and F |
| `media` | media objects, variants, hashes, staging | storage interface |
| `matching` | embeddings, attributes, matches, feedback, model versions, fusion configs | retrieval and scoring, state machine M |
| `assist` | none | orchestrates the sync analysis call and masking |
| `claims` | claims, answers, evidence, messages, checklist | state machine K |
| `custody` | custody events, dispositions, handover codes | state machine C |
| `notify` | notifications, preferences, email jobs, SSE hub | `LISTEN notifications` |
| `search` | none, reads published views | trigram, full text, vector |
| `officer` | none | priority queue and review board read models |
| `admin` | system settings, moderation records | settings cache |
| `audit` | audit log, access log | append only |
| `analytics` | materialised views | refresh job |

## 5. Key runtime flows

### 5.1 Found report to candidates to notification

```mermaid
sequenceDiagram
    autonumber
    participant B as Browser
    participant A as api
    participant DB as postgres
    participant W as worker
    participant M as ml
    B->>A: POST /api/v1/media (photo)
    A->>DB: insert media (staged)
    A-->>B: 201 media id
    B->>A: POST /api/v1/assist/analyze-image
    A->>M: /v1/analyze/image
    M-->>A: categories, colours, text, brands
    A-->>B: suggestions
    B->>A: POST /api/v1/found-reports (Idempotency-Key)
    A->>DB: tx: report, attach media, history, audit, enqueue media.process
    A-->>B: 201 FND-2611-0042
    W->>DB: media.process: variants, hashes
    W->>M: embed image, analyze image
    W->>DB: store embeddings and attributes, enqueue ai.embed_text
    W->>M: embed text
    W->>DB: store text embeddings, enqueue match.report
    W->>DB: match.report: retrieval and scoring, upsert matches, status effects
    W->>DB: match.notify: notifications rows, NOTIFY
    DB-->>A: NOTIFY notifications
    A-->>B: SSE event to the owner's open sessions
```

### 5.2 Claim, approval, handover

```mermaid
sequenceDiagram
    autonumber
    participant O as Owner
    participant A as api
    participant DB as postgres
    participant F as Officer
    O->>A: POST /api/v1/claims
    A->>DB: tx: claim submitted, checklist, lost L4, found F4, notify officers
    F->>A: GET claim (review)
    A->>DB: claim under_review, access log
    F->>A: PUT checklist results
    F->>A: POST approve
    A->>DB: tx: guard custody in_custody, checklist complete, version check
    A->>DB: claim approved, code hash stored, other claims on_hold, F6, L6, M5
    A-->>O: notification, code visible in app
    O->>F: visits post with KTM and code
    F->>A: POST handover (code, ID type, last 4, name check)
    A->>DB: tx: verify HMAC, K11, F8, L8, C5, reject others, audit
    A-->>F: 200 handed over
```

### 5.3 Live notifications

The SSE endpoint holds one connection per open tab. Each `api` instance keeps a map of
user id to connections and one `LISTEN notifications` connection to PostgreSQL. A worker
or another instance that creates a notification issues `NOTIFY notifications, '<user id>'`
after commit. The receiving instance reads the new rows for that user and writes them to
the stream. On reconnect the browser sends `Last-Event-ID`, the notification id, and
receives everything newer. A heartbeat comment every 25 s keeps proxies from closing the
stream.

## 6. Cross cutting concerns

| Concern | Decision |
|---|---|
| Transactions | A request handler runs one transaction for one use case. Audit, history, notifications and job enqueueing (River's `InsertTx`) happen inside it, so either everything or nothing is committed |
| Concurrency | `version` column on mutable aggregates. A stale write returns `409 conflict` with code `stale_version` |
| Idempotency | `Idempotency-Key` on create endpoints stored with the response for 24 hours |
| Time | `timestamptz` in UTC everywhere. Conversion to `Asia/Jakarta` only in the presentation layer and in job schedules, which name the zone explicitly. Codes use the WIB month (BR-ID-01) |
| Identifiers | UUIDv7 generated in Go |
| Errors | RFC 9457 problem details with stable codes, see [api.md](api.md) |
| Configuration | environment variables parsed once at start into a typed struct, refusing to start on a missing required value |
| Logging | `log/slog` JSON in Go, structured JSON from Next.js and FastAPI, common fields in [observability.md](../operations/observability.md) |
| Request id | `X-Request-Id` set by Caddy when absent, forwarded by `web`, logged by `api`, passed to `ml` and stored in job metadata |
| Feature flags | environment flags read at start, for features that land across several pull requests |

## 7. Deployment view

```mermaid
flowchart TB
    subgraph vps[VPS: 4 vCPU, 8 GB RAM, 80 GB SSD, Ubuntu 24.04 LTS]
        subgraph net_public[network edge]
            caddy
        end
        subgraph net_app[network app, internal]
            web
            api
            worker
            ml
        end
        subgraph net_data[network data, internal]
            postgres
        end
        vol1[(pg-data)]
        vol2[(media)]
        vol3[(ml-models)]
        vol4[(backups)]
    end
    internet((Internet)) -->|443, 80| caddy
    caddy --> web
    caddy --> api
    web --> api
    api --> postgres
    worker --> postgres
    worker --> ml
    api --> ml
    postgres --- vol1
    api --- vol2
    worker --- vol2
    ml --- vol3
```

Only `caddy` publishes ports. `ml` and `postgres` are on internal networks without
published ports. Memory budget: postgres 2 GB, ml 3 GB, api and worker 512 MB each, web
512 MB, caddy, prometheus and grafana 512 MB together, leaving headroom for the OS. Full
procedure in [deployment.md](../operations/deployment.md).

## 8. Where to go next

| Topic | Page |
|---|---|
| Tables and constraints | [data-model.md](data-model.md) |
| Endpoints | [api.md](api.md) |
| Security and privacy | [security-privacy.md](security-privacy.md) |
| Matching internals | [../ai/matching-engine.md](../ai/matching-engine.md) |
| Folder layout per service | [../engineering/repository-structure.md](../engineering/repository-structure.md) |
