# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Security

- Pre-release security hardening and dependency audit (US-55, NFR-SEC-01 to NFR-SEC-10, NFR-PRIV-06): verified zero vulnerabilities across production runtime dependencies (`pnpm audit --prod`, `pip-audit`, and `govulncheck` on application packages); added `Strict-Transport-Security` header to Caddy configuration; documented status in `docs/operations/release-checklist-v1.0.0.md`.
- The API pins the go1.26.9 toolchain, which fixes the HTTP/2 HPACK race GO-2026-6617, and lifts `golang.org/x/image` from a 2019 snapshot to v0.47.0, closing eight TIFF and BMP decoding denial of service advisories reachable through image decoding, plus `moby/go-archive` to v0.3.3 for the integration tests. `govulncheck` reports no reachable vulnerability.

### Changed

- Updated requirement traceability matrix and Definition of Done verification audit for Sprint 4 stories (US-41 through US-55); recorded load test results with all three scenarios meeting p95 latency budgets.
- The fake clock and e2e application environment move from `deploy/compose.dev.yaml` to `deploy/compose.e2e.yaml`, so local development with `make dev` runs on the real system clock while e2e workflows and `make e2e` apply the e2e compose override.
- `simpan seed` no longer overwrites a master data row an administrator edited, and a compatibility pair an administrator switched off stays off. A leaf under a switched off parent can no longer be picked for a new report.
- Lost reports carry their real `candidate_count`. The owner of a lost report with a visible candidate may open that found item's display photo, or only its blurred thumbnail for an identity item (BR-VIS 5.2).
- Dependabot proposes only patch and digest updates for the Go, Python and Node base images. A runtime upgrade changes code, CI and docs together, so it is made by hand.
- The data model lists the `custody_keep` and `close_note` columns of found reports that the migrations already carry.
- `GET /api/v1/posts` is public, so the help page lists posts and opening hours before sign in. Every other master data read still needs a session.

### Fixed

- In `deploy/scripts/restore.sh`, corrected the post-restore report verification query to count from `lost_reports` and `found_reports` instead of the non-existent `reports` table; configured `web/src/app/readyz/route.ts` to support `API_INTERNAL_URL`.
- The fake clock of the end to end stack keeps running from where it was set instead of freezing, so a long suite no longer falls behind the browser and gets "found in the future" refusals. The generated sqlc models match the migrations again, and the photo step of the end to end journeys retries until the page has hydrated.
- The end to end suite no longer trips the registration limit of five accounts per hour and address: `AUTH_REGISTER_PER_HOUR` sets it, the dev Compose override raises it, and the axe helper waits for a fading toast before it scans.
- Resending a verification link or starting an email change no longer fails for an account without earlier tokens of that kind: the issue statistics default to the epoch instead of minus infinity, which Go cannot represent.
- `api/internal/media` and `api/internal/platform/storage` are in the repository: unanchored `media/` and `storage/` ignore rules had kept them out, so a fresh clone did not build. The rules are anchored to the root now.
- The API image starts again: it is built with the `nodynamic` tag, so `gen2brain/webp` stays on its WebAssembly encoder instead of linking libc through purego, which `distroless/static` cannot load. A root `.dockerignore` replaces the one in `api/` that the root build context never read.
- The fetch test compares resolved paths, so it also passes on Windows where temporary directories have short names.
- Masking now hides every 16 digit run, so NIK numbers are covered although they fail the Luhn check, and treats digits grouped by single spaces or hyphens as one run, as BR-PRV-03 requires.

### Added

- Automated release workflow and CI security checks (PRD section 9, testing-strategy section 6): created `.github/workflows/release.yml` for automated building and publishing of OCI images for API, Web, and ML to GHCR on tags `v*`; added `pnpm audit --prod` to `ci-web.yml` and `uv run pip-audit` to `ci-ml.yml`; added automated integration test `TestMigrationsUpDownUp` in `api/internal/server/migrations_integration_test.go` verifying that all 13 database migrations run forward, backward to 0, and forward again on a clean database; propagated `x-request-id` headers across Web middleware, API context handler, and ML FastAPI middleware with structured JSON request logging; expanded `docs/operations/release-checklist-v1.0.0.md` with complete line-by-line audit of PRD section 9.

