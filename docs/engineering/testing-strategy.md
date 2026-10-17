# Testing Strategy | SimpanUGM

Tests are written with the code, in the same pull request, and they are what CI runs to
decide whether `main` stays deployable.

## 1. Levels

| Level | Scope | Tools | Runs |
|---|---|---|---|
| Unit | one function or type, no I/O: state machines, scoring, fusion, permissions, masking, formatting, validation | Go `testing`, Vitest, pytest | every push, seconds |
| Integration | one service over its real boundaries: HTTP handler to PostgreSQL, storage drivers, River jobs, `ml` endpoints with real models | testcontainers PostgreSQL with pgvector, `httptest`, FastAPI `TestClient` | every pull request |
| Contract | `api` responses validate against `openapi.yaml`, `mlclient` against `ml/openapi.json`, Go and Python fusion agree on shared fixtures | kin-openapi, schemathesis on `ml`, shared JSON fixtures | every pull request |
| End to end | the seven journeys of prd section 6 through the browser on the full Compose stack | Playwright with axe | pull requests touching `web` or `api`, nightly on `main` |
| Evaluation | the AI gates | `ml/eval` | on demand, on `exp/` branches, before activation |
| Load | NFR-PERF on the reference server | k6 | before v0.3.0 and v1.0.0 |
| Security | dependency, secret and dynamic scans | govulncheck, pnpm audit, pip-audit, gitleaks, ZAP baseline | every pull request, ZAP weekly on staging |

## 2. What must be tested, by area

### 2.1 `api`

| Subject | Required tests |
|---|---|
| State machines | one table driven test per machine in state-machines: every listed transition succeeds with its effects, every unlisted pair is refused |
| Permissions | every cell of the permissions matrix, plus list scoping with three users |
| Projections | response snapshots per role proving forbidden fields are absent |
| Business rules | one test per `BR` id with a number in it, named after the id, for example `TestBR_CLM_01_MaxOpenClaims` |
| Matching | scoring against `data/fixtures/fusion_cases.json`, retrieval against a seeded database with known neighbours, the status effects of M1 to M7 |
| Concurrency | two concurrent approvals of one claim: exactly one succeeds, the other gets `stale_version` |
| Idempotency | a repeated `Idempotency-Key` returns the first response and creates one row |
| Jobs | each worker is idempotent: running it twice leaves the same state |
| Media | signature check, size limits, EXIF removal with a GPS tagged fixture, variant sizes, perceptual hash of a known image |
| Migrations | up, down, up on an empty database in CI |
| Audit | every state changing endpoint writes exactly the expected audit rows, and `simpan_app` cannot update or delete them |

### 2.2 `web`

| Subject | Required tests |
|---|---|
| Forms | each step's validation, draft restore, idempotent resubmission after a simulated network failure |
| Status vocabulary | every enum value has a label, icon and colour |
| Formatting | WIB conversion across midnight UTC, relative times, distances, percentages |
| Route guards | each role reaches its routes and is redirected from others |
| Accessibility | axe has no serious or critical violation on every page visited by the end to end suite |

### 2.3 `ml`

| Subject | Required tests |
|---|---|
| Embeddings | dimension, L2 norm of 1, and golden cosine values for a fixed image and text pair within `1e-4` for the pinned revision |
| Analysis | category top 3 on fixture images, colour naming on solid colour swatches, brand matching on fixture strings, masking on `masking_cases.json` |
| API | every endpoint with valid and invalid input, the token check, size limits, `readyz` before and after loading |
| Evaluation harness | evaluation section 12 |

### 2.4 `data`

| Subject | Required tests |
|---|---|
| Seeds | schema validation of each file, every location inside the campus bounding box, every leaf category with sensitivity and at least one prompt, every compatibility pair referring to existing leaves |
| Evaluation dataset | `make eval-check` on the manifest |

## 3. End to end journeys

Each journey runs on a fresh stack seeded with `simpan seed --demo` and a fake clock
endpoint enabled only in the `e2e` environment.

| Journey | Covers |
|---|---|
| J1 Register, verify through Mailpit, sign in | US-01, US-02 |
| J2 Owner files a lost report without photo, sees candidates after a seeded found report | US-08, US-19, US-20 |
| J3 Finder files a found report with photo and assistant, sees the hint, officer receives it | US-13, US-14, US-23, US-30 |
| J4 Owner claims, officer requests info, owner replies, officer approves | US-25, US-26, US-27 |
| J5 Handover with a wrong code, then the right one, owner confirms receipt | US-31 |
| J6 Retention passes on the fake clock, officer disposes | US-33 |
| J7 Admin activates a fusion version and rolls back, monitoring reflects it | US-49 |

## 4. Coverage targets

NFR-MAINT-01: Go domain packages 80 percent, `api` overall 70 percent, `ml/app` 70
percent, `web/src` 60 percent. Coverage is reported in CI, and a drop of more than 2
points on a pull request needs a reason in its description.

## 5. Load scenarios

Run against staging sized like production, with 5,000 seeded open reports and 20,000
images.

| Scenario | Shape | Pass |
|---|---|---|
| Browse | 100 virtual users for 10 minutes: home, report detail, candidates, notifications | NFR-PERF-01 |
| Report burst | 10 found reports with 2 photos per minute for 10 minutes on top of Browse | NFR-PERF-02, NFR-PERF-03 |
| Assistant | 5 concurrent analyses for 5 minutes | NFR-PERF-04 |
| Officer | 10 officers cycling queue, board, claim review | NFR-PERF-01 |

## 6. CI pipeline

| Workflow | Trigger | Steps | Time budget |
|---|---|---|---|
| `ci-api.yml` | changes in `api/`, `data/fixtures/`, `data/seed/` | generate and diff check, lint, unit, integration with PostgreSQL, migrations up and down, coverage, govulncheck | 6 min |
| `ci-web.yml` | changes in `web/`, `api/openapi.yaml` | install, generate client and diff check, lint, typecheck, unit, build, pnpm audit | 5 min |
| `ci-ml.yml` | changes in `ml/`, `data/fixtures/` | uv sync, ruff, mypy, unit, API tests with cached weights, pip-audit | 7 min |
| `ci-data.yml` | changes in `data/` | seed validation, manifest check | 1 min |
| `e2e.yml` | pull requests touching `web/` or `api/`, nightly | compose up, seed, Playwright, upload report | 10 min |
| all | every pull request | gitleaks, commit message lint | 1 min |

Model weights for CI are cached by the hash of `ml/models.lock.json`.

## 7. Test data

- No personal data from real people in fixtures. Names come from a fixed list of
  obviously fictional names.
- Images in fixtures are small, made by the team, and contain no people and no real
  documents.
- Demo seed data is clearly marked: report descriptions start with "[DEMO]" and demo
  accounts use the domain `demo.simpanugm.my.id`, which is allowed only outside
  production.
