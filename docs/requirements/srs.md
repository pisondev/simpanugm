# Software Requirements Specification | SimpanUGM

| | |
|---|---|
| Version | 1.0, 17 October 2026 |
| Owner | Pison Golda |
| Structure | Adapted from ISO/IEC/IEEE 29148 |
| Status | Baseline for implementation. Changes go through a pull request that updates this page |

## 1. Introduction

### 1.1 Purpose

This page states what SimpanUGM must do and how well. It is the reference for building,
reviewing and testing. Each requirement has an id that stories, pull requests and tests
cite.

### 1.2 Scope

SimpanUGM records lost and found items on the UGM campus, matches them with an AI engine,
and manages claims, verification, custody and handover through officers. The product
context, personas and release plan are in [prd.md](../product/prd.md).

### 1.3 Reading a requirement

| Column | Meaning |
|---|---|
| Id | `FR-<MODULE>-nn` or `NFR-<QUALITY>-nn`, never reused once published |
| Requirement | One testable statement. "Shall" is mandatory behaviour |
| P | Priority: **M** Must, **S** Should, **C** Could |
| Rel | First release that contains it: 0.1, 0.2, 0.3, 1.0, 1.1 |

Numbers that the system enforces are defined once in
[business-rules.md](business-rules.md) and referenced as `BR-...`. Lifecycles are in
[state-machines.md](state-machines.md). Permissions are in [permissions.md](permissions.md).

### 1.4 Definitions

Terms such as lost report, found report, match, candidate, claim, custody, post, tier and
handover code are defined in the [glossary](../glossary.md).

## 2. Overall description

### 2.1 Product perspective

A new, self contained web system. It has no upstream system to integrate with in v1.0.
Three runtime services and two stateful stores:

| Part | Responsibility |
|---|---|
| `web` | Next.js application serving every page for all roles |
| `api` | Go service owning the domain, authorization, persistence, jobs, notifications |
| `ml` | Python service computing embeddings, image attributes and clusters. Stateless, internal only |
| PostgreSQL | All relational data, vectors through pgvector, the job queue |
| Media storage | Photos and evidence, on the filesystem or an S3 compatible bucket |

Detail: [overview.md](../architecture/overview.md).

### 2.2 Product functions

1. Accounts, roles and sessions.
2. Master data for categories, zones, locations and posts.
3. Lost and found reporting with photos.
4. AI matching with explanations, report assistance and duplicate detection.
5. Claims, verification, custody and handover.
6. Notifications in app and by email.
7. Search and the officer workspace.
8. Administration, moderation and audit.
9. Analytics and AI monitoring.

### 2.3 User classes

| Class | System role | Characteristics | Access frequency |
|---|---|---|---|
| Owner | `member` | Any student or staff member, mobile, often stressed, may have no photo | Occasional, intense for a few days |
| Finder | `member` | Same population, time poor, wants minimal effort | Occasional |
| Officer | `officer` | Security staff at a post, shared desktop and phone, moderate digital literacy | Daily, most of a shift |
| Administrator | `admin` | Security unit staff, comfortable with dashboards | Weekly |

### 2.4 Operating environment

| Aspect | Requirement |
|---|---|
| Server | Linux x86_64, Docker Engine 27 or newer with Compose v2, 4 vCPU, 8 GB RAM, 80 GB SSD, no GPU |
| Clients | Current and previous major version of Chrome, Edge, Firefox, Safari on iOS 16 and newer, Chrome on Android 10 and newer |
| Network | Works on a 4G connection of 1.5 Mbps down. Campus Wi-Fi is not assumed to be stable |
| Time | Stored in UTC. Displayed in Asia/Jakarta (WIB, UTC+7) |
| Language | Indonesian user interface |

### 2.5 Design and implementation constraints

| Id | Constraint |
|---|---|
| CON-01 | Frontend in Next.js with TypeScript |
| CON-02 | Core backend in Go |
| CON-03 | Relational store is PostgreSQL. Vectors live in the same database through pgvector |
| CON-04 | Python is allowed only inside `ml/` for inference and offline evaluation |
| CON-05 | No external AI API in the request path. All inference runs on the team server |
| CON-06 | CPU only inference |
| CON-07 | Datasets, model weights and user media never enter the git repository |
| CON-08 | All configuration through environment variables |

### 2.6 Dependencies

Pretrained weights from Hugging Face, an SMTP account, a VPS, a domain, OpenStreetMap
data for building coordinates. Fallbacks are in [prd.md](../product/prd.md) section 11.

## 3. Functional requirements

