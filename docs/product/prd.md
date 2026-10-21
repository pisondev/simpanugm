# Product Requirements | SimpanUGM

| | |
|---|---|
| Product | SimpanUGM, Sistem Informasi Manajemen Penemuan dan Pengembalian Barang UGM |
| Document owner | Pison Golda |
| Contributors | Irfan Fadilah Rafif (AI), Axelle Chandra (data) |
| Version | 1.0, 17 October 2026 |
| Target release | v1.0.0 on 16 December 2026 |

## 1. Summary

SimpanUGM is a lost and found platform for the Universitas Gadjah Mada campus. It replaces
scattered social media posts, chat groups and paper logbooks at security posts with one
system that records every lost and found item, ranks the most likely matches with an AI
engine that reads photos, descriptions, places and times, and routes every handover
through an officer who verifies ownership first.

The AI is decision support. It shortens the search from "scroll through everything" to
"look at these five", explains why each candidate is on the list, and helps the officer
check the right details. It never hands an item to anyone.

## 2. Problem

Losing something on a campus of sixty thousand people spread over four clusters and
dozens of buildings goes like this today:

1. The owner posts on a campus confession account or a faculty chat group and hopes the
   right person reads it within a day.
2. A finder who wants to help either posts a photo publicly, which invites false claims
   because anyone can now describe the item, or drops it at the nearest security post,
   where it goes into a logbook nobody outside that post can search.
3. Posts do not share records. An umbrella left at Perpustakaan Pusat and reported lost
   by someone who searched only at their faculty post is never connected.
4. Officers verify ownership by intuition. There is no record of what was checked, who
   received the item, or when.
5. Unclaimed items pile up, and nobody can say which locations or categories produce the
   most losses, so prevention never happens.

The result is low return rates, exposure to fraudulent claims, lost accountability for
officers, and no data for the security unit.

## 3. Vision and principles

**Vision.** Anything lost on campus finds its way back to its owner within days, through
a process that is fast for students and accountable for officers.

| Principle | What it means for design |
|---|---|
| People decide, AI ranks | Every ownership decision and every handover is a recorded human action. AI output is shown as a score with its parts, never as a verdict |
| Private by default | Found item photos are never published as a gallery. Owners see only candidates matched to their own report. Identifying details stay hidden from everyone except officers |
| Three minutes or less | Reporting must be faster than writing a social media post, or nobody uses it. Photos fill the form, defaults are sensible, the flow works one handed on a phone |
| Every step leaves a trace | Status changes, verifications, custody moves and handovers write an audit entry with actor and time |
| Works on a bad connection | Mobile first, images compressed on the device, drafts survive a dropped upload |

## 4. Goals and non goals

### Goals for v1.0

| Id | Goal | Measured by |
|---|---|---|
| G1 | Owners find the right candidate quickly | Recall@5 of the matching engine, time from report to first candidate |
| G2 | Reporting is easy enough to beat social media | Median completion time, task completion rate, SUS score |
| G3 | Handovers are verified and accountable | Share of handovers with an approved claim, a valid code and an officer record |
| G4 | Officers see what needs action first | Priority queue usage, median time from claim to decision |
| G5 | The security unit gets usable data | Hotspot and trend reports available, monthly export |
| G6 | The system is production ready | Section 9 checklist complete |

### Non goals

- Not a marketplace or a public bulletin board of found item photos.
- Not a chat app between owners and finders. Contact goes through officers.
- No native mobile app. The responsive web app is installable as a PWA in v1.1.
- No integration with university identity systems in v1.0. Registration uses the
  university email domain. Google sign in limited to UGM domains comes in v1.1.
- No hardware: no lockers, no RFID, no CCTV analysis.
- No large language model in the request path. Descriptions are drafted from detected
  attributes with templates, so output is deterministic and no campus data leaves the
  server.

## 5. Users

The concept defines four roles. In the system, owner and finder are the same account type
(`member`) acting in a different relation to a report, because the same student loses an
umbrella one week and finds a charger the next.

| Role | System role | Who | Volume |
|---|---|---|---|
| Owner, pelapor barang hilang | `member` | Students, lecturers, staff with a UGM email | Thousands of accounts, tens of reports a day at peak |
| Finder, penemu barang | `member` | Same population | Same |
| Officer, petugas keamanan or Lost and Found | `officer` | Security staff assigned to one or more posts | Tens of accounts |
| System administrator | `admin` | Staff of the campus security unit and the team during the pilot | Two to five accounts |

Visitors without a UGM email are served through officers: an officer files a proxy
report on their behalf and is the contact for it.

### Personas

**Rani, owner.** Second year student. Lost her wallet with KTM, an ATM card and cash
somewhere between the central library and the canteen. Has no photo of the wallet. Is on
her phone, anxious, and wants to know within the hour whether anyone found it.
*Needs:* a fast form that works without a photo, an immediate answer on whether there are
candidates, a clear next step.

**Dimas, finder.** Engineering student. Found wireless earbuds in a lecture room. Wants to
help but has a class in ten minutes. *Needs:* snap a photo, have the form filled for him,
know which post to drop the item at, get a thank you when it reaches the owner.

