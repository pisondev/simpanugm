# User Stories | SimpanUGM

Every story the team will build for v1.0, with acceptance criteria written so that each
one becomes a test. Priority is MoSCoW (**M**, **S**, **C**). Points are Fibonacci and
measure relative effort. Sprint dates are in [project-plan.md](../project/project-plan.md).

Each story becomes one GitHub issue titled `US-nn <title>` using the Story template.

## Summary

| Epic | Stories | Points | Sprints |
|---|---|---|---|
| E1 Accounts and access | US-01 to US-05 | 21 | 1, 2 |
| E2 Master data | US-06 to US-07 | 8 | 1, 2 |
| E3 Lost reporting | US-08 to US-12 | 21 | 1, 3 |
| E4 Found reporting | US-13 to US-17 | 21 | 1, 2, 3 |
| E5 Matching and explanations | US-18 to US-22 | 34 | 2 |
| E6 Report assistant | US-23 to US-24 | 13 | 2 |
| E7 Claims | US-25 to US-29 | 26 | 3 |
| E8 Custody and handover | US-30 to US-34 | 21 | 1, 3 |
| E9 Notifications | US-35 to US-36 | 13 | 2 |
| E10 Officer workspace | US-37 to US-41 | 26 | 1, 3, 4 |
| E11 Administration and audit | US-42 to US-45 | 18 | 1, 4 |
| E12 Analytics and AI monitoring | US-46 to US-49 | 26 | 4 |
| E13 Platform and operations | US-50 to US-55 | 34 | 1, 4 |
| **Total** | 55 stories | **282** | Sprint 1: 73, Sprint 2: 71, Sprint 3: 70, Sprint 4: 68 |

## E1 Accounts and access

### US-01 Register with a campus email

As a student, I want to create an account with my UGM email, so that I can report items.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 1 | FR-AUTH-01, FR-AUTH-02, FR-AUTH-03, FR-AUTH-14, BR-ACC-01 to BR-ACC-03 |

1. Given an email at `mail.ugm.ac.id`, when I submit name, email, a valid password and accept the privacy notice, then my account is created as unverified and a verification email is sent.
2. Given an email at `gmail.com`, when I submit, then I see "Gunakan email UGM (@mail.ugm.ac.id atau @ugm.ac.id)" and no account is created.
3. Given a password of 9 characters or one on the common list, when I submit, then the field shows the reason and nothing is created.
4. Given an existing email in any letter case, when I register, then I see the same success screen as a new user and the existing owner receives an email that someone tried to register, so account existence is not revealed.
5. Given the verification link, when I open it within 24 hours, then my account becomes verified and I land on Beranda. After 24 hours the link shows "Tautan kedaluwarsa" with a resend button limited by BR-ACC-03.

### US-02 Sign in and out

As a user, I want to sign in and out safely, so that my reports stay mine.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 1 | FR-AUTH-04 to FR-AUTH-07, BR-ACC-04, BR-SEC-01, BR-SEC-02 |

1. Given correct credentials, when I sign in, then an `HttpOnly` session cookie is set and I land on my role's home.
2. Given a wrong password or unknown email, then I see "Email atau kata sandi salah" in both cases.
3. Given six attempts in 15 minutes from one IP for one email, then the sixth answers 429 with a retry time.
4. Given an officer session idle for 61 minutes, when the officer acts, then they are sent to sign in and the action is not performed.
5. When I choose "Keluar dari semua perangkat", then every session of mine is revoked.

### US-03 Reset a forgotten password

As a user, I want to reset my password by email, so that I can recover my account.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 1 | FR-AUTH-08 |

1. Given any email, when I request a reset, then I always see "Jika email terdaftar, tautan sudah dikirim".
2. Given a link older than 30 minutes or already used, then it is refused.
3. After a successful reset all my sessions are revoked and I receive `account.password_changed`.

### US-04 Invite officers and administrators

As an administrator, I want to invite officers and assign them to posts, so that each post has accountable staff.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 1 | FR-AUTH-09, FR-ADMIN-01, FR-MD-06 |

