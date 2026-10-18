# REST API | SimpanUGM

The public API is the contract between `web` and `api`. Its machine readable form is
`api/openapi.yaml` (OpenAPI 3.0.3), written by hand from this catalog in Sprint 1. Go server
interfaces are generated from it with `oapi-codegen` and the TypeScript client types with
`openapi-typescript`, so drift between the two sides fails the build.

## 1. Conventions

| Topic | Rule |
|---|---|
| Base path | `/api/v1`. A breaking change means `/api/v2` for the affected resource |
| Format | `application/json; charset=utf-8`, uploads `multipart/form-data` |
| Field names | `snake_case` |
| Identifiers | UUIDv7 strings in `id`. Human codes in `code`. A `{id}` path parameter of a lost report, found report or claim also accepts its code, so `/lost-reports/LST-2611-0007` works and page URLs can carry codes |
| Time | RFC 3339 in UTC with `Z`, for example `2026-11-02T07:15:00Z`. The client converts to WIB |
| Pagination | cursor based: `?limit=20&cursor=<opaque>`, response `{"items": [...], "next_cursor": "..."}`. `limit` from 1 to 100, default 20 |
| Filtering | query parameters named after fields, repeated for multiple values: `?status=reported&status=potential_match` |
| Sorting | `?sort=-created_at`, minus for descending, a whitelist per endpoint |
| Idempotency | `Idempotency-Key` header, required on report and claim creation, optional elsewhere |
| Concurrency | mutable resources return `version`. Updates send `If-Match: "<version>"` and receive `409 stale_version` when outdated |
| Caching | master data endpoints send `ETag` and honour `If-None-Match` |
| Request id | `X-Request-Id` echoed on every response |
| Authentication | session cookie `simpan_session` |
| CSRF | unsafe methods require an `Origin` header equal to the public origin, or `Sec-Fetch-Site: same-origin` |
| Limits | JSON bodies up to 64 KB, uploads up to 10 MB |

### 1.1 Errors

RFC 9457 problem details, `application/problem+json`.

```json
{
  "type": "https://simpanugm.my.id/problems/validation_failed",
  "title": "Data tidak valid",
  "status": 422,
  "code": "validation_failed",
  "detail": "Periksa kembali isian yang ditandai.",
  "instance": "/api/v1/lost-reports",
  "request_id": "01JBX9Q4Z8...",
  "errors": [
    { "field": "lost_until", "code": "window_too_long", "message": "Rentang waktu maksimal 7 hari" }
  ]
}
```

| Status | Code | When |
|---|---|---|
| 400 | `bad_request` | malformed JSON, unknown query parameter |
| 401 | `unauthenticated` | no valid session |
| 403 | `forbidden` | role lacks the permission |
| 403 | `account_unverified` | BR-ACC-02 |
| 403 | `csrf_origin_mismatch` | origin check failed |
| 404 | `not_found` | missing, or outside the caller's scope |
| 409 | `stale_version` | `If-Match` outdated |
| 409 | `invalid_transition` | the state machine refuses the event, `detail` names the current state |
| 409 | `custody_required` | approve while custody is `with_finder` |
| 409 | `checklist_incomplete` | approve with unresolved required checks |
| 413 | `payload_too_large` | body or file over the limit |
| 415 | `media_invalid_type` | not JPEG, PNG or WebP by signature |
| 422 | `validation_failed` | field errors in `errors` |
| 422 | `email_domain_not_allowed` | BR-ACC-01 |
| 422 | `claim_limit_reached`, `claim_exists`, `claim_rejected_before`, `found_not_claimable`, `lost_not_open` | BR-CLM-01, BR-CLM-02, FR-CLAIM-02 |
| 403 | `claim_restricted`, `own_found_report`, `same_officer` | BR-CLM-04, BR-CLM-03, BR-CLM-09 |
| 422 | `custody_not_received` | approve or hand over before the post holds the item, BR-CLM-07 |
| 422 | `report_limit_reached` | BR-LST-07, BR-FND-04 |
| 422 | `media_too_small`, `media_limit_reached`, `media_duplicate` | BR-MED |
| 422 | `handover_code_wrong` | a wrong code, counted before the error is returned |
| 422 | `handover_locked` | BR-HND-02, until an officer regenerates the code |
| 422 | `idempotency_key_reused` | same key, different request |
| 429 | `rate_limited` | with `Retry-After` |
| 503 | `assist_unavailable` | inference service down or slow, assistant only |
| 500 | `internal` | anything else, with `request_id` |