- Fusion version activation, rollback, and sweep rescoring (Journey J7, US-49, FR-ADMIN-06, testing-strategy section 3): added audit log recording and River background sweep rescoring to `ActivateModelVersion`; registered candidate fusion version `fusion-learned-2026.11.0` alongside active linear baseline; added integration test `FusionActivationAndRollback` verifying candidate activation, audit logging, monitoring reflection, and rollback; added Playwright Journey J7 end-to-end test scenario `web/tests/e2e/j7-fusion.spec.ts` with axe accessibility verification; improved model version selector accessibility.

- Observability dashboards, alert rules, and runtime metrics (US-52, NFR-OBS-01 to NFR-OBS-03): expanded API platform metrics to include in-flight HTTP requests, SSE connection counts, database pool connection tracking, login outcomes, rate limiting counters, state machine transitions, River job outcomes, queue depth, and notification delivery counters; added `/readyz` probe to web service checking API reachability; provisioned Prometheus alert rules in `deploy/prometheus/alerts.yml` covering site liveness, error rates, slow API reads, matching stalls, ML readiness, discarded jobs, disk thresholds, backup staleness, and TLS cert expiry; added Grafana dashboard definitions in `deploy/grafana/dashboards/` for Overview, Jobs and Matching, Database, and Host.

- Production deployment artifacts, backup, and restore automation (US-53, US-54, NFR-AVAIL-04, NFR-AVAIL-05, NFR-PORT-01): created `deploy/compose.prod.yaml` with production image tags, log rotation policies, and resource limits; created `deploy/scripts/deploy.sh` with zero-downtime phased restart, migration execution, automated smoke tests, and rollback; created `deploy/scripts/backup.sh` for custom-format pg_dump with Prometheus metrics and retention rotation; created `deploy/scripts/restore.sh` with post-restore integrity verification across schemas and tables; created `deploy/env/compose.prod.env.example`.

- Learned fusion experiment and evaluation comparison (Sprint 4 Irfan scope, matching-engine section 7.2, evaluation.md section 13.3): implemented logistic regression learned fusion on 13 multimodal components (`ml/eval/learned_fusion.py`) with 5-fold GroupKFold cross-validation grouped by item; benchmarked against calibrated linear fusion B3 on synthetic evaluation fixture (SINTETIS), achieving Recall@5 1.0000 vs 0.4000, MRR 0.9500 vs 0.4000, and high-tier precision 1.0000; retained calibrated linear fusion as active production model per project policy until frozen real dataset evaluation (TERTUNDA); updated `docs/ai/evaluation.md` and added fusion model card to `docs/ai/model-cards.md`.

- Officer semantic search web interface (Sprint 4, US-41, FR-SEARCH-04, AI-13): added semantic search toggle switch in `/petugas/cari` for natural phrasing queries; added relevance percentage badge in search results when semantic or text similarity scores are returned; updated Indonesian translations for semantic search; verified with TypeScript typecheck, ESLint, Prettier, and Vitest unit tests.

- Semantic search backend via dense e5 text embedding and reciprocal rank fusion (Sprint 4, US-41, FR-SEARCH-04, AI-13): added `semantic` query parameter to `GET /search`; implemented `searchSemantic` evaluating cosine distance against report `text_embeddings_e5` inside all active filters (parent and child categories, zones, locations, time windows, custody, post, status, hidden check); implemented reciprocal rank fusion with k = 60 between trigram keyword search and semantic vector search; added `FEATURE_SEMANTIC_SEARCH` configuration flag and graceful fallback to keyword search when ML service is offline; verified with unit tests and integration test suite.

- AI monitoring, case clustering, operational trends, and campus hotspots web interface (US-46 to US-49, FR-ANL-01 to FR-ANL-06, FR-ADMIN-06, FR-ADMIN-07, AI-10 to AI-12): added `/admin/ai/monitoring` for match engine metrics, score PSI drift alert banner, confirmed rank recall meters, dismiss rates per tier, and active model versions table with rollback dialog; added `/admin/ai/klaster` for case clustering runs, cluster cards with dominant categories and zones, thumbnail strips, and member tables; added `/admin/tren` for operational KPIs, 12-week progression table, hour-by-weekday matrix with cell count suppression (< 3), and monthly CSV export form; added `/admin/peta-rawan` for campus hotspot map with time and type filters, statistical surge alert banner, and ranked locations table with BR-PRV-05 suppression; updated staff shell navigation and Indonesian messages; verified with unit tests.

