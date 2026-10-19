# Traceability | SimpanUGM

Every item of the concept document (Google Doc "REVISI - Grup A", tab Draft) traced to the
requirements that implement it, the stories that build it, the code that holds it, and the
tests that prove it. A pull request that adds a requirement adds its row here.

Test levels: **U** unit, **I** integration, **C** contract, **E** end to end journey
(testing-strategy section 3), **V** offline evaluation, **S** usability study.

## 1. Functional specifications

| Concept item | Requirements | Stories | Code | Tests |
|---|---|---|---|---|
| Lost Item Reporting and Search | FR-LOST-01 to FR-LOST-14, FR-SEARCH-05, BR-LST | US-08 to US-12 | `api/internal/reports`, `web/src/app/(member)/lapor/hilang` | U, I, E J2 |
| Found Item Reporting | FR-FOUND-01 to FR-FOUND-13, BR-FND | US-13 to US-17 | `api/internal/reports`, `api/internal/custody` | U, I, E J3 |
| AI Assisted Multimodal Matching | FR-AI-01 to FR-AI-12 | US-18 to US-22 | `api/internal/matching`, `ml/app` | U, I, C, V T1 to T3, E J2 |
| Claim and Ownership Verification | FR-CLAIM-01 to FR-CLAIM-12, FR-CUST-03, FR-CUST-04, BR-CLM, BR-HND | US-25 to US-29, US-31 | `api/internal/claims`, `api/internal/custody` | U, I, E J4, J5 |
| Case Status and History Tracking | state-machines L, F, C, M, K, FR-AUDIT-04, FR-LOST-11, FR-LOST-12, FR-CUST-06 | US-09, US-11, US-29, US-33 | `machine.go` in each module | U per transition, E J6 |
| Alerts and Notification System | FR-NOTIF-01 to FR-NOTIF-07, notifications catalog | US-22, US-35, US-36 | `api/internal/notify` | U, I, E J1, J2 |
| Search and Operational Dashboard | FR-SEARCH-01 to FR-SEARCH-06, FR-OFF-01 to FR-OFF-06 | US-37 to US-41 | `api/internal/search`, `api/internal/officer`, `web/src/app/(officer)/petugas` | U, I, E J4 |
| Administration, Moderation, and Settings | FR-ADMIN-01 to FR-ADMIN-05, BR-RET-05 | US-42 to US-45 | `api/internal/server`, `web/src/app/(admin)/admin` | U, I |
| Analytics, Hotspots, and Case Clustering | FR-ANL-01 to FR-ANL-07, FR-ADMIN-06, FR-ADMIN-07, AI-10 to AI-12 | US-46 to US-49 | `api/internal/analytics`, `ml/app/routers/cluster.py`, `web/src/app/(admin)/admin` | U, I, C |
| Platform Operations and Deployment | NFR-OBS-01 to NFR-OBS-03, NFR-SEC-01 to NFR-SEC-10, NFR-AVAIL-04, NFR-AVAIL-05, NFR-PORT-01 | US-52 to US-55 | `deploy/`, `api/internal/platform/metrics` | U, I, load |

## 2. System features

| Concept item | Requirements | Stories | Code | Tests |
|---|---|---|---|---|
| Visual Similarity Matcher | FR-AI-01, matching-engine 6.1 | US-18, US-19 | `ml/app` image encoder, `matching/scoring.go` | U golden embeddings, V ablation A1 |
| Semantic Description Matcher | FR-AI-02, matching-engine 6.2 | US-18, US-19 | `ml/app` text encoders, `matching/scoring.go` | U, V ablations A2, A7 |
| Context Aware Match Scoring | FR-AI-04, FR-AI-05, matching-engine 6.3 to 7 | US-19 | `matching/fusion.go`, `ml/eval/fusion.py` | U shared fixtures, V ablations A3 to A6 |
| AI Assisted Item Description | FR-ASSIST-01 to FR-ASSIST-05, AI-03, AI-04 | US-23, US-24 | `api/internal/assist`, `ml/app/analysis` | U, I, V T4 to T6 |
| Explainable Match View | AI-02, FR-AI-12, BR-VIS-04 | US-20 | `matching/explain.go`, `web/src/components/match` | U, snapshot, S |
| Lost and Found Hotspot Analytics | FR-ANL-01 to FR-ANL-05, AI-10 | US-46, US-47 | `api/internal/analytics`, `web/src/app/(admin)/admin/analitik` | I, E |
| Officer Semantic Search | FR-SEARCH-04, AI-13 | US-41 | `api/internal/search`, `web/src/app/(officer)/petugas/cari` | U, I |
| Case Clustering Engine | FR-ANL-06, AI-11 | US-48 | `ml/app/routers/cluster.py`, `api/internal/analytics` | U, I, C |
| AI Monitoring and Drift Detection | FR-ANL-07, FR-ADMIN-06, AI-12 | US-49 | `api/internal/analytics`, `web/src/app/(admin)/admin/ai` | U, I |
| Cross Platform Responsive Dashboard | UI-01 to UI-05, NFR-USE-01 to NFR-USE-05 | every `web` story | `web/` | E with axe, S |