## 2. Endpoint catalog

Permission names are from [permissions.md](../requirements/permissions.md).

### 2.1 Authentication and account

| Method | Path | Permission | Purpose |
|---|---|---|---|
| POST | `/auth/register` | public | Create a member account, send verification |
| POST | `/auth/verify-email` | public | Consume a verification token |
| POST | `/auth/resend-verification` | session | BR-ACC-03 |
| POST | `/auth/login` | public | Create a session |
| POST | `/auth/logout` | session | Revoke the current session, `?all=true` for all |
| POST | `/auth/password/forgot` | public | Send a reset link |
| POST | `/auth/password/reset` | public | Consume a reset token |
| POST | `/auth/invitations/accept` | public | Set the password of an invited account |
| GET | `/me` | session | Profile, role, flags, posts, effective permissions |
| PATCH | `/me` | `account.self.update` | Name, phone, NIM or NIP |
| POST | `/me/email` | `account.self.update` | Start an email change, `{email, password}`. The link goes to the new address |
| POST | `/auth/email/confirm` | public | Consume the email change token from that link |
| PUT | `/me/password` | `account.self.update` | Change password with the current one, every other session ends |
| GET, PUT | `/me/notification-preferences` | `account.self.update` | Optional channels per group |
| GET | `/me/sessions` | `account.sessions.manage` | Active sessions with a device label and the network prefix of the address |
| DELETE | `/me/sessions/{id}` | `account.sessions.manage` | Revoke one |
| POST | `/me/deletion` | `account.self.delete` | Request account deletion |

### 2.2 Master data

| Method | Path | Permission | Purpose |
|---|---|---|---|
| GET | `/categories` | `masterdata.read` | Tree with sensitivity and verification questions |
| GET | `/zones` | `masterdata.read` | Zones with centroids |
| GET | `/locations` | `masterdata.read` | `?q=` search by name and alias, `?zone_id=` |
| GET | `/posts` | public | Posts with opening hours, `?near_location_id=` sorts by distance. Public because `/bantuan` lists them before sign in |
| GET | `/colours` | `masterdata.read` | Palette |

### 2.3 Media and assistant

| Method | Path | Permission | Purpose |
|---|---|---|---|
| POST | `/media` | `media.upload` | Upload one image as `file`, with `purpose`. Returns `{id, state: "staged"}` |
| GET | `/media/{id}` | `media.read` | Image bytes, `?variant=thumb|display|original|thumb_blur`, `Cache-Control: private` |
| DELETE | `/media/{id}` | `media.delete` | Remove a photo from an editable report |
| POST | `/assist/analyze-image` | `assist.analyze` | `{media_id, report_type}` of the caller's own staged photo returns up to three known leaf categories, up to two palette colours, a brand when the OCR match scores 0.85 or more, and the masked visible text (AI-03). `503 assist_unavailable` when ml is missing, down or slower than 8 s; report creation accepts an `assist_outcome` object with `accepted`, `edited` or `ignored` per field |
| POST | `/assist/describe` | `assist.analyze` | `{category_id, colours, brand, visible_text}` returns the template draft (AI-04) |
| POST | `/assist/duplicates` | `assist.analyze` | Draft found report fields and media ids, returns similar reports (AI-06) |

### 2.4 Lost reports

| Method | Path | Permission | Purpose |
|---|---|---|---|
| POST | `/lost-reports` | `report.lost.create` | Create. Proxy fields accepted from officers |
| GET | `/lost-reports` | `report.lost.read` | Members: own. Officers: all, with search filters |
| GET | `/lost-reports/{id}` | `report.lost.read` | Detail, projected by role |
| PATCH | `/lost-reports/{id}` | `report.lost.update` | Edit with `If-Match` |
| POST | `/lost-reports/{id}/close` | `report.lost.close` | `{reason, note}` |
| POST | `/lost-reports/{id}/reopen` | `report.lost.reopen` | BR-LST-04 |
| POST | `/lost-reports/{id}/keep-open` | `report.lost.update` | Answer to the stale reminder |
| POST | `/lost-reports/{id}/confirm-receipt` | `report.lost.update` | L9 by the owner |
| GET | `/lost-reports/{id}/history` | `report.history.read` | Status timeline |
| GET | `/lost-reports/{id}/matches` | `match.read` | Candidates projected by role. The owner sees active medium and high candidates in rank order, officers add `?include_low=true` for tier low (BR-VIS-01) |