- AI monitoring, case clustering, operational trends, and campus hotspots backend (US-46 to US-49, FR-ANL-01 to FR-ANL-06, FR-ADMIN-06, FR-ADMIN-07, AI-10 to AI-12): migration 00013 adding cluster_runs, cluster_members, and materialized views mv_reports_daily, mv_hour_weekday, mv_return_times, mv_match_outcomes; implemented GET /admin/model-versions and POST /admin/model-versions/{id}/activate; implemented GET /admin/ai/monitoring with assist outcomes, dismiss rates, confirmed ranks, top-1 histogram, and score PSI drift computation; implemented POST /admin/ai/clusters/runs calling ML HDBSCAN clustering and GET /admin/ai/clusters; implemented GET /analytics/hotspots with BR-PRV-05 cell count suppression (< 3) and statistical surge insight generation; implemented GET /analytics/summary, GET /analytics/trends, and GET /analytics/export with monthly CSV export; verified with integration test suite.

- Administration web pages for settings, audit logs, moderation, and account deletion (US-42 to US-45, FR-ADMIN-01, FR-ADMIN-04, FR-ADMIN-05): added `/admin/pengaturan` for runtime business rules configuration across lost reports, custody retention, claims, notifications, and auth; added `/admin/audit` with filtering by action, entity type, date range, pagination, and direct CSV download; added user moderation menu in `/admin/pengguna` for suspend, unsuspend, session revocation, claim restriction clearance, and anonymization; added report moderation dialog for officer and admin views; added self-service account deletion dialog in `/profil/keamanan` with cascading confirmation and active claim safeguards.

- Administration backend endpoints for moderation, runtime settings, audit log with CSV export, and self-service account deletion (US-42 to US-45, FR-ADMIN-01, FR-ADMIN-04, FR-ADMIN-05, BR-RET-05): implemented report hide and unhide with candidate superseding, user suspension with immediate session revocation, session termination, claim restriction clearance, runtime settings query and update with validation against business-rules section 10, audit log filtering and CSV streaming export, database immutability verification for audit logs, and account self-deletion with anonymization and cascading report closure.

- Officer match review board web interface (Sprint 3 Stage 3D, US-39, FR-OFF-03, FR-OFF-04, FR-AI-11, AI-07): added `/petugas/papan` review board page providing side-by-side comparison of selected report and candidates, component score breakdown meters, photo comparisons, private and hidden detail viewing for staff, on-demand rematch button, duplicate link, candidate pinning and unpinning, claim invitations to lost report owners, and match dismissals; added unit tests and navigation links from officer workspace and candidate tables.

- Match review board, candidate pinning, and claim invitation backend (Sprint 3 Stage 3D, US-39, FR-OFF-03, FR-OFF-04, FR-AI-11, AI-07): GET /officer/board providing side-by-side match review data with full report details, candidate rankings, and private detail access; POST /matches/{id}/pin allowing officers to pin top candidate pairs; POST /matches/{id}/invite-claim dispatching claim invitations to lost report owners and recording audit logs; updated OpenAPI schema and generated types; verified with unit and integration test suites.

- Synthetic evaluation results, model cards, and dataset collection protocol (Sprint 3 AI handoff, AI-11, AI-12, FR-ANL-07): recorded synthetic evaluation benchmark results with baseline and ablation comparisons and acceptance gates in `docs/ai/evaluation.md` clearly labeled SINTETIS; authored initial model cards for locked models in `docs/ai/model-cards.md` (CLIP image encoder, multilingual CLIP cross-modal text encoder, multilingual e5 small text encoder, EasyOCR) covering revisions, licenses, CPU latencies, intended uses, and limitations; authored physical dataset collection checklist and freeze execution protocol in `docs/ai/dataset-collection.md`.

- Database foundation for AI monitoring (Sprint 3 AI handoff, AI-12, FR-ANL-07, FR-ADMIN-06): migration 00012 adding database views `ai_monitoring_assist_outcomes`, `ai_monitoring_dismiss_rates`, `ai_monitoring_confirmed_ranks`, `ai_monitoring_top1_histogram`, and composite summary view `ai_monitoring_summary`; documented Sprint 4 data foundations in `docs/ai/ai-features.md`; verified with integration test suite.