### 3.1 Authentication and accounts, `FR-AUTH`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-AUTH-01 | The system shall let a person register with full name, email, password, and optionally NIM or NIP and a phone number. The email domain must be in the allowed list (BR-ACC-01) | M | 0.1 |
| FR-AUTH-02 | The system shall reject a password shorter than 10 characters, longer than 128, equal to the email, or present in the bundled list of the 10,000 most common passwords | M | 0.1 |
| FR-AUTH-03 | The system shall send a verification email with a single use link valid for 24 hours. An unverified account can sign in and browse but cannot create reports or claims (BR-ACC-02) | M | 0.1 |
| FR-AUTH-04 | The system shall sign a user in with email and password and answer every failure with the same message, "Email atau kata sandi salah", regardless of the cause | M | 0.1 |
| FR-AUTH-05 | The system shall create a server side session on sign in and send its opaque token only in an `HttpOnly`, `Secure`, `SameSite=Lax` cookie. The database stores only a SHA-256 hash of the token | M | 0.1 |
| FR-AUTH-06 | The system shall apply the session lifetimes of BR-ACC-04 per role | M | 0.1 |
| FR-AUTH-07 | The system shall sign out the current session, and on request every session of the user | M | 0.1 |
| FR-AUTH-08 | The system shall reset a password through an emailed single use link valid for 30 minutes, and revoke every session of the user after a successful reset | M | 0.1 |
| FR-AUTH-09 | The system shall let an administrator invite an officer or administrator by email. The invitation link is valid for 72 hours and leads to a set password page | M | 0.1 |
| FR-AUTH-10 | The system shall let a user view and edit name, phone and NIM or NIP. Changing the email requires verifying the new address before it takes effect | M | 0.1 |
| FR-AUTH-11 | The system shall list the user's active sessions with device, approximate location from IP, created and last seen time, and let the user revoke any of them. Version 0.2 shows the network prefix of the address, a city lookup needs a GeoIP database and follows in 1.1 | S | 0.1 |
| FR-AUTH-12 | The system shall deny sign in to a suspended account and revoke its sessions at the moment of suspension | M | 0.1 |
| FR-AUTH-13 | The system shall let a user request deletion of their account. Deletion anonymises personal data per BR-RET-05 and keeps non personal case records for statistics | M | 1.0 |
| FR-AUTH-14 | The system shall require acceptance of the privacy notice at registration and record the version accepted and the time | M | 0.1 |
| FR-AUTH-15 | The system shall offer Google sign in restricted to the allowed hosted domains | C | 1.1 |
| FR-AUTH-16 | The system shall offer TOTP as a second factor and require it for officers and administrators once enabled by an administrator | C | 1.1 |

### 3.2 Master data, `FR-MD`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-MD-01 | The system shall maintain categories in two levels, parent and leaf. Each leaf carries a sensitivity class (`normal`, `valuable`, `identity`), a retention period, an ordered list of verification questions, a list of English prompts for zero shot classification, and an active flag | M | 0.1 |
| FR-MD-02 | The system shall maintain compatibility pairs between leaf categories with a factor between 0 and 1, used by matching | M | 0.2 |
| FR-MD-03 | The system shall maintain zones, the campus clusters, each with a name and a centroid coordinate | M | 0.1 |
| FR-MD-04 | The system shall maintain locations with name, aliases, zone, kind (`building`, `outdoor`, `facility`, `transport`, `parking`), coordinate and active flag | M | 0.1 |
| FR-MD-05 | The location picker shall search names and aliases case insensitively and without diacritics, so "perpus" finds Perpustakaan Pusat | M | 0.1 |
| FR-MD-06 | The system shall maintain posts with name, location, phone, opening hours and active flag, and an assignment of officers to one or more posts | M | 0.1 |
| FR-MD-07 | Deactivating a master data row shall remove it from pickers and keep it on existing records | M | 0.1 |
| FR-MD-08 | Master data endpoints shall support conditional requests with `ETag` so the client can cache them | S | 0.1 |
| FR-MD-09 | The system shall load the initial master data from versioned seed files in `data/seed/` with an idempotent seed command | M | 0.1 |