### 2.5 Found reports

| Method | Path | Permission | Purpose |
|---|---|---|---|
| POST | `/found-reports` | `report.found.create` | Create |
| GET | `/found-reports` | `report.found.read` | Members: own. Officers: all with filters |
| GET | `/found-reports/public` | `report.found.read` public scope | Public listing (FR-FOUND-12) |
| GET | `/found-reports/{id}` | `report.found.read` | Detail projected by role and scope |
| PATCH | `/found-reports/{id}` | `report.found.update` | Edit with `If-Match` |
| POST | `/found-reports/{id}/close` | `report.found.close` | Finder close, F13 |
| GET | `/found-reports/{id}/history` | `report.history.read` | Status and custody timeline |
| GET | `/found-reports/{id}/match-hint` | finder or officer | AI-05: item name and category of lost reports with an active high candidate, nothing else |
| GET | `/found-reports/{id}/matches` | `officer.queue.read` | Candidates for officers, `?include_low=true` as above |

### 2.6 Matches

| Method | Path | Permission | Purpose |
|---|---|---|---|
| GET | `/matches/{id}` | `match.read` | Candidate detail with explanation, projected by role. The officer detail adds the hidden details, an access logged read |
| POST | `/matches/{id}/feedback` | `match.feedback` | `{verdict: relevant or not_relevant, reason?}`, a label for the evaluation set |
| POST | `/matches/{id}/dismiss` | `match.dismiss` | `{reason?}`. The owner hides the pair for themselves, an officer moves it to `dismissed` (M4). Both store a `not_relevant` label and settle both reports |
| POST | `/matches/{id}/pin` | officer | AI-07 |
| POST | `/matches/{id}/invite-claim` | officer | `match.officer_invites_claim` |
| POST | `/reports/{type}/{id}/rematch` | `match.rematch` | On demand rescoring |

### 2.7 Claims

| Method | Path | Permission | Purpose |
|---|---|---|---|
| POST | `/claims` | `claim.create` | `{found_report_id, lost_report_id?, match_id?, statement, answers, evidence_media_ids, preferred_visit}`. Without `lost_report_id` a lost report is created from `lost_report` fields in the body |
| GET | `/claims` | `claim.read` | Members: own. Officers: queue filters `status`, `post_id`, `found_report_id` |
| GET | `/claims/{id}` | `claim.read` | Detail. Officer view adds internal note and checklist. Claimant view adds the handover code while `approved` |
| POST | `/claims/{id}/pick-up` | `claim.review` | K2 |
| POST | `/claims/{id}/request-info` | `claim.review` | K3, `{message}` |
| POST | `/claims/{id}/respond` | own | K4, `{message, evidence_media_ids}` |
| GET, PUT | `/claims/{id}/checks` | `claim.review` | AI-08 checklist and results |
| POST | `/claims/{id}/approve` | `claim.approve` | K6, `{decision_note}` with `If-Match` |
| POST | `/claims/{id}/reject` | `claim.review` | K10, `{reason, note}` |
| POST | `/claims/{id}/revoke` | `claim.revoke` | K13, `{reason}` |
| POST | `/claims/{id}/reopen` | `claim.review` | K15, `{reason}` |
| POST | `/claims/{id}/withdraw` | `claim.withdraw` | K14 |
| POST | `/claims/{id}/handover-code` | `custody.handover` | Regenerate the code, BR-HND-02 |
| GET, POST | `/claims/{id}/messages` | `claim.message` | Thread |
| PATCH | `/claims/{id}/internal-note` | `claim.internal_note` | Officers only |

### 2.8 Custody

| Method | Path | Permission | Purpose |
|---|---|---|---|
| POST | `/found-reports/{id}/receive` | `custody.receive` | C3, `{post_id, storage_slot, condition_note}` |
| POST | `/found-reports/{id}/transfer` | `custody.transfer` | C4, `{to_post_id, note}` |
| POST | `/found-reports/{id}/extend-retention` | `custody.extend` | `{days, reason}` |
| POST | `/found-reports/{id}/handover` | `custody.handover` | `{claim_id, code, recipient_id_type, recipient_id_last4, name_confirmed}` |
| POST | `/found-reports/{id}/dispose` | `custody.dispose` | F12, `{kind, recipient, note}` |
| POST | `/found-reports/{id}/revert-unclaimed` | `custody.dispose` | F11 |
| GET | `/posts/{id}/inventory` | `custody.inventory.read` | FR-CUST-05 |