- Observability metrics for matching engine and ML inference (Sprint 3 AI handoff, AI-12, NFR-OBS-01 to NFR-OBS-03): added `ml_models_loaded`, `ml_inference_seconds`, and `ml_request_duration_seconds` to the ML service; added `ml_request_duration_seconds`, `match_pipeline_seconds`, `match_top1_score`, and `match_candidates_stored` to API platform metrics; wired Prometheus observation across HTTP client and worker scoring pipeline; verified with unit test suites.

- ML case clustering endpoint (Sprint 3 AI handoff, AI-11, US-48, FR-ANL-06, ML-02): implemented `POST /v1/cluster` using HDBSCAN from scikit-learn on L2-normalized vectors with configurable `min_cluster_size` and `min_samples`, returning cluster assignments, cluster membership probabilities, and the scikit-learn version; updated OpenAPI contract in `ml/openapi.json` and endpoint documentation in `ml/README.md`; verified with unit and contract test suites.

- Duplicate warning on found report form and officer duplicate review workspace (Sprint 3 Stage 3D, US-15, US-40, FR-FOUND-05, FR-OFF-05, AI-06): found report submission checks duplicate candidates via POST /assist/duplicates at the review step and displays warning card with match confidence, thumbnails, and dismiss or proceed choices; officer duplicate workspace at `/petugas/duplikat` featuring suggested, confirmed, and rejected duplicate link tabs, side-by-side report comparison, survivor report selection, merge confirmation dialog, reject action, and manual merge modal; verified with TypeScript typecheck, ESLint, Prettier, and Vitest unit tests.
- Duplicate detection and officer merge backend (Sprint 3 Stage 3D, US-15, US-40, FR-FOUND-05, FR-OFF-05, AI-06): migration 00011 adding duplicate_links table with uniqueness constraint on report pairs; POST /assist/duplicates running AI-06 duplicate detection gate (parent category, 72h window, 500m distance) and scoring (pHash <= 6 bits yields 1.0, token set semantic text similarity with image pHash distance fusion) returning up to 3 candidates with score >= 0.80; GET /officer/duplicates listing suggested duplicate links; POST /officer/duplicates/merge, POST /officer/duplicates/{id}/confirm, and POST /officer/duplicates/{id}/reject executing report merges with photos moved to survivor, custody cancelled on duplicate found report, conflicting open claims resolved, matches superseded, and audit logged; verified with unit and integration test suites.
- Web officer workspace dashboard and AI priority queue (US-38, FR-OFF-01, FR-OFF-02, FR-OFF-06, AI-09): officer home page at `/petugas` featuring summary counters for in-custody, retention due, with-finder, claims pending, and high-match candidates, campus-wide vs assigned-posts scope toggle, and the AI priority work queue with priority score badges, category tags, reason chips, and primary action links; verified with TypeScript typecheck, ESLint, Prettier, and Vitest unit tests.
- Officer workspace summary counters and priority queue endpoints (Sprint 3 Stage 3D, US-38, FR-OFF-01, FR-OFF-02, FR-OFF-06, AI-09): `GET /officer/summary` returning counters for items in custody, retention due within 7 days, with finder, pending claims, unclaimed items, items found today, and high-match candidates without claims with post and campus scoping; `GET /officer/queue` computing the fixed priority formula P = 0.35 * m + 0.20 * s + 0.20 * a + 0.25 * u across claim reviews, overdue custody, retention due, unclaimed dispositions, direct return reviews, and high-tier matches without claims, with reason chips, primary action labels, and action URLs; verified with unit and integration test suites.

