# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Add a slide-aligned officer dashboard with six explicit feature names, a local officer illustration, real scoped queue data, case selection and direct verification and handover shortcuts.
- Add a slide-aligned finder dashboard with private-safe match hints, actual custody status, duplicate-check prerequisites and the supplied local character illustration.
- Add a member dashboard mode switch and choose the finder view for accounts with only active found reports, preserving the shared member role and existing permissions.
- Add an owner dashboard aligned with the presentation's six feature names, local campus and student illustrations, report-specific candidate previews, explainable signals and claim follow-up links.
- Enable the existing editable description template and explicit redraft action for photo-assisted lost reports, matching the owner's presentation flow without an additional model.
- Keep matching and claim actions visible with clear prerequisites when reports or candidates are absent, and repeat the feature names at their destination screens.
- Align data seed model identities with `ml/models.lock.json`.
- Render the admin hotspot view as an interactive Mapbox Streets map using the reviewed campus coordinates.
- Run complete demo seed validation in the data CI workflow with the reviewed demo bbox.
- Add an Hestia production profile with opt-in local ML, memory caps and separate live environment files for 4 GiB VPS deployments.
- Seed ready-to-use library handover, physical receipt and owner information-response stories; verify replay safety and complete all three through the real API in integration tests.
- Expand demo master-data vocabulary with common campus synonyms and Indonesian brand aliases.
- Align the Axelle master-data seed with the 41 OSM landmarks and six pilot posts used by the
  demo overlay, while marking the combined bbox as reviewed for demo and production review
  pending.
- Add an explicit phase 2 screen and data playbook, and align the live demo script with actual routes, editable model suggestions and recording commands.
- Cover desktop and 360 px reference screenshots with axe checks, real model suggestions, retry and reduced-motion failure handling.
- Polish the public landing, owner candidates and photo-assisted found report flow with honest AI status, editable suggestions and animated evidence.
- Expand the attributed demo photo pool to 159 assets, seed 90 deduplicated background finds, and verify a real misclassified holdout through the member lost-report flow.
- Polish public help, privacy and terms pages with readable post hours and responsive legal content while retaining public account links.
- Add the local auth texture and responsive card treatment while preserving validation, demo role entry and expired-link recovery.
- Polish account recovery and invitation screens with inline token guidance and the shared password visibility control.
- Polish member discovery with clearer report cards, responsive found-item filtering and a photo-free lost-report flow with reload-safe drafts.
- Polish member report details and editing with clearer private evidence, calmer timelines, reliable photo zoom and retained drafts on edit conflicts.
- Polish member claim details, secure handover codes, claim conversations and notification recovery while retaining notification read behavior.
- Polish member profile, active-session and notification settings with clear destructive-action confirmations.
- Polish officer priority review, report-code lookup, semantic search and duplicate controls for desktop and phone screens.
- Add officer inventory cards for phone screens and guarded custody transfer, retention extension and disposition actions.
- Add semantic surfaces, local Inter, accessible motion primitives, finite skeleton shimmer and a reusable evidence comparison panel.
- Document visual and motion research for the three phase 1 reference screens, including offline, performance and accessibility budgets.

### Changed

- Separate complete normal and demo browser suites, wait for finite entrance animations before axe scans, and give the multi-page master-data audit its own test budget.
- Fix the narrow officer claim detail layout, select a seeded owner report that actually has candidates in reference journeys, and validate the complete normal and demo suites together.
- Prepare Hestia and Cloudflare configuration for `simpanugm.pisondev.com`, align production image paths with the release workflow and fix the migration service invocation.
- Make IP and email/IP-pair login budgets configurable, retaining production defaults of 20 and 5 per 15 minutes and raising both to 500 only in development; retain failed-password lockout.
- Replace the wallet, power bank and backpack heroes after testing three Commons alternatives each; the fixed 14-image hero sample improves from 11 to 14 correct Top 1 categories without changing prompts.
- Reduce CPU OCR input to 768 px after ML telemetry identified OCR as the image-assistant latency bottleneck; real-weight text and number-masking checks and desktop/phone reference journeys pass.

## [1.0.0] - Draft