1. When I invite an email with role `officer` and posts, then an invitation valid 72 hours is sent, and the email domain rule does not apply.
2. When the invitee sets a password, then the account is active with the role and posts.
3. When I change an officer's posts, then their queues change on their next request.

### US-05 Manage my profile and sessions

As a user, I want to keep my contact details current and see where I am signed in.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 2 | FR-AUTH-10, FR-AUTH-11, FR-NOTIF-04 |

1. I can edit name, phone, NIM or NIP. Changing email sends a confirmation to the new address and keeps the old one until confirmed.
2. I see my sessions with device and last seen, and can revoke any except the current one from the list.
3. I can switch optional notifications on or off per channel.

## E2 Master data

### US-06 Seed and browse campus master data

As a team member, I want categories, zones, locations and posts loaded from seed files, so that every environment starts with the same campus.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 1 | FR-MD-01, FR-MD-03, FR-MD-04, FR-MD-06, FR-MD-09 |

1. Running the seed command twice leaves the database identical to running it once.
2. The location picker finds "Perpustakaan Pusat" for the inputs "perpus", "perpustakaan" and "Perpus Pusat".
3. Every location has a zone and a coordinate inside the campus bounding box, checked by a seed validation test.

### US-07 Edit master data

As an administrator, I want to edit categories, locations and posts, so that the system follows campus changes.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 2 | FR-MD-02, FR-MD-05, FR-MD-07, FR-ADMIN-02 |

1. I can create, edit and deactivate each master data type. Deactivated rows disappear from pickers but stay on existing reports.
2. Editing a category's verification questions changes the form for new claims only.
3. Every change is in the audit log with before and after.

## E3 Lost reporting

### US-08 File a lost report

As an owner, I want to report a lost item from my phone in under three minutes, so that finders and officers can connect it to found items.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 1 | FR-LOST-01 to FR-LOST-04, BR-LST-01, BR-LST-07 |

1. The form has four steps: Foto (optional), Detail, Lokasi dan waktu, Ciri khusus, then a review screen.
2. Given no photo, I can still submit.
3. Given a time window of 8 days, the form refuses it with "Rentang waktu maksimal 7 hari".
4. Given a dropped connection on submit, when I retry, then exactly one report exists, thanks to the idempotency key.
5. Given I reload the page halfway, then my draft is restored.
6. After submit I land on the report detail with code `LST-YYMM-NNNN` and the text "Mencari kandidat..." until matching finishes.

### US-09 Follow my lost report

As an owner, I want to see the status and history of my report, so that I know what is happening.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 1 | FR-LOST-07, FR-LOST-08, FR-AUDIT-04 |

1. Laporan Saya lists my reports with code, item, status label, candidate count and last update.
2. The detail shows a timeline of every status change with time in WIB.

### US-10 Edit or close my lost report

As an owner, I want to correct or close my report, so that matching uses the right information.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 1 | FR-LOST-09, FR-LOST-10, FR-LOST-05 |

1. Edit is available only in `reported` and `potential_match`.
2. Changing the description triggers a rematch, and the candidate list refreshes.
3. Closing asks for a reason and withdraws my open claims after a confirmation.

### US-11 Keep or close a stale report, reopen it

As an owner, I want to be asked before my report expires, and be able to reopen it.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 3 | FR-LOST-11, FR-LOST-12, BR-LST-02 to BR-LST-04 |

1. Given 60 days of no activity, I receive the reminder with "Tetap buka" and "Tutup laporan".
2. Given no answer for 14 more days, the report closes as expired.
3. Given an expired report closed 20 days ago, I can reopen it and matching restarts.

### US-12 File a lost report for a walk in visitor

As an officer, I want to file a lost report for a visitor without an account, so that visitors are served too.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 2 | 1 | FR-LOST-13 |

1. The officer form adds visitor name and phone. Notifications for the report go to the post officers.

## E4 Found reporting

### US-13 File a found report

As a finder, I want to report a found item quickly with a photo, so that its owner can be found.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 1 | FR-FOUND-01 to FR-FOUND-04, FR-FOUND-07, BR-FND-01, BR-FND-02, BR-FND-04 |