- Load testing and end to end journeys J4 to J6 (Sprint 3 Stage 3E): Python standard library load testing runner verifying performance budgets of NFR-PERF-01 for public found listing (p95 <= 300 ms), authentication login (p95 <= 500 ms), and claims listing (p95 <= 300 ms); root Makefile load target; Playwright journeys J4 (officer requests additional information, claimant replies, officer verifies checklist and approves claim), J5 (claimant views handover code, officer attempts invalid code then confirms correct code, report owner confirms item receipt on lost report transitioning status to closed), and J6 (advancing retention past expiration on fake clock, officer triggers retention sweep, item transitions to unclaimed in post inventory, officer records disposition as donation); test endpoints for fake clock advance, inspection, reset, and manual retention sweep; confirm receipt dialog and custody disposition dialog with Indonesian UI strings; verified with axe accessibility and full test suites.
- Custody operations and lifecycle sweep (Sprint 3 Stage 3C, US-11, US-16, BR-CUS-02 to BR-CUS-06, BR-LST-02 to BR-LST-04, BR-FND-03): custody transfer between posts with audit event and target post validation; retention extension once per item by up to 30 days with reason; disposition of unclaimed items (donated, destroyed, returned_to_finder, transferred_to_unit) enforcing the identity category rule requiring transfer to issuing unit; revert unclaimed reports back to reported with 7 day retention; lost report keep-open, reopen within 30 days restarting matching, and confirmation of receipt upon handover; background River sweep workers for custody retention expiration, 7-day post officer retention warnings, 24h and 48h finder custody reminders, 72h overdue custody escalation, and lost report stale reminder (60 days) and automatic expiry (14 days grace); verified with full integration test suite.
- Web claims and handover flows (Sprint 3 Stage 3B): member claim pages for creating claims from match candidates or public found items, answering category verification questions, uploading evidence, tracking claim status, and messaging officers; officer claim review workspace with review checklist, internal notes, request information dialog, approval, rejection, and 6 digit handover code verification; integration with member header navigation, reports tab, candidate view, and public found items; verified with unit tests and complete Playwright end to end journeys under axe accessibility checks.