First release candidate of SimpanUGM (Sistem Informasi Manajemen Penemuan dan Pengembalian Barang, Universitas Gadjah Mada), a campus lost and found platform featuring multimodal AI matching with human in the loop verification by campus security officers.

### Added

- Added a public-source Axelle data MVP baseline for the assignment: five project zones,
  25 OSM-derived campus locations, the official K5L emergency contact, and exact public
  model/package revisions, with private participant data kept outside Git.
- Core Lost and Found reporting: four step lost and found reporting forms with optional or required photo uploads, HEIC conversion, client side draft persistence, idempotent report creation, status history timeline, and public found item browsing without private details (US-08 to US-14, US-17).
- Multimodal AI Matching Engine: fusion scoring across visual embeddings (CLIP ViT-B/32), semantic text embeddings (multilingual e5 small), spatial distance, temporal proximity, and category compatibility; cross modal text to photo matching; top ten candidate ranking; explainable candidate view with percentage scores, tier badges, and Indonesian explanation bars; candidate dismissals and feedback logging (US-18 to US-22, AI-01, AI-02).
- AI Report Assistant: automatic suggestion of category, colour, and brand from uploaded photos within 4 s; auto drafted public descriptions; automatic masking of 16 digit identity numbers (NIK and KTM) in visible text (US-23, US-24, AI-03, AI-04).
- Claims and Verification: member claim submission with category verification questions and evidence uploads; public listing direct claim flow with inline lost report creation for members without prior lost reports (US-26); officer claim review workspace with AI-08 verification checklist, request info workflow, and claim approval and rejection; 6 digit secure handover codes generated from stored nonces with automatic locking after 5 invalid attempts and owner receipt confirmation (US-25 to US-29, US-31).
- Custody and Storage Operations: post inventory tracking, intake receipt desk, custody transfer between campus posts, retention extension up to 30 days, unclaimed item disposition with identity issuing unit transfer rule, and automated background sweeps for expiration, reminders, and escalation (US-30, US-32, US-33, US-34).
- Officer Workspace: campus and post scoped summary counters, AI priority work queue with weighted priority formula (P = 0.35m + 0.20s + 0.20a + 0.25u), side by side match review board with rematch, pin and unpin, and claim invitations, trigram and code search, duplicate report detection and merge workspace, and natural language semantic search via dense e5 embeddings and reciprocal rank fusion (US-37 to US-41, AI-06, AI-07, AI-09, AI-13).
- Administration, Moderation, and Audit: account suspension with immediate session termination; abusive report hiding with candidate supersession; runtime business settings adjustment without redeployment; tamper evident audit logging with date, actor, and entity filtering and CSV export; self service account deletion with anonymization (US-42 to US-45).
- Analytics and AI Monitoring: campus hotspot analytics map with privacy suppression under 3 reports and surge detection; operational trends with weekly progression, hour by weekday matrix, and monthly CSV export; case clustering via HDBSCAN on vector embeddings; AI monitoring dashboard tracking assist outcomes, tier dismiss rates, confirmed ranks, top-1 histogram, score PSI drift detection, and candidate fusion version activation with zero downtime rollback (US-46 to US-49, AI-10, AI-11, AI-12).
- Notifications and Communication: real time in app notification bell powered by PostgreSQL LISTEN and Server Sent Events; multipart Indonesian email notifications with quiet hours queuing and 07:00 officer digests (US-35, US-36).
- Platform Operations and Observability: multi service Docker Compose development and production stacks; Prometheus metrics across API, web, ML, database pool, and River queue; Grafana dashboard suites for Overview, Jobs and Matching, Database, and Host; Prometheus alert rules for site liveness, error rates, queue stalls, and backup age; automated pg_dump backup script with retention rotation and restore script with database integrity verification; OCI container release workflow for GHCR; structured JSON logging with request ID propagation across all services (US-50 to US-55).
- Offline Evaluation Harness: Python evaluation suite (`ml/eval`) with synthetic benchmark dataset (SINTETIS), baseline comparisons (B0 to B2), ablations (A1 to A7), and learned fusion logistic regression benchmark meeting PRD section 8 metrics.
- Scheduled weekly workflow (`.github/workflows/ml-golden.yml`, with workflow dispatch) running heavy real weight inference golden tests with `models.lock.json` caching.
- Real evaluation runner pipeline (`make eval-real`, `ml/eval/run_real.py`) with automated `check_freeze` verification, `val` split calibration, `test` split gate execution, packaged run artifact logging, and authoritative report generation labeled RESMI or SINTETIS.
- Repeatable learned fusion experiment script (`make eval-learned-fusion`, `ml/eval/learned_fusion.py`) with real weight execution support and strict gate decision rule requiring verified gains on frozen test data before activation.
- Containerized CPU latency benchmark support in ML Docker container with in-memory image fallbacks, container mount discovery, VPS execution runbook, and production latency tracking template tables.
- Live demonstration stack seeder (`make demo` and `simpan demo` CLI command) loading realistic scenarios, demo accounts for all actors, lost and found reports, claims across all lifecycle stages, duplicate links, and analytics view refreshes (W1).
- Demo photo pack and manifest (`data/demo/`): 52 curated and derived open-license photographs meeting strict licensing (CC0, CC-BY, CC-BY-SA, Public Domain) and privacy rules (no faces or identity documents), with automatic fetching from Wikimedia Commons, variation derivation, manifest validator (`validate_manifest.py`), unit tests (`test_demo_manifest.py`), credits documentation (`docs/project/demo-credits.md`), and automated integration into the demo seeder (W2).
- Demonstration presentation run kit and fallback recording suite (W5): updated `docs/project/demo-script.md` with 12 minute allocation, exact click paths, and Indonesian speaker notes across all four actors, paired with Playwright fallback video and screenshot recording suite (`web/tests/e2e/demo-record.spec.ts`).
- UX and visual polish pass per actor with accessibility compliance (W3): live demo banner with Mailpit link, one-click quick login buttons for demo actors, missing photo fallback icon, tokenized WCAG AA contrast compliance (`text-success-text`, `text-warning-text`), keyboard focusable scrollable containers (`tabIndex={0}`), and admin loading skeleton (`loading.tsx`).
- AI moments visibility and verification journeys (W4): end to end Playwright suite (`web/tests/e2e/demo-ai.spec.ts`) validating every AI interaction across all four actors (Finder assistant suggestions and match hint, Owner candidate tier and 4 score bars, Officer priority queue and semantic search, Admin monitoring and hotspot surge alerts) with screenshot capture and accessibility checks.
- Light usability study kit and feedback log (W6): documented 10 minute testing protocol and task sheets for 4 students with tracked usability findings and fixes in `docs/ux/usability-kit.md`.
- Interactive demonstration access and verification (W8): visitor self registration with email capture inbox (Mailpit link in `DemoBanner`), one click demo actor sign in buttons in `LoginForm`, idempotent admin demo reset command, and automated integration test asserting demo mode refusal in production.
- Live demonstration reliability and pre-flight health check (W7): created `deploy/scripts/demo_check.py` and `make demo-check` target verifying container liveness, HTTP health and readiness probes (direct API `/healthz`, `/readyz`, Web `/healthz`, ML `/readyz`, Mailpit `/api/v1/info`), neural model residency (CLIP and E5), authenticated smoke tests across all four demo actors, container memory footprint within laptop budget, and offline readiness with local assets.