### 3.3 Lost reports, `FR-LOST`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-LOST-01 | The system shall let a verified member create a lost report with: category (required), item name (required, 3 to 80 characters), description (required, 20 to 1000), time window start and end (required, BR-LST-01), one to three locations or the flag "tidak tahu lokasinya", location note (optional, up to 200), private identifying details (optional, up to 500), up to two colours from the palette, brand (optional, up to 40), zero to five photos | M | 0.1 |
| FR-LOST-02 | The system shall assign a human readable code `LST-YYMM-NNNN` unique per month, in addition to the internal UUID | M | 0.1 |
| FR-LOST-03 | Creation shall be idempotent on the `Idempotency-Key` header for 24 hours, so a double tap or a retry on a bad connection creates one report | M | 0.1 |
| FR-LOST-04 | The web client shall keep an unsent draft of the form on the device and restore it after a reload or a failed submission | M | 0.1 |
| FR-LOST-05 | On creation, and on any edit of category, name, description, details, colour, brand, locations, time window or photos, the system shall enqueue embedding and matching for the report | M | 0.2 |
| FR-LOST-06 | The system shall warn a member who submits a lost report similar to one of their own open reports (BR-LST-05) and let them open the existing one instead | S | 0.2 |
| FR-LOST-07 | The owner shall see a list of their lost reports with code, item, status, candidate count and last update, filterable by status | M | 0.1 |
| FR-LOST-08 | The owner shall see the report detail: fields, photos, status timeline, candidates with tier and score breakdown, and claims | M | 0.1 |
| FR-LOST-09 | The owner may edit the report while its status is `reported` or `potential_match`. Each edit is recorded with the changed fields | M | 0.1 |
| FR-LOST-10 | The owner may close an open report with reason `found_by_owner` or `no_longer_needed` | M | 0.1 |
| FR-LOST-11 | The owner may reopen a report within the window of BR-LST-04 if it was closed as `expired`, `found_by_owner` or `no_longer_needed` | M | 0.3 |
| FR-LOST-12 | The system shall send a stale reminder and auto close inactive reports per BR-LST-02 and BR-LST-03 | M | 0.3 |
| FR-LOST-13 | An officer shall be able to create a lost report on behalf of a walk in visitor, recording visitor name and phone. The officer becomes the contact and receives its notifications | M | 0.1 |
| FR-LOST-14 | After handover the owner shall be asked to confirm receipt. Confirmation closes the report as `returned` immediately, otherwise BR-LST-06 applies | M | 0.3 |

### 3.4 Found reports, `FR-FOUND`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-FOUND-01 | The system shall let a verified member create a found report with: category (required), item name (required, 3 to 80), public description (required, 10 to 500), hidden details (optional, up to 500), found time (required, BR-FND-01), found location (required, exactly one) with note, one to five photos (required), and a custody choice | M | 0.1 |
| FR-FOUND-02 | The custody choice shall be either "Serahkan ke pos" with a chosen post, or "Saya simpan sementara". The second option is offered only for categories of sensitivity `normal` (BR-FND-02) | M | 0.1 |
| FR-FOUND-03 | The system shall assign a code `FND-YYMM-NNNN` | M | 0.1 |
| FR-FOUND-04 | Creation shall be idempotent on `Idempotency-Key` as in FR-LOST-03, and the client shall keep a draft as in FR-LOST-04 | M | 0.1 |
| FR-FOUND-05 | Before submission the system shall run a duplicate check (AI-06) and show up to three similar open found reports. The finder may continue or open the existing report | S | 0.3 |
| FR-FOUND-06 | After submission the finder shall see the instant match hint (AI-05): whether likely owners exist and the best tier, never the owner identity or the lost report details | M | 0.2 |
| FR-FOUND-07 | The finder shall see a list and detail of their found reports with status, custody status and timeline | M | 0.1 |
| FR-FOUND-08 | The finder may edit the report while custody is `with_finder` and no claim exists. Officers may edit it until it is returned | M | 0.1 |
| FR-FOUND-09 | The finder may close a report only while custody is `with_finder` and no claim is open, with reason `returned_directly` or `reported_in_error` and a note | M | 0.1 |
| FR-FOUND-10 | The system shall remind the finder and alert officers per BR-FND-03 while an item stays with the finder | M | 0.3 |
| FR-FOUND-11 | An officer shall be able to create a found report for a walk in finder. Custody starts as `in_custody` at the officer's post and the finder name is optional | M | 0.1 |
| FR-FOUND-12 | The system shall provide a public listing of open found reports that shows only category, item name, zone, found date and holding post, never photos, descriptions, hidden details or the finder | M | 0.1 |
| FR-FOUND-13 | The finder shall be notified when the item is received at a post and when it is returned to its owner | M | 0.2 |

