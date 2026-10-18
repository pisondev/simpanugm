# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Security

- The API pins the go1.26.9 toolchain, which fixes the HTTP/2 HPACK race GO-2026-6617, and lifts `golang.org/x/image` from a 2019 snapshot to v0.47.0, closing eight TIFF and BMP decoding denial of service advisories reachable through image decoding, plus `moby/go-archive` to v0.3.3 for the integration tests. `govulncheck` reports no reachable vulnerability.

### Changed

- Dependabot proposes only patch and digest updates for the Go, Python and Node base images. A runtime upgrade changes code, CI and docs together, so it is made by hand.
- The data model lists the `custody_keep` and `close_note` columns of found reports that the migrations already carry.
- `GET /api/v1/posts` is public, so the help page lists posts and opening hours before sign in. Every other master data read still needs a session.

### Fixed

- Resending a verification link or starting an email change no longer fails for an account without earlier tokens of that kind: the issue statistics default to the epoch instead of minus infinity, which Go cannot represent.
- `api/internal/media` and `api/internal/platform/storage` are in the repository: unanchored `media/` and `storage/` ignore rules had kept them out, so a fresh clone did not build. The rules are anchored to the root now.
- The API image starts again: it is built with the `nodynamic` tag, so `gen2brain/webp` stays on its WebAssembly encoder instead of linking libc through purego, which `distroless/static` cannot load. A root `.dockerignore` replaces the one in `api/` that the root build context never read.
- The fetch test compares resolved paths, so it also passes on Windows where temporary directories have short names.
- Masking now hides every 16 digit run, so NIK numbers are covered although they fail the Luhn check, and treats digits grouped by single spaces or hyphens as one run, as BR-PRV-03 requires.

### Added

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