1. At least one photo is required.
2. For a wallet (sensitivity `valuable`), the option "Saya simpan sementara" is not offered and the form explains why.
3. The form recommends the nearest active post in the zone of the found location.
4. After submit I see the code, the post address and opening hours, and the instruction to hand the item in.

### US-14 See if an owner may exist

As a finder, I want to know right away whether someone is looking for this item, so that I prioritise handing it in.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 2 | FR-FOUND-06, AI-05, BR-VIS 5.1 |

1. Given an active candidate of tier high, my found report shows "Kemungkinan pemiliknya sedang mencari" with the lost item name and category only.
2. No owner name, contact, description or photo is ever shown, verified by a response snapshot test.

### US-15 Be warned about a duplicate report

As a finder, I want to know if this item was already reported, so that I do not create a duplicate.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| S | 5 | 3 | FR-FOUND-05, AI-06 |

1. Given an open found report of the same parent category within 72 hours, 500 m and a duplicate score at least 0.80, when I reach the review step, then I see up to three similar reports with thumbnail, time and place.
2. I can choose "Ini barang yang sama" which opens the existing report, or "Bukan, lanjutkan".

### US-16 Be reminded to hand in the item

As a finder, I want reminders while the item is with me, and officers want to know about overdue items.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 3 | FR-FOUND-10, BR-FND-03 |

1. At 24 and 48 hours I receive reminders. At 48 hours post officers are alerted. At 72 hours the item tops the priority queue as overdue.
2. Receiving the item at a post cancels pending reminders.

### US-17 Browse the public found listing

As a member, I want to browse found items without photos, so that I can claim one I recognise.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 2 | 1 | FR-FOUND-12, FR-SEARCH-05, BR-PRV-01 |

1. The listing shows category, item name, zone, found date and holding post, filterable by category, zone and date.
2. The listing response contains no description, photo, hidden details or finder field.

## E5 Matching and explanations

### US-18 Embed and analyse every report

As the system, I want embeddings and attributes for every report, so that matching has its inputs.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 2 | FR-AI-01 to FR-AI-03, FR-AI-07, ML-01 to ML-03 |

1. After a report is submitted, its image and text embeddings exist within the time of NFR-PERF-03.
2. With the inference service stopped, report creation still succeeds and the job retries until the service returns.
3. Every stored embedding records the model version id.

### US-19 Rank candidates for every report

As an owner, I want the most likely found items ranked for my report, so that I look at a few instead of all.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 13 | 2 | FR-AI-04 to FR-AI-06, FR-AI-08, FR-AI-12 |

1. Matching follows matching-engine sections 4 to 7 exactly. Unit tests pin the score of hand computed examples to four decimals.
2. At most ten active candidates are stored per report, each with component scores, tier and versions.
3. A lost report without a photo still gets candidates through the cross modal score.
4. The nightly sweep rescores open reports without changing dismissed, confirmed or rejected rows.
5. The offline evaluation on the test split meets the AI targets of prd section 8.

### US-20 Understand why a candidate is suggested

As an owner, I want to see why a found item is suggested, so that I trust and judge the suggestion.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 2 | AI-02, BR-VIS-04 |

1. Each candidate shows the overall score as a percentage, a tier badge, and four bars: Kemiripan foto, Kemiripan deskripsi, Kedekatan lokasi, Kedekatan waktu.
2. Each bar has a one line reason, for example "Ditemukan 150 m dari lokasi terakhir" or "Ditemukan 5 jam setelah perkiraan waktu hilang".
3. When the visual score is cross modal, the bar reads "Kecocokan deskripsi dengan foto".
4. No text from hidden details or extracted visible text appears in the explanation.

### US-21 Give feedback or dismiss a candidate

As an owner, I want to dismiss candidates that are clearly not mine, so that my list stays useful, and the team gets labels.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 2 | FR-AI-09, FR-AI-10 |

1. "Bukan milik saya" hides the candidate for me and stores a negative feedback with an optional reason.
2. An officer dismissal hides it for everyone.

### US-22 Be told when a strong candidate appears later

As an owner, I want a notification when a strong candidate appears after I reported, so that I do not have to check daily.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 2 | FR-AI-06, FR-NOTIF-05, BR-NTF-01 |

