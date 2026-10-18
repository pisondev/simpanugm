# Tech Stack | SimpanUGM

Every runtime, library and tool, with the reason it is here. Versions name the release
line. Exact versions are pinned in `go.mod` and `go.sum`, `pnpm-lock.yaml`, `uv.lock`, and
container images by digest in `deploy/compose.yaml`. Dependabot proposes updates weekly.

An added dependency needs a line in this table in the same pull request.

## 1. Runtimes and platform

| Component | Version line | Why |
|---|---|---|
| Go | 1.26 | Agreed stack. Standard library `log/slog`, `net/http` routing patterns and `http.CrossOriginProtection` reduce dependencies |
| Node.js | 22 LTS | Runtime for Next.js 16, supported until April 2027, matches team machines |
| pnpm | 10 | Fast, strict dependency resolution, content addressed store |
| Python | 3.12 | The line every ML dependency below supports |
| uv | latest 0.x | Locking and environment management for `ml`, much faster than pip |
| PostgreSQL | 18 | Agreed stack. Current major with async I/O and UUIDv7 support |
| pgvector | 0.8 | HNSW with iterative index scans, needed for filtered retrieval (ADR-0004) |
| Docker Engine and Compose | 27 or newer, Compose v2 | The deployment unit (NFR-PORT-01) |
| Caddy | 2.10 | Automatic HTTPS, HTTP/3, simple routing and headers in one file |

## 2. `api`, Go

| Library | Purpose | Why this one |
|---|---|---|
| `github.com/go-chi/chi/v5` | router and middleware composition | small, idiomatic, compatible with `net/http`, the target of `oapi-codegen` |
| `github.com/oapi-codegen/oapi-codegen/v2` | server interfaces and types from `openapi.yaml` | contract first (ADR-0009), strict server mode |
| `github.com/getkin/kin-openapi` | request validation against the spec | runtime enforcement of the contract |
| `github.com/jackc/pgx/v5` | PostgreSQL driver and pool | fastest Go driver, native types, `LISTEN` support, required by River |
| `github.com/sqlc-dev/sqlc` (tool) | type safe Go from SQL | SQL stays SQL, reviewed as SQL, no ORM magic |
| `github.com/pgvector/pgvector-go` | vector type for pgx | official pgvector binding |
| `github.com/pressly/goose/v3` | migrations, embedded in the binary | SQL migrations with up and down, run by `simpan migrate` |
| `github.com/riverqueue/river` | job queue on PostgreSQL | transactional enqueue, retries, scheduling, unique and periodic jobs (ADR-0005) |
| `github.com/caarlos0/env/v11` | environment to typed config | fails fast on missing values |
| `github.com/google/uuid` | UUIDv7 | `uuid.NewV7` |
| `golang.org/x/crypto/argon2` | password hashing | NFR-SEC-03 |
| `golang.org/x/time/rate` | token bucket rate limiting | in process limits, enough for one instance. A PostgreSQL backed limiter replaces it if `api` is scaled out |
| `github.com/disintegration/imaging` | resize, orientation, blur | pure Go |
| `github.com/gen2brain/webp` | WebP encode and decode | libwebp through WebAssembly, no cgo, keeps the image static. Built with the `nodynamic` tag, otherwise purego links libc |
| `github.com/corona10/goimagehash` | perceptual hash | FR-MEDIA-05 |
| `github.com/aws/aws-sdk-go-v2/service/s3` | S3 storage driver | standard client for any S3 compatible store (ADR-0007) |
| `github.com/wneessen/go-mail` | SMTP | maintained, TLS and STARTTLS, multipart |
| `github.com/prometheus/client_golang` | metrics | NFR-OBS-03 |
| `github.com/agnivade/levenshtein` | edit distance for the Go port of the token set ratio | small, no dependencies |
| `github.com/google/go-cmp` | test comparisons | readable diffs with the standard `testing` package |
| `github.com/testcontainers/testcontainers-go` | real PostgreSQL in integration tests | tests against the same extensions as production |

Tools: `golangci-lint` v2, `govulncheck`, `air` for live reload in development.

## 3. `web`, TypeScript

