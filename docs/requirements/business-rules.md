# Business Rules | SimpanUGM

Every limit, deadline, visibility decision and retention period the system enforces.
Requirements and code reference these ids instead of repeating the numbers. Values marked
**setting** can be changed by an administrator at runtime (section 10). All others change
only through a pull request.

Matching thresholds, weights and score definitions are not here. They belong to the
fusion configuration in [matching-engine.md](../ai/matching-engine.md), which is versioned
like a model.

## 1. Accounts, `BR-ACC`

| Id | Rule |
|---|---|
| BR-ACC-01 | Self registration is open only to email addresses in the allowed domain list, default `mail.ugm.ac.id` and `ugm.ac.id` (**setting**). Accounts created by invitation are exempt, because security staff may not hold a university address |
| BR-ACC-02 | An unverified account can sign in, browse the public found listing and edit its profile. It cannot create reports or claims. It is deleted 7 days after registration if still unverified |
| BR-ACC-03 | A verification email can be resent at most once per 60 seconds and 5 times per day |
| BR-ACC-04 | Session lifetime: `member` 30 days sliding, extended on use at most every 5 minutes. `officer` and `admin` 12 hours absolute and 60 minutes idle |
| BR-ACC-05 | One account per email, compared case insensitively |
| BR-ACC-06 | A role change, post assignment or suspension takes effect on the next request, because permissions are resolved per request |
| BR-ACC-07 | The last active administrator cannot be demoted, suspended or deleted |

## 2. Lost reports, `BR-LST`

| Id | Rule |
|---|---|
| BR-LST-01 | Time window: `lost_from <= lost_until`, `lost_until <= now + 5 minutes`, `lost_until - lost_from <= 7 days`, `lost_from >= now - 90 days` |
| BR-LST-02 | An open lost report without activity for 60 days (**setting**) triggers a reminder: "Masih mencari barang ini?" with the options to keep it open or close it. Activity means an edit, a candidate view, feedback or a claim |
| BR-LST-03 | If the owner does not answer within 14 days (**setting**) of the reminder, the report closes with reason `expired` |
| BR-LST-04 | A report closed as `expired`, `found_by_owner` or `no_longer_needed` can be reopened by its owner within 30 days (**setting**) of closure. Reopening restarts matching |
| BR-LST-05 | A new lost report is flagged as a likely duplicate of an open report by the same owner when both share the parent category, their time windows overlap, and their normalised semantic text score (matching-engine section 6.2) is at least 0.80 |
| BR-LST-06 | A `returned` report closes with reason `returned` when the owner confirms receipt, or automatically 7 days after handover |
| BR-LST-07 | A member can have at most 5 open lost reports |

## 3. Found reports, `BR-FND`

| Id | Rule |
|---|---|
| BR-FND-01 | Found time: `found_at <= now + 5 minutes` and `found_at >= now - 30 days` |
| BR-FND-02 | Custody `with_finder` is offered only for categories of sensitivity `normal`, and for at most 48 hours (**setting**). Items of sensitivity `valuable` or `identity` must go to a post |
| BR-FND-03 | While custody is `with_finder`: at 24 hours the finder is reminded. At 48 hours the finder is reminded again and the officers of the nearest post in the same zone are alerted. At 72 hours the report is flagged `overdue_custody` and enters the top of the officer priority queue |
| BR-FND-04 | A member can create at most 10 found reports in any rolling 7 days |
| BR-FND-05 | A found report closed as `returned_directly` requires a note and is shown in the officer queue for a light review |
| BR-FND-06 | Once a found report has an open claim, the finder can no longer edit or close it |

## 4. Media, `BR-MED`

| Id | Rule |
|---|---|
| BR-MED-01 | Per file at most 10 MB as received, shortest side at least 320 px. Per report at most 5 photos. Per claim at most 3 evidence images. Per user at most 30 uploads per hour |
| BR-MED-02 | A staged upload not attached within 24 hours is deleted |
| BR-MED-03 | An image whose SHA-256 equals one already on the same report is rejected with "Foto ini sudah ditambahkan" |

## 5. Visibility, `BR-VIS`

What each person can see of a record. "Officer" means any officer, because matching works
across the campus. Custody actions are limited by post assignment in section 7.

### 5.1 Lost report

| Field group | Owner | Finder of a matched found report | Other member | Officer, admin |
|---|---|---|---|---|
| Category, item name | yes | yes, only for candidates of tier medium or high | no | yes |
| Description, colour, brand, places, time window | yes | no | no | yes |
| Photos | yes | no | no | yes |
| Private identifying details | yes | no | no | yes, access logged |
| Owner identity and contact | yes | no | no | yes |

Lost reports are never listed publicly.

### 5.2 Found report