### 3.5 Media and evidence, `FR-MEDIA`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-MEDIA-01 | The system shall accept JPEG, PNG and WebP, identified by content signature and not by file extension. The web client converts HEIC and HEIF to JPEG before upload | M | 0.1 |
| FR-MEDIA-02 | The system shall enforce the size and count limits of BR-MED-01 | M | 0.1 |
| FR-MEDIA-03 | The web client shall downscale images to at most 2048 px on the long side at quality 0.85 before upload | M | 0.1 |
| FR-MEDIA-04 | The server shall decode every image, apply its orientation, drop all metadata including EXIF and GPS, and write three variants: `original` (JPEG, long side at most 2560 px), `display` (WebP, 1280 px), `thumb` (WebP, 320 px). For `identity` categories it also writes `thumb_blur` | M | 0.1 |
| FR-MEDIA-05 | The system shall store SHA-256 and a 64 bit perceptual hash of each image, reject an exact duplicate within the same report, and pass perceptual duplicates across reports to AI-06 | M | 0.1 |
| FR-MEDIA-06 | Uploads are staged and must be attached to a report or claim within 24 hours, after which they are deleted | M | 0.1 |
| FR-MEDIA-07 | Media shall be served only through an authorised endpoint with `Cache-Control: private`. No storage object is reachable by a public URL | M | 0.1 |
| FR-MEDIA-08 | Which variant a user may see is decided by BR-VIS rules | M | 0.1 |
| FR-MEDIA-09 | The uploader may remove a photo while the report is editable. A found report keeps at least one photo | M | 0.1 |

### 3.6 Matching engine, `FR-AI`

The algorithm, its numbers and its failure modes are specified in
[matching-engine.md](../ai/matching-engine.md). These requirements state the observable
behaviour.

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-AI-01 | The system shall compute an image embedding for every `display` image attached to a report | M | 0.2 |
| FR-AI-02 | The system shall compute two text embeddings for every report, a multilingual sentence embedding and a multilingual CLIP text embedding, from the matching text defined in matching-engine section 3 | M | 0.2 |
| FR-AI-03 | The system shall extract image attributes for every attached image: category suggestions, dominant colours, visible text and brand candidates | M | 0.2 |
| FR-AI-04 | For every new or changed report the system shall retrieve candidate reports of the opposite type and score them with the visual, text, location, time and category components | M | 0.2 |
| FR-AI-05 | The system shall keep for each report at most ten active candidates whose overall score is at least the storage floor, each with its component scores, visual mode, tier, rank and the ids of the model versions used | M | 0.2 |
| FR-AI-06 | The system shall set and clear `potential_match` on reports as defined in state-machines | M | 0.2 |
| FR-AI-07 | Matching failure shall never block report creation. Jobs retry with exponential backoff up to five attempts, then are marked failed and shown in AI monitoring | M | 0.2 |
| FR-AI-08 | The system shall rescore all open reports nightly at 02:00 WIB and after a model version is activated | M | 0.2 |
| FR-AI-09 | Owners and officers shall be able to mark a candidate relevant or not relevant with an optional reason. The feedback is stored as a label | M | 0.2 |
| FR-AI-10 | An owner dismissing a candidate hides it from that owner. An officer dismissing a candidate hides it from everyone | M | 0.2 |
| FR-AI-11 | The system shall offer an officer an on demand rematch of a single report | S | 0.3 |
| FR-AI-12 | Every displayed score shall be traceable to the stored component scores and model versions that produced it | M | 0.2 |

### 3.7 Report assistant, `FR-ASSIST`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-ASSIST-01 | When a photo is added to a report form, the system shall return within the time of NFR-PERF-04 up to three category suggestions with confidence, up to two colour names, visible text and brand candidates | M | 0.2 |
| FR-ASSIST-02 | Suggestions shall appear as editable defaults that the user confirms. The system shall record for each field whether the suggestion was accepted, edited or ignored | M | 0.2 |
| FR-ASSIST-03 | For found reports the system shall draft a public description from the accepted attributes with a template, which the finder can edit | M | 0.2 |
| FR-ASSIST-04 | If the inference service is unavailable or slow, the form shall continue without suggestions and show only the note "Saran otomatis sedang tidak tersedia" | M | 0.2 |
| FR-ASSIST-05 | Visible text shall be masked per BR-PRV-03 before it is stored or returned | M | 0.2 |