1. Given my report and a new found report that scores tier high, I receive `match.high_for_owner` in app and by email.
2. Given three high candidates in one hour, I receive one notification, and later ones fold into the next allowed slot.

## E6 Report assistant

### US-23 Get suggestions from a photo

As an owner or finder, I want the form to suggest category, colour and brand from my photo, so that reporting is faster.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 2 | FR-ASSIST-01, FR-ASSIST-02, FR-ASSIST-04, FR-ASSIST-05, AI-03 |

1. Within 4 s of adding a photo, the category field shows the top suggestion preselected and two alternatives as chips. Colour and brand are prefilled when detected.
2. Each suggestion is editable. The submitted report records accepted, edited or ignored per field.
3. With the inference service down, the form works and shows "Saran otomatis sedang tidak tersedia".
4. A photo of a mock KTM returns visible text with the 16 digit number masked.

### US-24 Get a drafted description

As a finder, I want a description drafted for me, so that I only need to check it.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 2 | FR-ASSIST-03, AI-04 |

1. Given accepted attributes category Payung, colour biru, brand none, then the draft reads "Payung berwarna biru." and adds "Terdapat tulisan "...". " only when visible text is not masked and is short.
2. The draft is editable and never includes masked numbers.

## E7 Claims

### US-25 Claim a candidate

As an owner, I want to claim a found item from my candidates, so that I can get it back.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 3 | FR-CLAIM-01 to FR-CLAIM-04, FR-CLAIM-12, BR-CLM-01 to BR-CLM-04, BR-CLM-07 |

1. The claim form shows the category's questions, for a wallet: "Sebutkan isi dompet", "Ciri khusus di luar atau dalam dompet", plus a statement and optional evidence.
2. Given three open claims, a fourth is refused with the reason.
3. Given my own found report, the claim button is not shown and the API refuses it.
4. Claiming from the public listing creates a lost report from the claim details.

### US-26 Review a claim with the verification assistant

As an officer, I want a checklist that tells me what to verify, so that I make consistent decisions.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 3 | FR-CLAIM-11, AI-08, BR-CLM-10, FR-AUDIT-03 |

1. The review page shows claim answers next to the found item's hidden details, AI attributes and photos, with the checklist from ai-features AI-08.
2. Approve is disabled until every required item is `pass` or `not_applicable` with a note.
3. Opening the page writes an access entry.

### US-27 Ask for more information, approve or reject

As an officer, I want to request info, approve or reject a claim, so that the right person gets the item.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 3 | FR-CLAIM-05 to FR-CLAIM-07, FR-CLAIM-09, BR-CLM-08, BR-CLM-09 |

1. Request info requires a message and moves the claim to `info_requested`. The claimant's reply moves it back.
2. Approve is refused while custody is `with_finder`, with "Barang belum diterima di pos".
3. Approving one claim puts other open claims on the item on hold and notifies them.
4. Reject requires a reason from the list. `other` also requires a note.

### US-28 Withdraw a claim

As a claimant, I want to withdraw my claim, so that the officer stops working on it.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 2 | 3 | FR-CLAIM-08 |

1. Withdraw is available in every open state and asks for confirmation.

### US-29 Expire stalled claims

As the system, I want stalled claims to expire, so that items are not blocked forever.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 3 | FR-CLAIM-10, BR-CLM-05, BR-CLM-06 |

1. Given `info_requested` for 7 days, the claim expires.
2. Given `approved` for 7 days without handover, the claim expires, the code stops working and claims on hold return to review.
3. The claimant receives `claim.expiring` one day before.

## E8 Custody and handover

### US-30 Receive an item at a post

As an officer, I want to receive an item in under a minute, so that the desk keeps moving.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 1 | FR-CUST-01, BR-CUS-01, BR-CUS-03 |

1. I type or scan the found code, see the photos, enter the storage slot and confirm.
2. Custody becomes `in_custody` with a due date per sensitivity, and the finder is notified.
3. An officer not assigned to the post cannot receive the item.

### US-31 Hand over with a code

As an officer, I want to hand over an item only after the code and ID check, so that I am protected if a handover is disputed.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 3 | FR-CUST-03, FR-CUST-04, BR-HND-01 to BR-HND-04 |

