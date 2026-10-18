# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- `GET /api/v1/posts` is public, so the help page lists posts and opening hours before sign in. Every other master data read still needs a session.

### Fixed

- The fetch test compares resolved paths, so it also passes on Windows where temporary directories have short names.
- Masking now hides every 16 digit run, so NIK numbers are covered although they fail the Luhn check, and treats digits grouped by single spaces or hyphens as one run, as BR-PRV-03 requires.

### Added

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