| Field group | Finder | Owner with a candidate of tier medium or high | Other member | Officer, admin |
|---|---|---|---|---|
| Category, item name, zone, found date, holding post | yes | yes | yes, public listing | yes |
| Public description, found location name, colour, brand | yes | yes | no | yes |
| Photos | yes | `display` variant, or only `thumb_blur` for sensitivity `identity` | no | yes, all variants |
| Hidden details | yes | no | no | yes, access logged |
| Visible text extracted from photos | no | no | no | yes |
| Finder identity and contact | yes | no | no | yes |
| Storage slot and custody history | custody status only | custody status only | no | yes |

### 5.3 Claims and matches

| Id | Rule |
|---|---|
| BR-VIS-01 | Candidates of tier `low` are never shown to members. Officers see them on request in the review board |
| BR-VIS-02 | A claim is visible to its claimant, officers and administrators. The finder never sees claims, only the outcome `returned` |
| BR-VIS-03 | Officer internal notes on a claim are never shown to the claimant. The decision reason and decision note are |
| BR-VIS-04 | The explanation of a match shown to an owner contains component scores and matched public attributes only, never words from hidden details or visible text |
| BR-VIS-05 | Hidden reports (moderation) are visible only to administrators |

## 6. Claims, `BR-CLM`

| Id | Rule |
|---|---|
| BR-CLM-01 | A member can have at most 3 open claims (**setting**). Open means `submitted`, `under_review`, `info_requested`, `approved` or `on_hold` |
| BR-CLM-02 | A member can have at most one open claim per found report. After a rejection the same member cannot claim that found report again unless an officer reopens the claim |
| BR-CLM-03 | A member cannot claim a found report they created |
| BR-CLM-04 | A member with 3 rejected claims within 30 days (**setting**) is marked `claim_restricted` and cannot submit claims until an administrator clears the mark |
| BR-CLM-05 | A claim in `info_requested` without a reply for 7 days (**setting**) becomes `expired` |
| BR-CLM-06 | A claim in `approved` without handover for 7 days (**setting**) becomes `expired`. Claims on hold for the same item return to `under_review` |
| BR-CLM-07 | A claim may be submitted and reviewed while custody is `with_finder`, but it can be approved only when custody is `in_custody`. Submitting such a claim escalates BR-FND-03 to the 48 hour stage immediately |
| BR-CLM-08 | Rejection reasons: `details_mismatch`, `insufficient_evidence`, `item_already_returned`, `duplicate_claim`, `suspected_fraud`, `other`. `other` requires a note |
| BR-CLM-09 | A claim created by an officer for a walk in visitor must be approved by a different officer than the one who created it |
| BR-CLM-10 | Approval requires every checklist item marked `required` to be `pass` or `not_applicable` with a note |

## 7. Custody and handover, `BR-CUS` and `BR-HND`

| Id | Rule |
|---|---|
| BR-CUS-01 | Retention starts when the item is received into custody. Period by sensitivity: `normal` 30 days, `valuable` 60 days, `identity` 30 days (**settings**). A category may override it |
| BR-CUS-02 | A daily job at 01:00 WIB moves found reports whose retention has passed and that have no open claim to `unclaimed`. If an open claim exists, the due date moves to the day after the claim closes |
| BR-CUS-03 | Only officers assigned to the holding post, or administrators, may receive, transfer, hand over or dispose of an item |
| BR-CUS-04 | Items of sensitivity `identity` can only be disposed of as `transferred_to_unit`, meaning handed to the issuing office such as the faculty academic office for a KTM |
| BR-CUS-05 | The post officers receive a retention warning 7 days before the due date |
| BR-CUS-06 | Retention can be extended once per item, by at most 30 days, with a reason |
| BR-HND-01 | Approval generates a 6 digit numeric handover code from a cryptographic random source. Only an HMAC-SHA256 of the code with the server pepper is stored. Regenerating the code invalidates the previous one |
| BR-HND-02 | Five wrong code entries for a claim lock the code. An officer must verify identity in person and regenerate it. Every attempt is audited |
| BR-HND-03 | The officer confirms with a checkbox that the name on the ID matches the claimant account name, and records the ID type and its last four characters |
| BR-HND-04 | For a walk in visitor claim, the code step is replaced by BR-CLM-09 and a match of the ID name with the recorded visitor name |

## 8. Notifications, `BR-NTF`

| Id | Rule |
|---|---|
| BR-NTF-01 | Match notifications per report: at most one every 6 hours (**setting**). New candidates inside the window are folded into the next notification |
| BR-NTF-02 | Officers receive a daily email digest at 07:00 WIB with pending claims, overdue custody and retention due this week, in addition to in app notifications |
| BR-NTF-03 | Non urgent emails are not sent between 22:00 and 06:00 WIB (**setting**) and go out at 06:00. In app notifications are never delayed. Account and security emails are never delayed |

## 9. Retention and privacy, `BR-RET` and `BR-PRV`