### 3.8 Claims, `FR-CLAIM`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-CLAIM-01 | A verified member shall be able to claim a found report from a candidate of their own lost report, or from the public listing. In the second case the system creates a lost report from the claim details and links both | M | 0.3 |
| FR-CLAIM-02 | The claim form shall require a statement (30 to 1000 characters) and answers to the verification questions of the found item's category, and accept up to three evidence images and a preferred visit window | M | 0.3 |
| FR-CLAIM-03 | The system shall enforce the claim limits of BR-CLM-01 to BR-CLM-04 | M | 0.3 |
| FR-CLAIM-04 | The system shall assign a code `CLM-YYMM-NNNN` | M | 0.3 |
| FR-CLAIM-05 | An officer shall be able to request more information with a message. The claimant answers with a message and optional evidence, which returns the claim to review | M | 0.3 |
| FR-CLAIM-06 | An officer shall be able to approve a claim only when the found item is `in_custody` and every required checklist item is resolved (FR-CLAIM-11). Approval generates the handover code (BR-HND-01) and puts other open claims on the same item on hold | M | 0.3 |
| FR-CLAIM-07 | An officer shall be able to reject a claim with a reason from the fixed list and a note. The claimant is notified and the underlying candidate is labelled negative | M | 0.3 |
| FR-CLAIM-08 | The claimant may withdraw a claim at any time before handover | M | 0.3 |
| FR-CLAIM-09 | Each claim shall have a message thread between claimant and officers. Finders are not part of it | M | 0.3 |
| FR-CLAIM-10 | The system shall expire claims per BR-CLM-05 and BR-CLM-06 and release claims on hold | M | 0.3 |
| FR-CLAIM-11 | Each claim shall carry a verification checklist generated by AI-08. Officers record pass, fail or not applicable with a note per item, and the results are kept with the decision | M | 0.3 |
| FR-CLAIM-12 | The claimant shall see the handover code in the app only, never by email, together with the post address, opening hours and what to bring | M | 0.3 |

### 3.9 Custody and handover, `FR-CUST`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-CUST-01 | An officer shall receive an item into custody by finding the report by code, QR or search, confirming it against the photos, and recording post, storage slot (free text up to 40, for example "Rak B-3") and condition note. The retention due date is computed per BR-CUS-01 | M | 0.1 |
| FR-CUST-02 | An officer shall be able to transfer an item between posts. Each transfer records source, destination, officer and time | S | 0.3 |
| FR-CUST-03 | An officer shall complete a handover by entering the claimant's handover code, the type of ID shown (`ktm`, `ktp`, `sim`, `passport`, `other`) and its last four characters. The code is checked per BR-HND-02 | M | 0.3 |
| FR-CUST-04 | A successful handover shall, in one transaction: set the claim to `handed_over`, the found report and lost report to `returned`, custody to `released`, reject the other claims on the item, write audit entries, and notify the claimant, the finder and the officers of the post | M | 0.3 |
| FR-CUST-05 | Each post shall have an inventory view of items in custody with age, retention due date, sensitivity, claim state and storage slot, sortable by due date | M | 0.1 |
| FR-CUST-06 | When retention ends without an open claim, the found report shall become `unclaimed` and the post officers notified (BR-CUS-02). An officer then records a disposition (`donated`, `destroyed`, `transferred_to_unit`, `returned_to_finder`) with a note, which closes the report | M | 0.3 |
| FR-CUST-07 | An officer may extend retention once per item by up to 30 days with a reason | S | 0.3 |
| FR-CUST-08 | The system shall render a printable label with the found code and a QR code for a stored item | C | 1.1 |

### 3.10 Notifications, `FR-NOTIF`

The full event catalog is [notifications.md](notifications.md).

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-NOTIF-01 | The system shall store in app notifications per user, list them newest first with an unread count, and mark one or all as read | M | 0.2 |
| FR-NOTIF-02 | The system shall push new notifications to open sessions over Server Sent Events and resume from `Last-Event-ID` after a reconnect | M | 0.2 |
| FR-NOTIF-03 | The system shall send email for the events that the catalog marks as email, through a queued job with retries, using Indonesian templates | M | 0.2 |
| FR-NOTIF-04 | The system shall respect per user preferences for optional events, and always send mandatory account and security emails | M | 0.2 |
| FR-NOTIF-05 | The system shall throttle match notifications per BR-NTF-01 | M | 0.2 |
| FR-NOTIF-06 | Every notification shall deep link to the page where the user acts on it | M | 0.2 |
| FR-NOTIF-07 | The system shall offer Web Push for installed clients | C | 1.1 |

### 3.11 Search, `FR-SEARCH`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-SEARCH-01 | Officers and administrators shall search reports by free text over code, item name, description, reporter name and email, with filters for type, category, zone, location, date range, status, custody status and post, sortable and paginated | M | 0.1 |
| FR-SEARCH-02 | Name and item search shall tolerate typos through trigram similarity | M | 0.1 |
| FR-SEARCH-03 | A code typed exactly, such as `FND-2611-0042`, shall open that record directly | M | 0.1 |
| FR-SEARCH-04 | Officers shall be able to switch to semantic search (AI-13), which ranks reports by meaning and still applies the filters | S | 1.0 |
| FR-SEARCH-05 | Members shall search only their own reports and the public found listing, filtered by category, zone and date | M | 0.1 |
| FR-SEARCH-06 | Search results shall never contain fields the requester may not see under BR-VIS | M | 0.1 |