1. Given the right code, ID type, last four characters and the name check, the handover completes in one transaction with every effect in FR-CUST-04.
2. Given five wrong codes, the code locks and only regeneration unlocks it.
3. The owner then sees "Konfirmasi barang sudah diterima".

### US-32 Manage the post inventory

As an officer, I want to see what my post holds and what is due, so that nothing is forgotten.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 1 | FR-CUST-05 |

1. The inventory lists items in custody with age, due date, sensitivity, claim state and slot, sorted by due date by default.

### US-33 Dispose of unclaimed items

As an officer, I want unclaimed items surfaced with a recorded disposition, so that storage is cleared accountably.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 3 | FR-CUST-06, FR-CUST-07, BR-CUS-02, BR-CUS-04 to BR-CUS-06 |

1. Seven days before the due date the post officers are warned.
2. After the due date an item without an open claim becomes `unclaimed`.
3. An identity item can only be disposed of as `transferred_to_unit`.
4. Retention can be extended once by up to 30 days with a reason.

### US-34 Transfer between posts

As an officer, I want to move an item to another post with a record, so that custody is always known.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| S | 2 | 3 | FR-CUST-02 |

1. A transfer records source, destination, officer and time, and appears in the custody history.

## E9 Notifications

### US-35 Receive notifications in the app

As a user, I want a notification bell that updates live, so that I see changes without reloading.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 2 | FR-NOTIF-01, FR-NOTIF-02, FR-NOTIF-06 |

1. A new notification appears within 2 s while the page is open.
2. After a network drop the stream resumes and missed notifications appear.
3. Each notification opens the page where I act.

### US-36 Receive email notifications

As a user, I want emails for important events, so that I learn about them away from the app.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 2 | FR-NOTIF-03, FR-NOTIF-04, BR-NTF-02, BR-NTF-03 |

1. Events marked email in the catalog send an Indonesian email with HTML and plain text parts.
2. A failed SMTP send is retried up to five times.
3. Officers receive the 07:00 digest only when it has content.

## E10 Officer workspace

### US-37 Search all reports

As an officer, I want to search every report by name, category, place, date, status or code.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 1 | FR-SEARCH-01 to FR-SEARCH-03, FR-SEARCH-06 |

1. "dompet hitam" finds reports with those words in name or description. "tumblr biru" finds a report named "Tumbler biru dongker", and "Raani" finds a reporter named Rani, through trigram similarity.
2. Typing `FND-2611-0042` opens that report.

### US-38 Work from a priority queue

As an officer, I want my home screen to tell me what to do next, so that urgent items are not missed.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 3 | FR-OFF-01, FR-OFF-02, FR-OFF-06, AI-09 |

1. The queue orders items by the priority score of ai-features AI-09 and shows the reason chips for each.
2. Counters match the inventory and claim lists exactly.

### US-39 Review matches side by side

As an officer, I want to compare a report and its candidates side by side, so that I can spot the owner fast.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 3 | FR-OFF-03, FR-OFF-04, FR-AI-11, AI-07 |

1. The board shows the selected report on the left and its ranked candidates on the right with photos, private details and component bars.
2. "Undang pemilik mengajukan klaim" notifies the owner.
3. "Hitung ulang" reruns matching for the report.

### US-40 Merge duplicates

As an officer, I want to merge duplicate reports, so that one item has one record.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| S | 5 | 3 | FR-OFF-05, AI-06 |

1. Merging closes the duplicate with a link to the survivor and moves its photos.

### US-41 Search by meaning

As an officer, I want to search with natural phrases, so that I find items described in other words.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| S | 3 | 4 | FR-SEARCH-04, AI-13 |

1. "tempat minum biru" finds a report described as "tumbler warna biru dongker" in the top five on the evaluation fixture.

## E11 Administration and audit

### US-42 Moderate reports and users

As an administrator, I want to hide abusive reports and suspend accounts.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 4 | FR-ADMIN-01, FR-ADMIN-03, FR-AUTH-12 |

1. A hidden report disappears from every listing and from matching, and its candidates are superseded.
2. Suspension revokes sessions immediately.

### US-43 Change runtime settings