## 3. Module specifications

| Concept module | Requirements | Code |
|---|---|---|
| Authentication and RBAC | FR-AUTH-01 to FR-AUTH-16, permissions | `api/internal/auth`, `api/internal/access` |
| Lost and Found Reporting | FR-LOST, FR-FOUND | `api/internal/reports` |
| Media and Evidence | FR-MEDIA-01 to FR-MEDIA-09 | `api/internal/media`, `api/internal/platform/storage` |
| AI Matching Engine | FR-AI, matching-engine | `api/internal/matching`, `ml/` |
| Claim and Verification | FR-CLAIM, FR-CUST | `api/internal/claims`, `api/internal/custody` |
| Notification | FR-NOTIF | `api/internal/notify` |
| Database and Audit | data-model, FR-AUDIT-01 to FR-AUDIT-04 | `api/migrations`, `api/internal/audit` |
| Administration and Analytics | FR-ADMIN-01 to FR-ADMIN-08, FR-MD, FR-ANL | `api/internal/admin`, `api/internal/masterdata`, `api/internal/analytics` |

## 4. User roles

| Concept role | System role and scope | Requirements |
|---|---|---|
| Pemilik, pelapor barang hilang | `member`, scope `own` on lost reports, `matched` on found reports | permissions 3.2, BR-VIS 5.1 and 5.2 |
| Penemu barang | `member`, scope `own` on found reports | permissions 3.2, BR-VIS 5.2 |
| Petugas keamanan, Lost and Found officer | `officer`, scope `all` for reading, `post` for custody | permissions 3.5, BR-CUS-03 |
| System administrator | `admin` | permissions 3.6 |

## 5. AI per user

| Concept feature | Id | Stories |
|---|---|---|
| Visual Match Finder | AI-01 | US-19 |
| Explainable Match | AI-02 | US-20 |
| Smart Report Assistant | AI-03 | US-23 |
| AI Assisted Item Description | AI-04 | US-24 |
| Instant Match Suggestions | AI-05 | US-14 |
| Duplicate and Related Report Detection | AI-06 | US-15, US-40 |
| AI Match Review Board | AI-07 | US-39 |
| Verification Assistant | AI-08 | US-26 |
| AI Priority Queue | AI-09 | US-38 |
| Lost and Found Hotspot Analytics | AI-10 | US-46 |
| AI Case Clustering | AI-11 | US-48 |
| AI and Matching Monitoring | AI-12 | US-49 |
| Semantic search, added by the team | AI-13 | US-41 |

## 6. Data specifications

| Concept data group | Tables |
|---|---|
| User and Access Data | `users`, `sessions`, `auth_tokens`, `officer_posts` |
| Lost Item Report Data | `lost_reports`, `lost_report_places` |
| Found Item Report Data, including storage status and receiving officer | `found_reports` (`custody_status`, `received_by`, `holding_post_id`), `custody_events`, `duplicate_links` |
| Media and AI Feature Data | `media`, `image_attributes`, `image_embeddings`, `text_embeddings_e5`, `text_embeddings_clip` |
| AI Match Data, including model and version metadata | `matches` (`fusion_version_id`, `component_versions`, `pinned_at`), `model_versions`, `match_feedback`, `ai_monitoring_views` |
| Claim and Verification Data | `claims`, `claim_checks`, `claim_messages`, `media` with `claim_id` |
| Notification and Audit Data | `notifications`, `audit_logs`, `status_history` |

## 7. Case statuses

| Concept status | Lost report state | Found report state |
|---|---|---|
| Reported | `reported` | `reported` |
| Potential Match | `potential_match` | `potential_match` |
| Under Verification | `under_verification` | `under_verification` |
| Claimed | `claimed` | `claimed` |
| Returned | `returned` | `returned` |
| Closed | `closed` with a reason | `closed` with a reason |
| added by the team | none | `unclaimed`, plus the separate custody lifecycle |