### 3.12 Officer workspace, `FR-OFF`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-OFF-01 | The officer home shall show counters for the officer's posts: found today, in custody, claims waiting, retention due within seven days, high tier matches without a claim | M | 0.3 |
| FR-OFF-02 | The officer home shall show the AI priority queue (AI-09) with the reason for each item's position | M | 0.3 |
| FR-OFF-03 | The review board (AI-07) shall show a selected report next to its ranked candidates with full photos, private details, component scores and the actions: pin as strong, dismiss, invite the owner to claim | M | 0.3 |
| FR-OFF-04 | Inviting the owner to claim shall notify the owner that an officer considers a candidate strong | M | 0.3 |
| FR-OFF-05 | An officer shall be able to mark two reports of the same type as duplicates. One remains, the other closes with reason `duplicate` and a link to the survivor, and its photos and candidates are folded into the survivor | S | 0.3 |
| FR-OFF-06 | By default an officer sees queues for their own posts and can switch to the whole campus view | M | 0.3 |

### 3.13 Administration, `FR-ADMIN`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-ADMIN-01 | Administrators shall list and search users, invite officers and administrators, change roles, assign posts, suspend and reactivate accounts, revoke sessions and delete accounts | M | 0.1 |
| FR-ADMIN-02 | Administrators shall create, edit and deactivate categories, compatibility pairs, zones, locations and posts | M | 0.1 |
| FR-ADMIN-03 | Administrators shall hide a report with a reason and restore it. A hidden report is visible only to administrators and is excluded from matching and listings | M | 1.0 |
| FR-ADMIN-04 | Administrators shall edit the system settings listed in business-rules section 10. Each change is audited with old and new value | M | 0.3 |
| FR-ADMIN-05 | Administrators shall browse the audit log filtered by actor, action, entity, entity id and date range, and export the result as CSV | M | 1.0 |
| FR-ADMIN-06 | Administrators shall list model versions with their evaluation metrics, activate one per component, and roll back to the previous active version | M | 1.0 |
| FR-ADMIN-07 | Administrators shall export a monthly summary as CSV with counts per category, zone, status, and return times | S | 1.0 |
| FR-ADMIN-08 | Administrators shall set and clear a site wide banner message, from the admin page or the `simpan admin banner` command, shown on every page until cleared | S | 1.0 |

### 3.14 Analytics and AI monitoring, `FR-ANL`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-ANL-01 | The hotspot page (AI-10) shall show lost and found counts per location on a campus map for a chosen period and category, with a heat layer and a ranked table | M | 1.0 |
| FR-ANL-02 | The analytics shall show breakdowns by category, hour of day and weekday, and weekly trends | M | 1.0 |
| FR-ANL-03 | The analytics shall show return rate, median time from found report to return, unclaimed rate and open case count | M | 1.0 |
| FR-ANL-04 | Any analytics cell built from fewer than three reports shall be suppressed (BR-PRV-05) | M | 1.0 |
| FR-ANL-05 | Analytics shall be served from materialised views refreshed every hour | M | 1.0 |
| FR-ANL-06 | The clustering page (AI-11) shall run clustering on open reports of the last 30 days on demand and weekly, list clusters with representative photos, and let an administrator mark members as duplicates | S | 1.0 |
| FR-ANL-07 | The AI monitoring page (AI-12) shall show the metrics listed in ai-features AI-12 for a chosen period and model version | M | 1.0 |

### 3.15 Audit and history, `FR-AUDIT`

| Id | Requirement | P | Rel |
|---|---|---|---|
| FR-AUDIT-01 | Every state changing action shall write an audit entry in the same database transaction, with actor, actor role, action, entity type and id, changed fields before and after, request id, IP, user agent and time | M | 0.1 |
| FR-AUDIT-02 | The application database role shall have no `UPDATE` or `DELETE` privilege on the audit table | M | 0.1 |
| FR-AUDIT-03 | An officer or administrator opening private details (lost report identifying details, found report hidden details, claim answers, evidence) shall be recorded as an access entry | M | 0.3 |
| FR-AUDIT-04 | Every status change of a lost report, found report, custody or claim shall be recorded in the status history with actor and reason, and shown as a timeline to the users allowed to see the record | M | 0.1 |

## 4. External interface requirements

### 4.1 User interface

| Id | Requirement |
|---|---|
| UI-01 | Mobile first responsive layout from 360 px width. Officer and admin screens are also optimised for 1366 by 768 desktops |
| UI-02 | Indonesian copy following [ui-guidelines.md](../ux/ui-guidelines.md) |
| UI-03 | Icons from Lucide as inline SVG. No emoji anywhere in the interface |
| UI-04 | Every page reachable from the role navigation of [information-architecture.md](../ux/information-architecture.md) |
| UI-05 | Every error state offers a next action. Validation messages sit next to the field |