### 2.9 Officer workspace and search

| Method | Path | Permission | Purpose |
|---|---|---|---|
| GET | `/officer/summary` | `officer.queue.read` | Counters, `?scope=posts|campus` |
| GET | `/officer/queue` | `officer.queue.read` | AI-09 work items with score and reasons |
| GET | `/officer/board` | `officer.board.read` | `?report_type=&report_id=` AI-07 data in one call |
| GET | `/officer/duplicates` | `report.mark_duplicate` | Suggested duplicate pairs |
| POST | `/officer/duplicates/{id}/confirm` | `report.mark_duplicate` | Merge, FR-OFF-05 |
| POST | `/officer/duplicates/{id}/reject` | `report.mark_duplicate` | |
| GET | `/search` | `search.reports` | `?q=&type=&category_id=&zone_id=&location_id=&from=&to=&status=&custody=&post_id=&semantic=true`. Staff only. Members find their own reports through the list endpoints and everything else through the public listing |

### 2.10 Notifications

| Method | Path | Permission | Purpose |
|---|---|---|---|
| GET | `/notifications` | session | Newest first, `?unread=true` |
| GET | `/notifications/unread-count` | session | |
| POST | `/notifications/{id}/read` | session | |
| POST | `/notifications/read-all` | session | |
| GET | `/notifications/stream` | session | `text/event-stream`, events `notification` and `heartbeat`, honours `Last-Event-ID` |

### 2.11 Administration

| Method | Path | Permission | Purpose |
|---|---|---|---|
| GET | `/admin/users` | `user.read` | Search and filter |
| POST | `/admin/users/invitations` | `user.invite` | `{email, full_name, role, post_ids}` |
| PATCH | `/admin/users/{id}` | `user.update_role`, `user.assign_post` | Role and posts |
| POST | `/admin/users/{id}/suspend`, `/unsuspend` | `user.suspend` | |
| POST | `/admin/users/{id}/revoke-sessions` | `account.sessions.manage` | |
| POST | `/admin/users/{id}/clear-claim-restriction` | `user.clear_claim_restriction` | |
| DELETE | `/admin/users/{id}` | `user.delete` | Anonymise, BR-RET-05 |
| GET, POST, PATCH | `/admin/categories`, `/admin/categories/{id}` | `masterdata.manage` | US-07. Lists include switched off rows, `PATCH {active: false}` switches a row off, rows are never deleted. Same shape for `/admin/zones`, `/admin/locations` (`?q=&zone_id=`) and `/admin/posts` |
| GET, PUT | `/admin/category-compatibility` | `masterdata.manage` | `PUT {category_a, category_b, factor, active}` stores one pair of leaves, a pair is switched off rather than deleted |
| POST | `/admin/reports/{type}/{id}/hide`, `/unhide` | `report.moderate` | `{reason}` |
| GET, PATCH | `/admin/settings` | `settings.read`, `settings.manage` | business-rules section 10 |
| GET | `/admin/audit-logs` | `audit.read` | Filters, `Accept: text/csv` exports |
| GET | `/admin/model-versions` | `ai.models.read` | |
| POST | `/admin/model-versions/{id}/activate` | `ai.models.activate` | Also used for rollback |
| GET | `/admin/ai/monitoring` | `ai.monitoring.read` | `?from=&to=&model_version_id=` AI-12 |
| GET | `/admin/ai/clusters` | `ai.clusters.read` | Latest run, `?run_id=` |
| POST | `/admin/ai/clusters/runs` | `ai.clusters.run` | Start a run |
| GET | `/analytics/hotspots` | `analytics.read` | `?from=&to=&type=&category_id=&zone_id=` AI-10 |
| GET | `/analytics/summary` | `analytics.read` | KPIs |
| GET | `/analytics/trends` | `analytics.read` | Weekly series, hour by weekday grid |
| GET | `/analytics/export` | `analytics.export` | Monthly CSV, `?month=2026-11` |