## 8. Non functional requirements to verification

| Requirement group | Verified by |
|---|---|
| NFR-PERF | k6 scenarios of testing-strategy section 5 |
| NFR-AVAIL | restart tests in CI, restore drill, SLO measurement |
| NFR-SEC, NFR-PRIV | permission and projection tests, header tests, scans, security checklist |
| NFR-USE | usability study rounds 1 and 2, axe in E |
| NFR-MAINT | CI coverage and lint gates |
| NFR-OBS | dashboard review in Sprint 4, log redaction test |
| NFR-PORT | deployment from a clean server in the restore drill |
| NFR-REPRO | determinism and leakage tests, run metadata checks |
| NFR-LEGAL | release checklist |

## 9. User story acceptance criteria verification matrix (US-01 to US-55)

| Story | Title | Priority | Test Level | Automated Test Evidence | Status / Verification Note |
|---|---|---|---|---|---|
| US-01 | Register with a campus email | M | U, I | `auth_test.go`, `auth_integration_test.go` | Proven (AC 1-5 verified) |
| US-02 | Sign in and out | M | U, I, E | `login-form.test.tsx`, `auth_integration_test.go`, `j1-registration.spec.ts` | Proven (AC 1-5 verified) |
| US-03 | Reset a forgotten password | M | I | `auth_integration_test.go` | Proven (AC 1-3 verified) |
| US-04 | Invite officers and administrators | M | I | `auth_integration_test.go`, `admin_integration_test.go` | Proven (AC 1-3 verified) |
| US-05 | Manage my profile and sessions | M | I | `auth_integration_test.go` | Proven (AC 1-3 verified) |
| US-06 | Seed and browse campus master data | M | U, I | `masterdata_test.go`, `masterdata_integration_test.go` | Proven (AC 1-3 verified) |
| US-07 | Edit master data | M | I | `admin_integration_test.go` | Proven (AC 1-3 verified) |
| US-08 | File a lost report | M | I, E | `reports_integration_test.go`, `j2-lost-report.spec.ts` | Proven (AC 1-6 verified) |
| US-09 | Follow my lost report | M | I | `reports_integration_test.go` | Proven (AC 1-2 verified) |
| US-10 | Edit or close my lost report | M | I | `reports_integration_test.go` | Proven (AC 1-3 verified) |
| US-11 | Keep or close a stale report, reopen it | M | I | `maintenance_integration_test.go` | Proven (AC 1-3 verified) |
| US-12 | File a lost report for a walk in visitor | M | I | `reports_integration_test.go` | Proven (AC 1 verified) |
| US-13 | File a found report | M | I, E | `reports_integration_test.go`, `j3-found-report.spec.ts` | Proven (AC 1-4 verified) |
| US-14 | See if an owner may exist | M | I | `matching_integration_test.go` | Proven (AC 1-2 verified) |
| US-15 | Be warned about a duplicate report | S | U, I | `use-assist.test.tsx`, `assist_integration_test.go` | Proven (AC 1-2 verified) |
| US-16 | Be reminded to hand in the item | M | I | `custody_integration_test.go` | Proven (AC 1-2 verified) |
| US-17 | Browse the public found listing | M | I | `reports_integration_test.go` | Proven (AC 1-2 verified) |
| US-18 | Embed and analyse every report | M | U, I, C | `ml/tests/test_embed.py`, `matching_integration_test.go` | Proven (AC 1-3 verified) |
| US-19 | Rank candidates for every report | M | U, I | `fusion_test.go`, `matching_integration_test.go` | Proven (AC 1-4 verified; real dataset gate PENDING HUMAN) |
| US-20 | Understand why a candidate is suggested | M | U | `score-bars.test.tsx`, `matching_test.go` | Proven (AC 1-4 verified) |
| US-21 | Give feedback or dismiss a candidate | M | I | `matching_integration_test.go` | Proven (AC 1-2 verified) |
| US-22 | Be told when a strong candidate appears later | M | I | `notify_integration_test.go`, `matching_integration_test.go` | Proven (AC 1-2 verified) |
| US-23 | Get suggestions from a photo | M | U, I, C | `ml/tests/test_analyze.py`, `assist_integration_test.go` | Proven (AC 1-4 verified) |
| US-24 | Get a drafted description | M | U | `assist_test.go` | Proven (AC 1-2 verified) |
| US-25 | Claim a candidate | M | I, E | `claims_integration_test.go`, `j4-claim.spec.ts` | Proven (AC 1-4 verified) |
| US-26 | Review a claim with the verification assistant | M | I | `claims_integration_test.go` | Proven (AC 1-3 verified) |
| US-27 | Ask for more information, approve or reject | M | I | `claims_integration_test.go` | Proven (AC 1-4 verified) |
| US-28 | Withdraw a claim | M | I | `claims_integration_test.go` | Proven (AC 1 verified) |
| US-29 | Expire stalled claims | M | I | `claims_integration_test.go`, `maintenance_integration_test.go` | Proven (AC 1-3 verified) |
| US-30 | Receive an item at a post | M | I, E | `custody_integration_test.go`, `j3-found-report.spec.ts` | Proven (AC 1-3 verified) |
| US-31 | Hand over with a code | M | I, E | `custody_integration_test.go`, `j5-handover.spec.ts` | Proven (AC 1-3 verified) |
| US-32 | Manage the post inventory | M | I | `custody_integration_test.go` | Proven (AC 1 verified) |
| US-33 | Dispose of unclaimed items | M | I, E | `custody_integration_test.go`, `j6-retention.spec.ts` | Proven (AC 1-4 verified) |
| US-34 | Transfer between posts | S | I | `custody_integration_test.go` | Proven (AC 1 verified) |
| US-35 | Receive notifications in the app | M | I | `notify_integration_test.go` | Proven (AC 1-3 verified) |
| US-36 | Receive email notifications | M | I | `notify_integration_test.go` | Proven (AC 1-3 verified) |
| US-37 | Search all reports | M | I | `search_integration_test.go` | Proven (AC 1-2 verified) |
| US-38 | Work from a priority queue | M | I | `officer_integration_test.go` | Proven (AC 1-2 verified) |
| US-39 | Review matches side by side | M | U, I | `review-board.test.tsx`, `officer_integration_test.go` | Proven (AC 1-3 verified) |
| US-40 | Merge duplicates | S | I | `officer_integration_test.go` | Proven (AC 1 verified) |
| US-41 | Search by meaning | S | U, I | `search_test.go`, `search_integration_test.go` | Proven (AC 1 verified) |
| US-42 | Moderate reports and users | M | I | `admin_integration_test.go` | Proven (AC 1-2 verified) |
| US-43 | Change runtime settings | M | U, I | `settings_test.go`, `admin_integration_test.go`, `admin-settings.test.tsx` | Proven (AC 1 verified) |
| US-44 | Browse the audit log | M | I | `admin_integration_test.go` | Proven (AC 1-2 verified) |
| US-45 | Delete my account | M | I | `privacy_integration_test.go` | Proven (AC 1-2 verified) |
| US-46 | See hotspots on a campus map | M | I | `analytics_integration_test.go` | Proven (AC 1-2 verified) |
| US-47 | See operational trends | M | U, I | `admin-analytics.test.tsx`, `analytics_integration_test.go` | Proven (AC 1-2 verified) |
| US-48 | Find clusters of related reports | S | U, I, C | `ml/tests/test_cluster.py`, `analytics_integration_test.go` | Proven (AC 1-2 verified) |
| US-49 | Monitor the matching engine | M | I, E | `analytics_integration_test.go`, `j7-fusion.spec.ts` | Proven (AC 1-2 verified) |
| US-50 | Scaffold the services and the local stack | M | I | `docker compose config`, `/healthz`, `/readyz` | Proven (AC 1-2 verified) |
| US-51 | Continuous integration | M | CI | `.github/workflows/ci-*.yml` | Proven (AC 1 verified in CI workflows) |
| US-52 | Observability | M | I, Ops | `/metrics`, request ID logging, `deploy/monitoring/` | Proven (AC 1-2 automated evidence verified) |
| US-53 | Deploy to staging and production | M | Ops | `.github/workflows/release.yml`, `deploy/scripts/deploy.sh` | Automated artifacts verified; server execution PENDING (HUMAN) |
| US-54 | Backups and a restore drill | M | Ops | `deploy/scripts/backup.sh`, `deploy/scripts/restore.sh`, `docs/operations/restore-drill.md` | Automated scripts verified; physical drill on server PENDING (HUMAN) |
| US-55 | Security hardening | M | U, I, Scan | `ci-*.yml` audits, `httpx`, `security-privacy.md` | Automated checks verified; ZAP baseline scan against staging PENDING (HUMAN) |