Rationale in [security-privacy.md](../architecture/security-privacy.md) section 6.

| Id | Rule |
|---|---|
| BR-RET-01 | 12 months after a report closes, it is anonymised: links to the reporter or finder and proxy contact are removed, item name, description, private details, hidden details, location note and visible text are cleared. Category, location, zone, timestamps, status and close reason stay for statistics |
| BR-RET-02 | 90 days after a report closes, its photos, all variants and their embeddings are deleted |
| BR-RET-03 | 12 months after a claim closes, its statement, answers, messages and evidence are deleted. Decision metadata stays, anonymised |
| BR-RET-04 | Audit entries are kept for 24 months, then deleted |
| BR-RET-05 | Account deletion runs within 30 days of the request: name becomes "Pengguna terhapus", email is replaced by a hash placeholder, phone and NIM or NIP are removed, sessions are deleted, open reports close as `account_deleted`, open claims are withdrawn. Deletion is refused while a claim is `approved`, until it is withdrawn or handed over |
| BR-RET-06 | Notifications are deleted 180 days after creation. Expired sessions are deleted 30 days after expiry |
| BR-PRV-01 | Found item photos are never published in a listing. They are shown only to the finder, to owners with a medium or high candidate, and to officers |
| BR-PRV-02 | The finder's identity is never shown to the owner and the owner's identity is never shown to the finder. The finder's thank you notification is anonymous |
| BR-PRV-03 | Before storing or returning visible text, and before storing a public description, any run of 16 digits and any run of 13 to 19 digits that passes the Luhn check is masked to its last four digits, for example `************3175` |
| BR-PRV-04 | Officer and administrator access to private fields is recorded per FR-AUDIT-03 |
| BR-PRV-05 | Analytics suppress any cell built from fewer than 3 reports (**setting**), shown as "< 3" |

## 10. Runtime settings

Stored in `system_settings`, editable by administrators, every change audited. The `api`
reads them through a cache that refreshes every 60 seconds.

| Key | Default | Allowed range | Rule |
|---|---|---|---|
| `auth.allowed_email_domains` | `["mail.ugm.ac.id","ugm.ac.id"]` | at least one domain | BR-ACC-01 |
| `lost.stale_days` | 60 | 14 to 180 | BR-LST-02 |
| `lost.expiry_grace_days` | 14 | 3 to 30 | BR-LST-03 |
| `lost.reopen_days` | 30 | 7 to 90 | BR-LST-04 |
| `found.with_finder_max_hours` | 48 | 12 to 72 | BR-FND-02 |
| `custody.retention_days.normal` | 30 | 7 to 120 | BR-CUS-01 |
| `custody.retention_days.valuable` | 60 | 14 to 180 | BR-CUS-01 |
| `custody.retention_days.identity` | 30 | 7 to 90 | BR-CUS-01 |
| `claim.max_open` | 3 | 1 to 10 | BR-CLM-01 |
| `claim.reject_flag_count` | 3 | 2 to 10 | BR-CLM-04 |
| `claim.reject_flag_window_days` | 30 | 7 to 90 | BR-CLM-04 |
| `claim.info_timeout_days` | 7 | 2 to 30 | BR-CLM-05 |
| `claim.approved_timeout_days` | 7 | 2 to 30 | BR-CLM-06 |
| `notify.match_throttle_hours` | 6 | 1 to 24 | BR-NTF-01 |
| `notify.email_quiet_hours` | `{"from":"22:00","to":"06:00"}` | any, WIB | BR-NTF-03 |
| `analytics.min_cell_count` | 3 | 3 to 20 | BR-PRV-05 |

## 11. Security limits, `BR-SEC`

| Id | Rule |
|---|---|
| BR-SEC-01 | Rate limits, answered with HTTP 429 and `Retry-After`: sign in 5 per 15 minutes per IP and email pair and 20 per 15 minutes per IP. Registration 5 per hour per IP. Password reset request 3 per hour per email. Uploads per BR-MED-01. Report assistant 30 per hour per user. Claim creation 5 per day per user. Handover code entry 30 per hour per officer. Any other endpoint 300 per minute per user and 600 per minute per IP |
| BR-SEC-02 | Five failed sign ins for one account within 15 minutes add a 15 minute delay for that account, without revealing whether the account exists |

## 12. Identifiers, `BR-ID`

| Id | Rule |
|---|---|
| BR-ID-01 | Human readable codes are `LST-YYMM-NNNN`, `FND-YYMM-NNNN` and `CLM-YYMM-NNNN`. `YYMM` is the creation month in WIB. `NNNN` is a zero padded sequence per prefix and month starting at 0001, widening to five digits past 9999 |
| BR-ID-02 | Codes are never reused and never change, including after anonymisation |
| BR-ID-03 | Internal identifiers are UUIDv7, which sort by creation time |