**Pak Slamet, officer.** Runs the Lost and Found desk at a cluster security post on a
shared desktop and a phone. Moderate digital literacy. Receives three to ten items a day.
*Needs:* one screen that tells him what to do next, a receive item flow that takes under a
minute, a checklist when someone claims an item, a record that protects him if a
handover is disputed.

**Mbak Wulan, administrator.** Works at the campus security unit. Manages officer
accounts and the list of buildings and posts. Reports monthly to leadership. *Needs:*
reliable numbers by location and category, an audit trail, assurance that the AI is
working and not drifting.

## 6. Core journeys

Full flows with screens are in [information-architecture.md](../ux/information-architecture.md).

1. **Report lost.** Owner opens Lapor Kehilangan, optionally adds photos, accepts or edits
   the suggested category, colour and brand, describes the item, picks up to three places
   and a time window, adds private identifying details, submits. Candidates appear on the
   report page within a minute, and a notification arrives when a strong match shows up
   later.
2. **Report found.** Finder takes a photo, the form fills itself, finder confirms place
   and time, is warned if the same item seems to be already reported, chooses to hand the
   item to a post now or within 48 hours, submits. The finder sees whether a likely owner
   exists, without seeing who.
3. **Receive at post.** Officer scans or types the found report code, confirms the item,
   records the storage slot. Retention starts.
4. **Claim and verify.** Owner opens a candidate, reads the score breakdown, submits a
   claim with answers to category specific questions. Officer reviews lost and found side
   by side, works through the verification checklist, approves, rejects or asks for more.
5. **Handover.** Approved owner comes to the post with KTM or KTP and the six digit
   handover code. Officer enters the code and the last four digits of the ID. Both reports
   move to returned, the finder gets a thank you, competing claims are closed.
6. **Unclaimed.** Items not claimed within the retention period surface in the officer
   inventory for a recorded disposition.
7. **Analyse.** Administrator opens the hotspot map, the trends, the clusters and the AI
   monitoring page.

## 7. Scope and releases

Priority uses MoSCoW. Must is required for v1.0, Should is planned for v1.0 and may slip
to v1.1, Could is v1.1 or later.

| Feature | Concept document source | Priority | Release |
|---|---|---|---|
| Email registration limited to UGM domains, verification, login, logout, password reset | Authentication and RBAC module | Must | v0.1 |
| Roles `member`, `officer`, `admin`, officer to post assignment | User roles and permissions | Must | v0.1 |
| Master data: categories, zones, locations, posts | Administration module | Must | v0.1 |
| Lost report create, edit, close, reopen, history | Lost item reporting | Must | v0.1 |
| Found report create, edit, custody choice | Found item reporting | Must | v0.1 |
| Photo upload with compression, EXIF stripping, thumbnails, access control | Media and evidence module | Must | v0.1 |
| Officer search by name, category, location, date, status, case id | Search and operational dashboard | Must | v0.1 |
| Custody intake at a post, storage slot, retention clock | Claim and verification module | Must | v0.1 |
| Proxy reports by officers for walk in visitors | Officer role | Must | v0.1 |
| Matching engine: image, text, cross modal, location, time, category | AI matching engine, AI-01 | Must | v0.2 |
| Ranked candidates and explainable match view | Explainable match view, AI-02 | Must | v0.2 |
| Smart report assistant: category, colour, visible text, brand | AI assisted item description, AI-03, AI-04 | Must | v0.2 |
| Instant match hints for finders | Instant match suggestions, AI-05 | Must | v0.2 |
| In app notifications with live updates, email notifications | Notification module | Must | v0.2 |
| Claims with category specific questions and evidence | Claim and ownership verification | Must | v0.3 |
| Claim review, request info, approve, reject, message thread | Claim and verification module | Must | v0.3 |
| Handover by code with ID check, competing claim resolution | Claim and verification module | Must | v0.3 |
| Officer review board, side by side | AI match review board, AI-07 | Must | v0.3 |
| Verification assistant checklist | Verification assistant, AI-08 | Must | v0.3 |
| AI priority queue | AI priority queue, AI-09 | Must | v0.3 |
| Duplicate and related report detection | Duplicate detection, AI-06 | Should | v0.3 |
| Retention jobs: stale lost reports, unclaimed items, auto close | Case status tracking | Must | v0.3 |
| Audit log viewer | Database and audit module | Must | v1.0 |
| Moderation: hide and restore reports, suspend accounts | Administration module | Must | v1.0 |
| Hotspot analytics with campus map | Hotspot analytics, AI-10 | Must | v1.0 |
| AI and matching monitoring | AI and matching monitoring, AI-12 | Must | v1.0 |
| AI case clustering | AI case clustering, AI-11 | Should | v1.0 |
| Semantic search for officers | Search and operational dashboard, AI-13 | Should | v1.0 |
| Learned score fusion, trained on the evaluation set | Context aware match scoring | Should | v1.0 |
| Printable QR label for stored items | Officer workflow | Could | v1.1 |
| Google sign in limited to UGM domains | Authentication | Could | v1.1 |
| Web push notifications, installable PWA | Cross platform dashboard | Could | v1.1 |
| TOTP second factor for officers and admins | Security | Could | v1.1 |
| English UI | Cross platform dashboard | Could | later |