- Claims and handover backend (US-25 to US-29, US-31, FR-CUST-04): a member claims from a candidate or the public listing with answers to the category questions and up to three evidence images, officers pick up, ask for information, check the AI-08 checklist, approve, reject with a reason, revoke and reopen, and the claimant can answer, message and withdraw. Approval needs the item at the post, puts rival claims on hold and issues a handover code derived from a stored nonce, so no code is ever stored. Five wrong codes lock it, a handover releases custody and closes the report pair in one transaction, a walk in visitor claim needs a second officer. Claims that stall expire after 7 days with a one day warning, a rejected claimant is restricted after three rejections in 30 days. Rejected claims record a feedback signal on the pair. Migration 00010 adds claims, messages, checks and dispositions.
- Evaluation harness (`ml/eval`): the Python fusion reference that agrees with the Go scoring on the shared fixture to six decimals, the scenario generator, metrics with bootstrap intervals, baselines B0 to B2 and ablations A1 to A7, component tasks T4 to T8, calibration on `val` that writes a new fusion document, the acceptance gate table, a split leakage check, a latency script, and `make eval`. It runs end to end on a synthetic dataset in the tests, the real run waits for the frozen dataset.
- Inference service: `/v1/embed/image`, `/v1/embed/text` and `/v1/analyze/image` with the pinned CLIP, multilingual CLIP, multilingual e5 and EasyOCR weights, zero shot categories cached by prompt set hash, colour naming by k-means in CIELAB and CIEDE2000, brand matching with rapidfuzz, masked visible text, problem details errors, a committed `ml/openapi.json` contract, golden tests against the real weights and a Compose image that downloads the weights into a volume.
- Report assistant (US-23, US-24): after the first photo both report forms fill an empty category, colours and brand from the photo with a Saran otomatis label, offer two alternative categories, and the found form drafts the public description from the template and can draft it again. Visible text is masked and never enters the draft for an identity card. What the reporter did with each suggestion is stored as `assist_outcome`. When the inference service is missing, down or slower than 8 s the form only shows that suggestions are unavailable.
- Master data administration (US-07): administrators create, edit and switch off categories with their verification questions and photo prompts, compatibility pairs, zones, locations with aliases and posts with opening hours, under Admin, Data Master. Switched off rows leave the pickers and stay on existing reports, every change is audited field by field with its value before and after, and coordinates outside `campus.bounding_box` are refused once that setting exists.
- Matching engine v1 (US-14, US-18 to US-22): every lost and found report is scored when it is created, edited or closed, against the structured pool and the image, cross modal and text generators, with the fusion of matching-engine sections 6 and 7, the identity name rule, top ten ranks per side and the L2, L3, F2 and F3 status effects. Owners see medium and high candidates with a tier, a percentage, four score bars and Indonesian reason lines, can mark one as theirs or hide it, and hear about strong candidates at most once per throttle window. Finders learn that an owner may be looking, officers see every component number with a switch for weak candidates, and can rematch and dismiss. With an inference service configured, embedding and image analysis jobs record the model version of every vector and refuse a revision that is not active, without one matching runs on the structured signals. A 02:00 sweep rescores every open report.
- Integration test helpers: `testenv.Drain` works queued River jobs with the real workers, `testenv.NewFakeML` serves deterministic vectors with switchable failures, and `testenv.WithML` points the inference client at it.
- Profile and security pages: edit name, phone and NIM or NIP, change the email through a confirmation link sent to the new address, change the password with the current one while every other session signs out, and list or revoke signed in devices with a readable device label and the network prefix of the address (US-05).
- In app notifications with a live stream: a bell with the unread count and the latest items, a Notifikasi page and phone tab, Server Sent Events woken by PostgreSQL `LISTEN` and resumed with `Last-Event-ID`, read and read all, optional groups switchable per channel in Profil, Notifikasi, emails that wait for the end of the quiet hours, a 07:00 officer digest sent only when it has content, and 180 day retention. The finder is now notified when a post receives the item (US-30). Passive requests never keep an idle staff session alive.
- Continuous integration per area: the API workflow checks format, vet, lint, that generated code matches `openapi.yaml` and the queries, known vulnerabilities, unit and integration tests; the web workflow checks the generated client, format, lint, types, unit tests and the build; the ML workflow runs ruff, mypy and pytest; the end to end workflow starts the stack and runs the Playwright journeys. Dependabot groups weekly updates, `CODEOWNERS` routes reviews to the area owner.
- Development stack: one `make dev` builds the images, starts PostgreSQL 18 with pgvector, migrates, seeds with the development overlay, then runs api, worker, ml, web, Mailpit and Caddy on one origin at port 8000, every service with a health check. `make env` writes random local secrets, host ports are configurable, and the root `Makefile` carries the generate, check, test and end to end targets.
- Web app v0.1 on Next.js 16.4: registration with email verification, sign in and out, password reset, staff invitations; four step lost and found report forms with photo upload, HEIC conversion, drafts that survive a reload and idempotent submission; report detail with status timeline, editing and closing; the public found listing; officer receive desk, post inventory, search by code and text, and proxy reports; admin user list with invitations, roles and posts; privacy, terms and help pages. Every string in `messages/id.json`, 39 unit tests and Playwright journeys with axe on phone and desktop.
- Inference service skeleton: FastAPI app factory, bearer token on `/v1`, health and readiness tied to `models.lock.json`, a lock that refuses floating revisions, JSON logs, Prometheus metrics, strict mypy, ruff and pytest.
- Go API v0.1: OpenAPI 3.0.3 contract with generated strict server, one contract middleware for permissions and request validation, accounts with argon2id and server side sessions, invitations and user administration, master data with ETags and an idempotent seed command, photo upload that strips metadata into four variants, lost and found reports with idempotent creation, editing, closing, history and audit, the public found listing, proxy reports, custody intake with retention, post inventory, officer search, email jobs on River, and 26 integration tests against PostgreSQL.
- Database init script creating the owner, application and read only roles with default privileges, shared by Compose and the integration tests, and a development seed overlay with 41 OpenStreetMap landmarks, five zones and six pilot posts, used only where `data/seed` has no reviewed file.
- Project plan, team charter, risk register and definition of done.
- UX, engineering and operations guides: information architecture, UI guidelines, tech
  stack, repository structure, coding standards, testing strategy, local development,
  deployment, observability and runbook.
- AI specification: matching engine, per role AI features, evaluation protocol, dataset
  collection guide and model lifecycle, plus the seed and dataset contracts in `data/`.
- Architecture overview, data model, REST API catalog, security and privacy design, and
  ten architecture decision records.
- Software requirements specification with business rules, state machines, permission
  matrix, notification catalog and traceability matrix.
- Product requirements, user stories, success metrics and glossary.
- Repository skeleton for `api/`, `web/`, `ml/` and `deploy/`, contribution guide, security
  policy, issue and pull request templates.
- Shared synthetic data fixtures for matching fusion, privacy masking, and fuzzy name
  matching, including a validator used before service contract tests consume them.
