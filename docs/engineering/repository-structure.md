# Repository Structure | SimpanUGM

The layout every folder grows into. Folders exist from the first commit with a README so
the shape is visible before the code. A new top level folder needs an ADR.

## 1. Root

```
simpanugm/
  .github/
    workflows/            ci-api.yml, ci-web.yml, ci-ml.yml, ci-data.yml, e2e.yml, release.yml, deploy.yml
    ISSUE_TEMPLATE/
    dependabot.yml
    pull_request_template.md
  api/                    Go service and worker
  web/                    Next.js app
  ml/                     inference service and evaluation harness
  data/                   seeds, evaluation dataset pipeline, fixtures shared by api and ml
  deploy/                 compose files, Caddy, monitoring, backups
  docs/                   this specification
  Makefile                one entry point for every task, see local-development section 4
  README.md  CHANGELOG.md  CONTRIBUTING.md  SECURITY.md
  .editorconfig  .gitattributes  .gitignore
```

## 2. `api/`

```
api/
  cmd/simpan/main.go              commands: serve, worker, migrate, seed, admin
  internal/
    platform/
      config/                     typed environment config
      db/                         pgx pool, transaction helper, sqlc wiring
      httpx/                      server, middleware, problem details, request id
      jobs/                       River client, queue config, periodic job registry
      storage/                    Store interface, fs and s3 drivers, shared contract tests
      mail/                       SMTP sender, template rendering
      mlclient/                   typed client for the inference API
      metrics/                    Prometheus registry and common metrics
      clock/  ids/                injectable time and UUIDv7 generation for tests
    auth/  access/  masterdata/  reports/  media/  matching/  assist/  claims/
    custody/  notify/  search/  officer/  admin/  audit/  analytics/
    db/sqlc/                      generated code, never edited
  migrations/                     0001_extensions.sql, 0002_accounts.sql, ...
  openapi.yaml                    the public API contract
  sqlc.yaml
  .golangci.yml
  Dockerfile
  README.md
```

Inside each module the same files, so a reader always knows where to look:

```
internal/claims/
  service.go          use cases, transactions, state machine calls
  machine.go          the state machine table as data plus the transition function
  store.go            thin wrapper over generated queries
  queries.sql         sqlc input for this module
  handlers.go         adapters from the generated server interface to the service
  jobs.go             River workers owned by the module
  projection.go       role specific response mapping (BR-VIS)
  *_test.go           unit tests next to the code, integration tests tagged `integration`
```

`matching/` adds `retrieval.go`, `scoring.go`, `fusion.go`, `explain.go`, and `fuzzy/`.

## 3. `web/`

```
web/
  src/
    app/
      (public)/           landing, masuk, daftar, verifikasi, lupa-sandi, atur-sandi, undangan, privasi, ketentuan, bantuan
      (member)/           beranda, lapor/hilang, lapor/temuan, laporan, klaim, barang-temuan, notifikasi, profil
      (officer)/petugas/  every officer route of information-architecture 2.3
      (admin)/admin/      every admin route of information-architecture 2.4
      layout.tsx  error.tsx  not-found.tsx
    components/
      ui/                 shadcn components, owned by us after generation
      report/  match/  claim/  officer/  admin/  map/  charts/  layout/
    lib/
      api/                client.ts, schema.d.ts generated from api/openapi.yaml
      auth/               session helpers for server components and the route guard
      image/              compression and HEIC conversion
      draft/              form drafts in local storage
      status.ts           the status vocabulary of ui-guidelines section 4
      format.ts           WIB dates, distances, percentages
    messages/id.json      every UI string
    styles/globals.css  styles/tokens.css
    middleware.ts         role guard and CSP nonce
  tests/
    unit/                 Vitest and Testing Library
    e2e/                  Playwright journeys
  public/
  next.config.ts  vitest.config.ts  playwright.config.ts  eslint.config.mjs  tsconfig.json
  package.json  pnpm-lock.yaml  Dockerfile  README.md
```

## 4. `ml/`

```
ml/
  app/
    main.py               FastAPI app, lifespan loads models
    config.py             settings from environment
    registry.py           reads models.lock.json, loads and warms models
    routers/              health.py, models.py, embed.py, analyze.py, cluster.py
    analysis/             category.py, colour/, ocr.py, brand.py, masking.py
    schemas.py            request and response models
    logging.py  metrics.py
  eval/
    run.py                entry point for make eval
    configs/              run configs and frozen fusion documents
    scenario.py           metadata generator of evaluation section 3
    datasets.py           manifest loading and verification
    fusion.py             reference implementation of matching-engine sections 6 and 7
    calibrate.py          evaluation section 6
    metrics.py            evaluation section 5
  tests/
  notebooks/              exploration only, outputs stripped, never a source of a reported number
  models.lock.json        pinned model ids and revisions
  openapi.json            generated, the inference contract
  pyproject.toml  uv.lock  Dockerfile  README.md
```

`ml/eval/fusion.py` and `api/internal/matching/` implement the same formulas in two
languages. Both test suites read `data/fixtures/fusion_cases.json` and must agree to six
decimals, so the evaluated engine and the deployed engine cannot drift apart.

## 5. `data/`

```
data/
  seed/
    zones.csv  locations.csv  posts.yaml  categories.yaml  category_compatibility.csv
    colours.csv  brands.csv  synonyms.csv  settings.yaml  model_versions.yaml
  eval/
    README.md             dataset card, consent form link, version history
    manifest.json         the only evaluation file in git
  fixtures/
    fusion_cases.json     shared scoring cases for Go and Python
    masking_cases.json    BR-PRV-03 cases
    fuzzy_cases.json      token set ratio cases generated with rapidfuzz
    images/               tiny synthetic test images made by the team, no people, no real documents
  scripts/
    intake.py             strips EXIF, renames, hashes, writes the manifest
    check.py              evaluation dataset quality checks
    fetch.py              downloads a dataset version from Drive and verifies hashes
    osm_locations.py      pulls building names and coordinates from OpenStreetMap for review
  analytics/
    views.sql             reference definitions reviewed before they become migrations
  README.md
```

## 6. `deploy/`

```
deploy/
  compose.yaml            every service, networks, volumes, health checks
  compose.dev.yaml        development overrides: Mailpit, published ports, hot reload mounts
  compose.prod.yaml       production overrides: images by digest, resource limits, restart policy
  caddy/Caddyfile
  env/                    api.env.example, web.env.example, ml.env.example
  prometheus/prometheus.yml  prometheus/alerts.yml
  grafana/provisioning/  grafana/dashboards/
  backup/                 backup.sh, restore.sh, restic excludes
  README.md
```