| Library | Purpose | Why this one |
|---|---|---|
| `next` 16.4 | framework, App Router, server components, Turbopack | agreed stack, server rendering for fast first paint on phones. Cache Components stay off: every page is personal and dynamic, so the prerender model adds rules without a benefit |
| `react` 19 | UI | required by Next.js 16 |
| `typescript` 5 | types, `strict` mode | |
| `tailwindcss` 4 | styling with tokens | utility classes with CSS variables, no runtime cost |
| shadcn/ui CLI 4, style `base-nova` on `@base-ui/react` | accessible primitives copied into the repo | full control of markup, Base UI is the primitive layer the CLI generates today. The pre 1.0 `cn` package the CLI suggests was replaced by `clsx` and `tailwind-merge` |
| `lucide-react` | icons | the single icon set (UI-03) |
| `openapi-typescript`, `openapi-fetch` | typed API client generated from the spec | ADR-0009 |
| `@tanstack/react-query` 5 | client data fetching, caching, mutations | retries and invalidation for flaky mobile connections |
| `@tanstack/react-table` 8 | officer tables with client sorting and keyboard navigation | headless. Adopted with the queue in Sprint 3, the Sprint 1 tables sort on the server |
| `react-hook-form` 7, `zod` 4, `@hookform/resolvers` | forms and validation | the multi step report forms |
| `next-intl` 4 | message catalog | copy in one place, English later without rewiring |
| `Intl.DateTimeFormat`, `Intl.RelativeTimeFormat` | WIB formatting and relative times | built into every supported browser and Node, the formats of ui-guidelines 6.3 need no library |
| `browser-image-compression` 2 | client resize before upload | FR-MEDIA-03 |
| `heic-to` | HEIC to JPEG in the browser | iPhone photos (FR-MEDIA-01) |
| `react-leaflet` 5, `leaflet` 1.9, `leaflet.heat` | campus map and heat layer | OpenStreetMap without an API key |
| `recharts` 3 | charts | declarative, works with server rendered pages |
| `@zxing/browser` | QR scanning on the receive screen | camera scanning without native code. Arrives with the handover QR in Sprint 3, until then the code is typed or pasted |
| `sonner` | toasts | shadcn default |

Testing and tooling: `vitest`, `@testing-library/react`, `msw` 2 for API mocks,
`@playwright/test` with `@axe-core/playwright`, ESLint 9 flat config with
`eslint-config-next` and `typescript-eslint` strict, Prettier 3 with the Tailwind plugin.
Node.js 22.22.2 or newer, the floor set by jsdom 30.

## 4. `ml`, Python

| Library | Purpose | Why this one |
|---|---|---|
| `fastapi`, `uvicorn` | HTTP service | typed request models, OpenAPI generated for the contract |
| `pydantic` 2 | schemas and settings | |
| `torch` CPU build | inference runtime | installed from the CPU wheel index, about a fifth of the CUDA build size |
| `sentence-transformers` | CLIP image and multilingual text encoders, e5 | one API for all three encoders |
| `transformers`, `huggingface-hub` | model loading by pinned revision | NFR-REPRO-02 |
| `easyocr` | visible text, Indonesian and English | CPU capable, Indonesian recogniser available |
| `opencv-python-headless` | image operations for OCR | required by EasyOCR, headless keeps the image small |
| `pillow` | decoding and resizing | |
| `numpy` 2 | arrays | |
| `scikit-learn` | HDBSCAN, k-means, logistic regression for learned fusion | one library for every classical method we use |
| none, vendored CIEDE2000 in `ml/app/colour/delta_e.py` | colour distance | a 40 line function with reference tests beats an unmaintained dependency |
| `rapidfuzz` | brand matching, reference for the Go port | fast and exact |
| `prometheus-fastapi-instrumentator` | metrics | NFR-OBS-03 |
| `python-json-logger` | JSON logs | NFR-OBS-01 |
| `mlflow` 3 | experiment tracking for `ml/eval` | runs, params and artefacts in one place, local file store |
| `pandas` | evaluation tables | eval only, not in the serving image |

Testing and tooling: `pytest`, `pytest-cov`, `httpx` for API tests, `ruff` for lint and
format, `mypy` in strict mode on `ml/app`, `pip-audit`.

## 5. Operations

| Tool | Purpose |
|---|---|
| GitHub Actions | CI and CD |
| GitHub Container Registry | image registry |
| Prometheus 3, Grafana 12 | metrics and dashboards |
| Mailpit | SMTP catcher in development |
| restic | encrypted backups of dumps and media |
| k6 | load tests for NFR-PERF |
| OWASP ZAP baseline | security scan of staging |
| gitleaks | secret scanning in CI |
| Dependabot | dependency updates for Go, npm, pip and GitHub Actions |

## 6. Rejected on purpose

| Option | Why not |
|---|---|
| An ORM such as GORM | hides SQL that must be reviewed for scope filters and indexes |
| Redis | nothing needs it: sessions, jobs, rate limits and pub sub are covered by PostgreSQL and process memory |
| A separate vector database | ADR-0004 |
| MinIO | ADR-0007 |
| A hosted LLM or vision API | CON-05, privacy and reproducibility |
| Kubernetes | one host, Compose is enough and the team can operate it |