### Changed

- The remaining schedule follows `docs/project/demo-first-plan.md`: a scripted, good looking demo with visible AI for every actor comes first, hosting and the formal evaluation are parked.
- Audited model cards and evaluation documentation against empirical lock files, latency benchmarks, and synthetic validation results, establishing docs/ai/evaluation-report.md with unverified real metrics strictly marked TERTUNDA.
- Requirement traceability matrix expanded with full acceptance criteria verification matrix covering user stories US-01 through US-55.
- Operations runbook aligned with automated backup, restore, and deployment scripts; established restore drill protocol template in `docs/operations/restore-drill.md` without claiming unperformed staging drills.
- Configured end to end test workflow to trigger on push to main in addition to pull requests, schedule, and workflow dispatch.
- Local compose environments separated into `deploy/compose.dev.yaml` for real system clock development and `deploy/compose.e2e.yaml` for Playwright end to end testing with isolated time travel.
- Campus master data seeded with 41 OpenStreetMap landmarks, 5 zones, and 6 pilot posts with bounding box validation.

### Fixed

- The demo seeder finds `data/demo/SOURCES.csv` from any working directory upwards (or `SIMPAN_DEMO_DIR`), so its integration test passes in CI, and the photo holdout journey runs only with `DEMO_E2E=1` because it needs the `make demo` story.
- The lost report journey picks the special features field by its textbox role, because the stepper now labels its steps and the plain label matched two elements.
- Demo mode of the web app (banner and one click demo sign in) is now opt in at build time with `NEXT_PUBLIC_DEMO_MODE=true`, set only by the dev and demo Compose override, so a production image never shows it. End to end runs use one worker so the time travel journey cannot expire invitations of other tests, and the demo journeys run only on request with `make demo-test` and `make demo-record`.
- Fixed operator precedence in GetMediaForAccess query by explicitly grouping candidate match criteria, preventing confirmed matches from leaking media access, and added media access matrix integration tests.
- Fixed restore script database verification to query system_settings instead of settings, removed invalid role parameter on pg_restore, and validated deployment artifacts with actionlint, shellcheck, and local database restore simulation.
- Reused authenticated session cookie across public and claims scenarios in the load test runner, avoiding excessive login attempts and false 401/429 failures against rate limited endpoints.
- Admin analytics pages pass the contrast and scrollable region checks of axe (success text token, keyboard focusable table containers), and the end to end admin bootstrap runs with `--no-deps` so parallel workers no longer fight over the migrate container.
- The web build works again: a `middleware.ts` next to `proxy.ts` made Next.js 16 refuse to build, the request id is set in `proxy.ts` alone.
- Migrations up and down verified: automated test `TestMigrationsUpDownUp` validates all 13 goose migrations rolling up, down to 0, and up again on a clean PostgreSQL instance.
- Corrected post restore report verification query in `deploy/scripts/restore.sh` to count from `lost_reports` and `found_reports`.
- Resolved River job deduplication interference on `MatchSweep` during model activation and rollback.
- Added automatic overlay directory discovery in `defaultOverlayDir()` and Dockerfile assets copy for standalone demo seeder execution.
- Masking hardened to cover all 16 digit runs including hyphenated and spaced formats and NIKs without Luhn checksums.
- WebAssembly fallback enabled for `webp` image encoder under distroless API container.