### 4.2 Public REST API

| Id | Requirement |
|---|---|
| API-01 | JSON over HTTPS under `/api/v1`, described by `api/openapi.yaml` (OpenAPI 3.0.3, the latest version the Go code generator and validator fully support), which is the contract for the web client |
| API-02 | Errors as RFC 9457 problem details with a stable `code` field |
| API-03 | Cursor pagination, RFC 3339 timestamps in UTC, UUIDv7 identifiers |
| API-04 | Conventions and the endpoint catalog in [api.md](../architecture/api.md) |

### 4.3 Inference API

| Id | Requirement |
|---|---|
| ML-01 | JSON over HTTP on the internal network only, authenticated with a shared bearer token |
| ML-02 | Endpoints for image embedding, text embedding, image analysis, clustering, health and model listing, described in [matching-engine.md](../ai/matching-engine.md) section 9 |
| ML-03 | Every response names the model id and revision that produced it |

### 4.4 Email

| Id | Requirement |
|---|---|
| EM-01 | SMTP with STARTTLS or TLS, credentials from environment |
| EM-02 | Multipart HTML and plain text, Indonesian, sender `SimpanUGM <no-reply@...>` configured per environment |
| EM-03 | Development uses Mailpit so no real email leaves a laptop |

### 4.5 Storage

| Id | Requirement |
|---|---|
| ST-01 | A storage interface with two drivers: local filesystem and S3 compatible. The driver is selected by environment |
| ST-02 | Object keys never contain personal data: `media/<yyyy>/<mm>/<uuid>/<variant>.<ext>` |

## 5. Non functional requirements

### 5.1 Performance and capacity, `NFR-PERF`

Measured on the reference server of section 2.4 with the k6 scenarios in
[testing-strategy.md](../engineering/testing-strategy.md).

| Id | Requirement |
|---|---|
| NFR-PERF-01 | API read endpoints p95 at most 300 ms, write endpoints except uploads p95 at most 500 ms, at 100 concurrent virtual users |
| NFR-PERF-02 | Image upload processing p95 at most 2 s for a 4 MB photo |
| NFR-PERF-03 | From report submission to stored candidates: median at most 20 s, p95 at most 60 s |
| NFR-PERF-04 | Report assistant analysis p95 at most 4 s, hard timeout 8 s |
| NFR-PERF-05 | Largest Contentful Paint at most 2.5 s on a mid range Android phone over 4G for home, report form and report detail. JavaScript per route at most 200 KB compressed |
| NFR-PERF-06 | Capacity sized for 60,000 accounts, 100,000 reports per year and 500,000 images. Vector retrieval at that size at most 100 ms per query |

### 5.2 Availability and reliability, `NFR-AVAIL`

| Id | Requirement |
|---|---|
| NFR-AVAIL-01 | Production availability at least 99.5 percent per month. Planned maintenance only outside 07:00 to 21:00 WIB |
| NFR-AVAIL-02 | When the inference service is down, reports, claims and handovers keep working. Matching catches up when it returns |
| NFR-AVAIL-03 | Background jobs are at least once. Every handler is idempotent. Failed jobs are retried with exponential backoff and remain inspectable |
| NFR-AVAIL-04 | Recovery point objective 24 hours, recovery time objective 4 hours, proven by a restore drill before v1.0.0 |
| NFR-AVAIL-05 | Every service handles `SIGTERM` by finishing in flight requests within 20 s |

### 5.3 Security, `NFR-SEC`

Design and threat model in [security-privacy.md](../architecture/security-privacy.md).

| Id | Requirement |
|---|---|
| NFR-SEC-01 | Authentication, session management and access control meet OWASP ASVS 4.0.3 level 2 for chapters V2, V3 and V4 |
| NFR-SEC-02 | TLS 1.2 or newer only, HSTS with one year max age |
| NFR-SEC-03 | Passwords hashed with argon2id, memory 19 MiB, 2 iterations, parallelism 1, 16 byte salt |
| NFR-SEC-04 | Every state changing request is protected against cross site request forgery by `SameSite=Lax` cookies and an origin check on unsafe methods |
| NFR-SEC-05 | Rate limits of BR-SEC-01 on authentication, uploads, claims and handover code entry |
| NFR-SEC-06 | Security headers: Content Security Policy without `unsafe-eval`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `Permissions-Policy` limited to camera on report pages, `frame-ancestors 'none'` |
| NFR-SEC-07 | Authorization is enforced in `api` on every request. The web client hides what a role cannot do but is never trusted |
| NFR-SEC-08 | Secrets come only from environment variables or mounted files, never from the repository |
| NFR-SEC-09 | CI runs dependency vulnerability scans for Go, Node and Python. A release has no open critical or high finding |
| NFR-SEC-10 | The inference service is not reachable from the internet and rejects requests without the internal token |