### 2.12 Operational endpoints

Not under `/api/v1` and not routed by Caddy to the internet. The one exception is
`/api/healthz`, a public alias of `/healthz` used by the synthetic availability check.

| Path | Purpose |
|---|---|
| `/healthz` | process alive, also public as `/api/healthz` |
| `/readyz` | database reachable, migrations at the expected version, settings loaded, for `worker` also River running |
| `/metrics` | Prometheus, internal network only |

## 3. Examples

### 3.1 Create a lost report

```http
POST /api/v1/lost-reports
Idempotency-Key: 6f1d3b0e-2a51-4f61-9a3b-1c0b7e0b9d55
Content-Type: application/json

{
  "category_id": "0192f0a0-5d1e-7c3a-9b1e-2f6a8c1d0e11",
  "item_name": "Dompet kulit cokelat",
  "description": "Dompet lipat dua warna cokelat, isi KTM, kartu ATM, dan uang sekitar 50 ribu.",
  "private_details": "Ada gantungan kunci kecil bentuk kucing di resletingnya.",
  "colours": ["cokelat"],
  "brand": null,
  "lost_from": "2026-11-02T03:00:00Z",
  "lost_until": "2026-11-02T06:30:00Z",
  "places": [
    { "location_id": "0192f0a0-6a10-7b2c-8d3e-4f5a6b7c8d90" },
    { "location_id": "0192f0a0-6a10-7b2c-8d3e-4f5a6b7c8d91" }
  ],
  "location_note": "Antara perpustakaan dan kantin",
  "media_ids": [],
  "assist_outcome": null
}
```

```http
HTTP/1.1 201 Created
Location: /api/v1/lost-reports/0192f1b2-0c3d-7e4f-a5b6-c7d8e9f0a1b2

{
  "id": "0192f1b2-0c3d-7e4f-a5b6-c7d8e9f0a1b2",
  "code": "LST-2611-0007",
  "status": "reported",
  "matching": { "state": "pending" },
  "version": 1,
  "created_at": "2026-11-02T07:01:12Z"
}
```

### 3.2 Candidates as the owner sees them

```json
{
  "items": [
    {
      "id": "0192f1c4-...",
      "rank": 1,
      "tier": "high",
      "overall": 0.8124,
      "status": "active",
      "found_report": {
        "id": "0192f1c3-...",
        "code": "FND-2611-0012",
        "item_name": "Dompet cokelat",
        "category": { "id": "...", "slug": "dompet", "name": "Dompet", "parent_name": "Dompet dan Tas Kecil", "sensitivity": "valuable" },
        "description": "Dompet lipat warna cokelat ditemukan di meja baca lantai 2.",
        "colours": ["cokelat"],
        "found_at": "2026-11-02T05:40:00Z",
        "found_location": { "id": "...", "name": "Perpustakaan Pusat", "zone_id": "...", "zone_name": "Klaster Sosio Humaniora" },
        "holding_post": { "id": "...", "name": "Pos Lost and Found Perpustakaan" },
        "custody_status": "in_custody",
        "photos": [{ "media_id": "0192f1c5-...", "variant": "display" }]
      },
      "lost_report": { "id": "...", "code": "LST-2611-0007", "item_name": "Dompet kulit cokelat", "category": { "...": "..." }, "lost_from": "...", "lost_until": "..." },
      "explanation": {
        "visual_mode": "cross_modal",
        "visual": { "level": "high", "reason": "Deskripsimu sangat cocok dengan foto barang temuan.", "missing": false },
        "text": { "level": "high", "reason": "Deskripsi sangat mirip.", "chips": ["Warna sama: cokelat"], "missing": false },
        "location": { "level": "high", "reason": "Ditemukan di lokasi yang sama.", "missing": false },
        "time": { "reason": "Ditemukan dalam rentang waktu kehilangan.", "missing": false },
        "category": { "reason": "Kategori sama.", "missing": false }
      },
      "scored_at": "2026-11-02T07:01:20Z"
    }
  ]
}
```

A photo of an `identity` item reaches the owner as `thumb_blur` only, and the media
endpoint refuses its other variants to that owner. The officer projection of the same match
adds `scores` with every component number, `flags`, `component_versions`, the owner name on
`lost_report`, and on `GET /matches/{id}` the hidden details.