### Release milestones

| Version | Date | Theme |
|---|---|---|
| v0.1.0 | Fri 6 Nov 2026 | Foundation: accounts, master data, reports, photos, custody, search |
| v0.2.0 | Fri 20 Nov 2026 | AI matching, explanations, report assistant, notifications |
| v0.3.0 | Fri 4 Dec 2026 | Claims, verification, handover, officer tools, retention |
| v1.0.0 | Wed 16 Dec 2026 | Analytics, monitoring, hardening, production deployment, evaluation report |

The final demonstration is in class on Thursday 17 December 2026.

## 8. Success metrics

### AI, offline on the held out test split

Protocol in [evaluation.md](../ai/evaluation.md).

| Metric | Target |
|---|---|
| Recall@5, lost to found, all reports | at least 0.80 |
| Recall@1 | at least 0.55 |
| Mean reciprocal rank | at least 0.65 |
| Precision of the high tier | at least 0.85 |
| Recall@10 when the lost report has no photo (cross modal only) | at least 0.60 |
| Category suggestion top 3 accuracy | at least 0.85 |
| Match job latency p95, CPU only | at most 60 s from submit to stored candidates |

### Product, usability test with at least ten participants

| Metric | Target |
|---|---|
| Lost report task completion without help | at least 90 percent |
| Median time to submit a lost report with one photo | at most 3 minutes |
| Median time to submit a found report | at most 2 minutes |
| System Usability Scale | at least 75 |
| Category suggestion accepted without edit | at least 70 percent |

### Operations, simulated pilot

| Metric | Target |
|---|---|
| Handovers with an approved claim, a valid code and an officer record | 100 percent |
| Median officer time from opening a claim to decision | at most 5 minutes |
| Median officer time to receive an item into custody | at most 1 minute |

### Engineering

| Metric | Target |
|---|---|
| API read latency p95 at 100 concurrent users | at most 300 ms |
| Availability of production during the pilot | at least 99.5 percent |
| CI duration on a pull request | at most 10 minutes |
| Critical or high vulnerabilities open at release | zero |

## 9. What production ready means here

v1.0.0 ships only when every line below holds. Each one is backed by a requirement in the
SRS.

- [ ] Runs from `docker compose up` on a clean server, configured only by environment
      variables, with HTTPS.
- [ ] Database migrations run forward and backward in CI. Seeds load master data.
- [ ] Every service exposes liveness and readiness checks and shuts down gracefully.
- [ ] Structured JSON logs with request ids across `web`, `api` and `ml`.
- [ ] Prometheus metrics and a Grafana dashboard for traffic, latency, errors, queue depth
      and matching quality.
- [ ] Nightly database and media backups, and one restore drill completed and recorded.
- [ ] Authentication, authorization, CSRF protection, rate limiting and upload validation
      covered by tests.
- [ ] Personal data handling follows [security-privacy.md](../architecture/security-privacy.md),
      including retention jobs.
- [ ] Model versions are recorded with every match and can be rolled back without a
      deploy.
- [ ] Offline evaluation report with the metrics in section 8, reproducible from the
      repository with fixed seeds.
- [ ] CI runs lint, type checks, unit, integration and end to end tests, and security
      scans. Images are built and published on tag.
- [ ] Runbook covers the incidents in [runbook.md](../operations/runbook.md).

## 10. Constraints

| Constraint | Consequence |
|---|---|
| Stack is Next.js, PostgreSQL and Go | The core is Go. Python is admitted only for inference and evaluation, see ADR-0003 |
| One CPU only server, no GPU | Models are chosen for CPU inference: ViT-B/32 class image encoders, small text encoders, see matching-engine |
| Three people, nine weeks, alongside six other courses | Modular monolith, one database, no microservices, see ADR-0002 |
| Personal data of students | UU No. 27 Tahun 2022 tentang Pelindungan Data Pribadi applies, see security-privacy |
| No real campus data before launch | Evaluation uses a dataset the team collects with consent, see dataset-collection |

## 11. Dependencies

| Dependency | Needed for | Fallback |
|---|---|---|
| A VPS with 4 vCPU and 8 GB RAM | staging and production | a team laptop running the compose stack for the demo |
| A domain with DNS control | HTTPS and email links | Caddy internal certificates on a LAN demo |
| An SMTP account | email notifications | in app notifications only, email queued |
| Pretrained model weights from Hugging Face | matching, assistant | weights cached in a volume after first download, mirrored in the team drive |
| Mapbox Streets tiles and reviewed building coordinates | campus map, coordinates | static map image of campus with point markers |

## 12. Related pages

- Requirements: [srs.md](../requirements/srs.md)
- Stories: [user-stories.md](user-stories.md)
- Plan: [project-plan.md](../project/project-plan.md)
- Risks: [risks.md](../project/risks.md)
