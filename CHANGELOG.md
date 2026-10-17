# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

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
- Added a CI check that detects drift in generated fixture image bytes.
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