### 5.4 Privacy, `NFR-PRIV`

| Id | Requirement |
|---|---|
| NFR-PRIV-01 | Personal data is processed according to UU No. 27 Tahun 2022: a stated purpose, consent at registration, minimal collection, access and deletion on request |
| NFR-PRIV-02 | Retention and anonymisation follow BR-RET |
| NFR-PRIV-03 | Image metadata is removed on upload (FR-MEDIA-04) |
| NFR-PRIV-04 | Identity numbers and card numbers are masked per BR-PRV-03 |
| NFR-PRIV-05 | Aggregates under three reports are suppressed (BR-PRV-05) |
| NFR-PRIV-06 | Logs never contain passwords, tokens, handover codes, private details or full identity numbers |

### 5.5 Usability and accessibility, `NFR-USE`

| Id | Requirement |
|---|---|
| NFR-USE-01 | A first time user completes a lost report with one photo in at most three minutes, median, in the usability test |
| NFR-USE-02 | System Usability Scale score at least 75 |
| NFR-USE-03 | WCAG 2.1 level AA: contrast, keyboard operation, focus visibility, labels, alternative text, no information by colour alone |
| NFR-USE-04 | Touch targets at least 44 by 44 px |
| NFR-USE-05 | Status is always shown as text and icon, never colour only |

### 5.6 Maintainability, `NFR-MAINT`

| Id | Requirement |
|---|---|
| NFR-MAINT-01 | Line coverage at least 80 percent on Go domain packages, 70 percent on the whole `api`, 70 percent on `ml/app`, 60 percent on `web/src` |
| NFR-MAINT-02 | Lint and type checks are clean on `main` for all three languages |
| NFR-MAINT-03 | Docs and `CHANGELOG.md` are updated in the pull request that changes behaviour |
| NFR-MAINT-04 | Database schema changes only through numbered migrations that run up and down in CI |

### 5.7 Observability, `NFR-OBS`

| Id | Requirement |
|---|---|
| NFR-OBS-01 | Structured JSON logs from every service with time, level, service, request id, user id when known, and message |
| NFR-OBS-02 | A request id is accepted from or generated at the edge and propagated from `web` to `api` to `ml` and into jobs |
| NFR-OBS-03 | Prometheus metrics as listed in [observability.md](../operations/observability.md) |
| NFR-OBS-04 | `/healthz` for liveness and `/readyz` for readiness on every service |

### 5.8 Portability and deployment, `NFR-PORT`

| Id | Requirement |
|---|---|
| NFR-PORT-01 | Every service ships as an OCI image. The whole system runs with Docker Compose on one host |
| NFR-PORT-02 | Configuration only through environment variables, documented in [local-development.md](../engineering/local-development.md) |
| NFR-PORT-03 | The same images run in development, staging and production |

### 5.9 Reproducibility of AI results, `NFR-REPRO`

| Id | Requirement |
|---|---|
| NFR-REPRO-01 | Every evaluation run fixes its seeds, records the dataset version hash, the split and the model revisions, and writes its configuration next to its metrics |
| NFR-REPRO-02 | Model weights are pinned to an exact Hugging Face revision hash, never a moving branch |
| NFR-REPRO-03 | Every stored match records the model version of each component and the fusion configuration version |
| NFR-REPRO-04 | The evaluation split is grouped by physical item, so no item appears in two splits. A test asserts it |

### 5.10 Legal and content, `NFR-LEGAL`

| Id | Requirement |
|---|---|
| NFR-LEGAL-01 | A privacy notice and terms of use page in Indonesian, versioned, linked from registration and the footer |
| NFR-LEGAL-02 | The product does not use the university logo or claim to be an official university service until the university approves it |

## 6. Data requirements

The logical model, every table and every constraint are in
[data-model.md](../architecture/data-model.md). Retention is in business-rules section 9.

## 7. Verification

| Method | Applies to |
|---|---|
| Automated unit and integration tests | Every `FR` with logic, every `BR` |
| End to end tests (Playwright) | The seven core journeys of prd section 6 |
| Load tests (k6) | NFR-PERF |
| Offline evaluation | FR-AI, FR-ASSIST and the AI targets of prd section 8 |
| Security scans and review | NFR-SEC, NFR-PRIV |
| Usability test | NFR-USE |
| Restore drill | NFR-AVAIL-04 |

The mapping from each requirement to its test is kept in
[traceability.md](traceability.md).