### Security

- The API toolchain moves to Go 1.26.10, which fixes the `crypto/x509` advisory GO-2026-6809 that `govulncheck` reports as reachable.
- Pre release dependency vulnerability audit: zero known vulnerabilities across production dependencies verified via `pnpm audit --prod`, `pip-audit`, and `govulncheck`.
- Toolchain pinned to Go 1.26.9, closing HTTP/2 race GO-2026-6617, and `golang.org/x/image` lifted to v0.47.0.
- Strict-Transport-Security (HSTS), Content-Security-Policy, and CORS headers enforced in reverse proxy and HTTP middleware.
- Role based access control (RBAC) enforced on all endpoints with `HttpOnly`, `SameSite=Lax` cookie sessions and CSRF protection.
- Polish officer claim review with required-check guidance, numeric handover validation, and clear lock and code-regeneration recovery.
- Verify and capture officer proxy lost/found reporting and staff record screens at desktop and phone widths.
- Make top-1 monitoring histogram values available as accessible meters and verify fusion controls without activating a model.
- Make hotspot markers keyboard operable with privacy-safe labels and add accessible horizontal scrolling to dense analytics tables.
- Improve keyboard access and narrow-screen scrolling for admin user and audit tables.
- Add keyboard-scrollable data regions to category, compatibility and location master tables.
- Improve narrow-screen administration of zones and posts, and verify opening-hour validation without saving invalid data.
- Label keyset pagination navigation and verify offline recovery, 404 guidance and role navigation on desktop and phone.
- Expire the lost-report candidate search state after two minutes without requiring a page refresh.