As an administrator, I want to adjust limits and deadlines without a deploy.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 4 | FR-ADMIN-04 |

1. Each setting of business-rules section 10 is editable within its allowed range and takes effect within 60 s.

### US-44 Browse the audit log

As an administrator, I want to trace who did what, so that disputes can be settled.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 4 | FR-ADMIN-05, FR-AUDIT-01 to FR-AUDIT-03 |

1. I filter by actor, action, entity and date, and export the result as CSV.
2. The application cannot update or delete audit rows, proven by an integration test.

### US-45 Delete my account

As a member, I want to delete my account and personal data.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 4 | FR-AUTH-13, BR-RET-05 |

1. Deletion is refused while I have an approved claim.
2. After deletion my reports remain as anonymised statistics only.

## E12 Analytics and AI monitoring

### US-46 See hotspots on a campus map

As an administrator, I want to see where and when items are lost and found most, so that security can act.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 4 | FR-ANL-01, FR-ANL-04, FR-ANL-05, AI-10 |

1. A map of campus shows a heat layer and markers per location for the chosen period and category, and a ranked table beside it.
2. Cells under three reports show "< 3".

### US-47 See operational trends

As an administrator, I want trends and return rates, so that I can report monthly.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 4 | FR-ANL-02, FR-ANL-03, FR-ADMIN-07 |

1. Charts show weekly counts, category and hour of day breakdowns, return rate and median time to return.
2. The monthly CSV export matches the charts.

### US-48 Find clusters of related reports

As an administrator, I want related reports grouped, so that duplicates and patterns surface.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| S | 5 | 4 | FR-ANL-06, AI-11 |

1. Running clustering lists clusters with size, representative thumbnails and dominant category and zone.
2. I can mark members of a cluster as duplicates.

### US-49 Monitor the matching engine

As an administrator, I want to see whether the AI works well and whether it drifts.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 4 | FR-ANL-07, FR-ADMIN-06, AI-12 |

1. The page shows every metric listed in AI-12 for a chosen period and model version.
2. I can activate a model version and roll back to the previous one, which triggers a rescore.

## E13 Platform and operations

### US-50 Scaffold the services and the local stack

As a developer, I want `docker compose up` to start the whole system locally, so that everyone develops against the same stack.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 1 | NFR-PORT-01 to NFR-PORT-03, NFR-OBS-04 |

1. A fresh clone plus `make dev` brings up web, api, ml, PostgreSQL, Mailpit, and applies migrations and seeds.
2. Every service answers `/healthz` and `/readyz`.

### US-51 Continuous integration

As a developer, I want CI to check every pull request, so that `main` stays deployable.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 1 | NFR-MAINT-01, NFR-MAINT-02, NFR-MAINT-04, NFR-SEC-09 |

1. CI runs lint, type checks, unit and integration tests and vulnerability scans per area, only for areas the pull request touches, in under 10 minutes.

### US-52 Observability

As the team, I want logs, metrics and dashboards, so that we see problems before users report them.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 4 | NFR-OBS-01 to NFR-OBS-03 |

1. Grafana shows request rate, latency, error rate, job queue depth, match latency and inference latency.
2. A request id typed into the log search shows the request across web, api, ml and its jobs.

### US-53 Deploy to staging and production

As the team, I want tagged releases deployed reproducibly, so that the demo runs on the real stack.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 5 | 4 | NFR-PORT-01, NFR-SEC-02, NFR-AVAIL-05 |

1. A tag `v*` builds and publishes images. The deploy workflow pulls them on the server, migrates and restarts with no more than 10 s of errors.

### US-54 Backups and a restore drill

As the team, I want nightly backups and a proven restore, so that data loss is bounded.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 3 | 4 | NFR-AVAIL-04 |

1. Nightly backups run and are retained per runbook. A restore to a fresh server is performed once and timed.

### US-55 Security hardening

As the team, I want the security controls verified before release.

| P | Points | Sprint | Requirements |
|---|---|---|---|
| M | 8 | 4 | NFR-SEC-01 to NFR-SEC-10, NFR-PRIV-06 |

1. A ZAP baseline scan of staging reports no high finding.
2. The checklist in security-privacy section 8 is complete.