- Added deterministic synthetic image fixtures for colour, EXIF, OCR, and masking tests.
- Added a CI check that detects drift in reviewed fixture image bytes with SHA-256 hashes.
- Added an intake privacy regression test using the shared synthetic GPS fixture.
- Added a checker for blank full and trial evaluation-template scaffolds before collection.
- Clarified the root status so it reflects the implemented data foundation without implying
  that the other services have started.
- Added a dedicated data CI workflow for seed, fixture, and data-script validation.
- Added a safe freeze step that publishes only a validated private evaluation manifest.
- Added an auditable raw-response option for the OpenStreetMap location review flow.
- Added a provenance check that catches drift between raw location data and its review CSV.
- Added bounded retry handling for transient OpenStreetMap extraction failures.
- Ensured location-review extraction excludes candidates outside the requested bbox.
- Improved OpenStreetMap location-kind classification for public amenities and institutions.
- Added a private counts-only summary for planning the manual location review.
- Added private dataset gates for the 1080 px image short-side minimum and exact duplicate
  image content.
- Added a cross-item perceptual-hash gate for visually near-identical private images.
- Hardened evaluation split verification against edits that change stratum allocation.
- Added a private two-annotator label agreement checker with aggregate Cohen's kappa.
- Hardened reviewed-location promotion against empty output and invalid OSM provenance.
- Added a blank trial exit-review log template for the private collection workflow.
- Added an atomic generator for the private trial-review workspace and annotation sheets.
- Added a final private trial-review gate for completed collection records.
- Bound trial completion to a current agreement report generated from both annotation sheets.
- Added a read-only private evaluation freeze gate that aggregates the required checks.
- Expanded orchestration coverage for the private trial/full evaluation freeze gate.
- Stabilized synthetic fixture image CI validation with reviewed SHA-256 integrity hashes.
- Added an explicit campus-bbox check for complete location seeds.
- Made evaluation image intake convert accepted inputs to canonical JPEG files.
- Added a safe local evaluation fetcher that verifies manifest-listed files before copying
  them from a private dataset sync or mount.
- Prevented manifest publication and evaluation fetch output from being written inside a
  private source dataset directory.
- Made seed foundation checks reject partial deferred seed files before complete-mode
  validation is explicitly requested.
- Made manifest publication print the dataset SHA-256 for reproducible handoff records.
- Hardened private manifest checks against symlink dataset roots and manifest files.
- Hardened seed validation against symlink roots and files outside the checkout.
- Required location promotion to recheck provenance against the saved raw Overpass response.
- Made negative campus-bbox examples portable across supported Python CLI parsers by
  using the `--bbox=...` form in tests and handoff documentation.
- Rejected manifest-listed paths that traverse symlinked directories, keeping private
  dataset validation inside the dataset root.
- Applied symlink-parent checks to private dataset intake, fetch, and manifest
  publication before any source or output path is resolved.
- Extended symlink-parent checks to private template/workspace generation and their
  validators, keeping all collection roots tied to the requested directory.
- Tightened category-seed validation for empty trees, field types, prompt/question text,
  and optional positive retention overrides.
- Rejected normalized collisions between brand names/aliases and between synonym terms,
  preventing ambiguous master-data lookups.
- Extended output parent-path safety to OSM review artifacts, location promotion, private
  summaries/reports, evaluation splits, and fixture regeneration.
- Hardened settings domain validation and complete-seed location aliases against invalid,
  blank, or normalized duplicate lookup values.
- Hardened complete post and model-version validation against blank contact values,
  overlapping hours, and malformed component/status types.
- Hardened complete post validation to report malformed location references instead of
  failing with an internal type error.
- Wrapped malformed YAML seed input as a readable validation error.
- Hardened fixture validation to report malformed JSON and non-object case entries
  without an internal traceback.
- Hardened shared seed/trial CSV reading to reject rows with missing or extra fields.
- Extended malformed-row rejection to auxiliary distractor, duplicate-pair, and query CSVs.
- Hardened private template checks to reject rows with missing or extra CSV fields.
- Hardened annotation and trial review-log readers to reject malformed rows before
  agreement and final-review checks.
- Hardened location review and promotion readers to reject malformed CSV rows before
  provenance or zone checks.
- Hardened prepared-dataset metadata readers to reject malformed item and label rows
  before content checks.
