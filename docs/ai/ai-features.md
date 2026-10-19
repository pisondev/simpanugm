# AI Features by Role | SimpanUGM

The concept document asks that every role meets AI directly in its workflow, not only as
a backend process. This page specifies each AI feature: what the user sees, what feeds
it, how it behaves when the AI is unavailable, and how we know it works. The engine
underneath is [matching-engine.md](matching-engine.md).

| Id | Feature | Role | Release |
|---|---|---|---|
| AI-01 | Visual Match Finder | Owner | 0.2 |
| AI-02 | Explainable Match | Owner, officer | 0.2 |
| AI-03 | Smart Report Assistant | Owner, finder | 0.2 |
| AI-04 | AI Assisted Item Description | Finder | 0.2 |
| AI-05 | Instant Match Suggestions | Finder | 0.2 |
| AI-06 | Duplicate and Related Report Detection | Finder, officer | 0.3 |
| AI-07 | AI Match Review Board | Officer | 0.3 |
| AI-08 | Verification Assistant | Officer | 0.3 |
| AI-09 | AI Priority Queue | Officer | 0.3 |
| AI-10 | Lost and Found Hotspot Analytics | Admin, officer read only | 1.0 |
| AI-11 | AI Case Clustering | Admin | 1.0 |
| AI-12 | AI and Matching Monitoring | Admin | 1.0 |
| AI-13 | Semantic Search | Officer, admin | 1.0 |

Every feature follows three rules. The AI output is labelled as a suggestion. A human
action is required before anything changes state. The feature degrades to the plain
workflow when `ml` is down.

## Owner

### AI-01 Visual Match Finder

| | |
|---|---|
| Where | Lost report detail, section "Kandidat barang temuan" |
| Shows | Up to ten candidates of tier medium or high, ordered by score: `display` photo (blurred thumbnail for identity items), item name, category, found place name, found date, holding post, overall score as a percentage, tier badge "Sangat mirip" or "Mirip" |
| Actions | Lihat detail, Ajukan klaim, Bukan milik saya |
| Empty state | "Belum ada barang temuan yang cocok. Kami akan memberi tahu kamu jika ada." while status is `reported` |
| While computing | "Mencari kandidat..." with a spinner, replaced through the notification stream when `match.report` finishes |
| Fallback | Candidates already stored stay visible. New ones appear when the engine catches up |
| Measured by | Online rank of the confirmed match, owner dismiss rate (AI-12) |

### AI-02 Explainable Match

| | |
|---|---|
| Where | Candidate detail for owners, review board for officers |
| Shows to owners | Four bars, Kemiripan foto (or Kecocokan deskripsi dengan foto), Kemiripan deskripsi, Kedekatan lokasi, Kedekatan waktu, each with the reason line of matching-engine section 11, attribute chips, and the category relation |
| Shows to officers | Everything above plus the numeric component scores, the category factor, the visual mode, the `name_hit` flag, the model and fusion versions, and when it was scored |
| Never shows | Words from hidden details or visible text (BR-VIS-04) |
| Measured by | Usability test question "Saya paham mengapa barang ini disarankan", target at least 4 on a 5 point scale |

### AI-03 Smart Report Assistant

| | |
|---|---|
| Where | Step 1 of both report forms, right after a photo is added |
| Calls | `POST /api/v1/assist/analyze-image` with the staged media id, which calls `ml /v1/analyze/image` |
| Fills | Category: the top suggestion preselected, two alternatives as chips. Colour: up to two chips. Brand: the top brand candidate. Each filled field shows a small "Saran otomatis" label until the user touches it |
| Records | Per field `suggested`, `accepted`, `edited` or `ignored`, stored on the report as `assist_outcome` JSON for AI-12 |
| Privacy | Visible text masked per BR-PRV-03. For identity categories visible text is not shown in the form at all |
| Timing | Result within 4 s p95. After 8 s the call is abandoned silently |
| Fallback | Form unchanged, note "Saran otomatis sedang tidak tersedia" |
| Measured by | Top 3 category accuracy offline, acceptance rate online, completion time in the usability test |

## Finder

### AI-04 AI Assisted Item Description

| | |
|---|---|
| Where | Step 2 of the found report form, the public description field |
| Template | `{item} berwarna {colour1}[ dan {colour2}][, merek {brand}].[ Terdapat tulisan "{text}".]` where `{item}` is the leaf category name in lower case |
| Text rule | The visible text sentence is added only if the text is at most 40 characters after masking, contains no masked run, and the category is not `identity` |
| Example | "Botol minum berwarna biru dan perak, merek Corkcicle." |
| Behaviour | The draft is inserted only into an empty field. Editing it is expected. A button "Buat ulang dari foto" regenerates it |
| Fallback | Empty field with the placeholder "Contoh: Payung lipat warna hitam, gagang kayu" |

### AI-05 Instant Match Suggestions

| | |
|---|---|
| Where | Found report detail, right after submission and whenever the finder returns |
| Shows | If an active candidate of tier high exists: "Kemungkinan pemiliknya sedang mencari barang ini" with the lost report's item name and category. If only medium: "Ada laporan kehilangan yang mirip". Else nothing |
| Never shows | Owner identity, contact, description, photos, places, or the score (BR-VIS 5.1) |
| Purpose | Motivates the finder to hand the item in. When custody is `with_finder` the hint is paired with the post address and a "Petunjuk arah" link |
| Measured by | Median time from found report to custody, split by whether a hint was shown |

### AI-06 Duplicate and Related Report Detection

| | |
|---|---|
| Where | Review step of the found report form for finders. A "Kemungkinan duplikat" tab in the officer workspace |
| Logic | matching-engine section 8 |
| Finder view | Up to three similar open found reports with thumbnail, item name, place and relative time. "Ini barang yang sama" opens the existing report. "Bukan, lanjutkan" submits |
| Officer view | Pairs above the flag with both reports side by side and the action "Gabungkan" (FR-OFF-05) |
| Measured by | Share of duplicate pairs confirmed by officers among those flagged |

## Officer

### AI-07 AI Match Review Board

| | |
|---|---|
| Where | Officer workspace, "Papan Tinjauan" |
| Layout | Left column, the selected report with every field including private details, all photos, timeline. Right column, its candidates of every tier ordered by score, each expandable to the AI-02 officer view with photos side by side |
| Entry points | From the priority queue, from a search result, from a notification |
| Actions per candidate | Tandai kuat (pins it at the top for every officer), Abaikan with a reason (match `dismissed`), Undang pemilik mengajukan klaim (`match.officer_invites_claim`), Buka klaim when one exists |
| Actions per report | Hitung ulang (on demand rematch), Tandai duplikat |
| Access | Opening private details writes an access entry (FR-AUDIT-03) |

### AI-08 Verification Assistant

A checklist generated when a claim is created and refreshed when evidence or answers
change. Each item has a suggestion computed by the system and a result recorded by the
officer. Approval needs every required item resolved (BR-CLM-10).

| # | Item | Required | System suggestion |
|---|---|---|---|
| V1 | Jawaban ciri khusus sesuai dengan detail tersembunyi barang | yes | Shows the claim answers and the found report's hidden details side by side with an indicator from `s_sem` between them: tinggi at least 0.7, sedang at least 0.4, rendah below |
| V2 | Nama pada kartu atau dokumen sesuai dengan nama pengklaim | yes, for sensitivity `identity` | `name_hit` true suggests pass, false suggests check |
| V3 | Warna dan merek yang disebut sesuai dengan foto | yes, when a mismatch is detected | Compares the claimant's colours and brand with the AI attributes of the found photos. A mismatch reads, for example, "Disebut: merah. Terdeteksi: biru" |
| V4 | Waktu kehilangan masuk akal | yes, when `s_time` is 0 | Flags a found time before the loss window |
| V5 | Lokasi kehilangan masuk akal | no | Flags `s_loc` below 0.1 |
| V6 | Pemeriksaan khusus kategori | yes | One item per check hint of the category's verification questions, for example for electronics "Pengklaim dapat membuka kunci perangkat atau menunjukkan nomor seri atau IMEI" |
| V7 | Bukti pendukung relevan | no | Shown only when evidence exists |
| V8 | Riwayat pengklaim | info only | "{n} klaim ditolak dalam 90 hari terakhir" |
| V9 | Klaim lain atas barang ini | info only | "{n} klaim lain sedang aktif", with links |

Results are `pass`, `fail`, `not_applicable`, each with an optional note, required for
`not_applicable`. The results and the suggestions at decision time are stored with the
claim, so a later audit sees what the officer saw.

### AI-09 AI Priority Queue

The officer home lists work items for the officer's posts, ordered by a priority score
that combines the match signal with operational urgency. The formula is fixed and shown
in the UI through reason chips, so officers can trust the order.

```
P = 0.35 * m + 0.20 * s + 0.20 * a + 0.25 * u
```

| Term | Meaning | Values |
|---|---|---|
| `m` | Best active match score of the item | 0 to 1, 0 without a candidate |
| `s` | Sensitivity of the category | `identity` 1.0, `valuable` 0.7, `normal` 0.3 |
| `a` | Age pressure | in custody: elapsed share of the retention period. With finder: elapsed share of 48 hours. Clamped to 0 to 1 |
| `u` | Pending action | 1.0 for a claim waiting over 24 hours, overdue custody (BR-FND-03 at 72 hours), or retention due within 3 days. 0.5 for a claim waiting under 24 hours. Else 0 |

| Work item type | Appears when | Primary action |
|---|---|---|
| `claim_review` | claim `submitted`, or `under_review` with a fresh reply | Periksa klaim |
| `high_match_no_claim` | candidate of tier high, no claim, owner not yet invited | Undang pemilik |
| `overdue_custody` | BR-FND-03 reached 72 hours | Hubungi penemu |
| `retention_due` | due within 7 days | Lihat barang |
| `unclaimed_disposition` | found report `unclaimed` | Tentukan tindak lanjut |
| `direct_return_review` | found report closed `returned_directly` | Tinjau |

Reason chips, for example: "Skor kecocokan 86%", "Kartu identitas", "Klaim menunggu 2 hari",
"Masa simpan habis 3 hari lagi". The weights live in the fusion configuration document as
`priority` so they are versioned with the rest.

## Administrator

### AI-10 Lost and Found Hotspot Analytics

| | |
|---|---|
| Where | Admin analytics, "Peta Rawan" tab. Officers have a read only view for their zones |
| Map | Campus map on OpenStreetMap tiles with Leaflet. A heat layer weighted by report count per location and markers sized by count. Toggle lost, found, both |
| Filters | Period (7, 30, 90 days, semester, custom), category parent, zone |
| Table | Locations ranked by lost count with found count, return rate, median time to return |
| Time patterns | Hour of day by weekday heat grid, weekly trend lines |
| Insights | Up to three generated lines when a location's count in the period exceeds its trailing 8 week mean by more than two standard deviations and at least 5 reports, for example "Kehilangan di Perpustakaan Pusat naik tajam minggu ini (12 laporan, rata rata 4)" |
| Privacy | Cells under 3 are shown as "< 3" (BR-PRV-05) |
| Source | Materialised views refreshed hourly (FR-ANL-05) |

### AI-11 AI Case Clustering

| | |
|---|---|
| Where | Admin analytics, "Klaster" tab |
| Input | Open reports of the last 30 days. Vector per report: the mean of its image embeddings (or its `clip_text` when it has no photo) and its `e5`, each L2 normalised, weighted 0.5 and 0.5, concatenated |
| Algorithm | HDBSCAN from scikit-learn in `ml /v1/cluster`, `min_cluster_size` 2, `min_samples` 1, Euclidean distance on the normalised vectors |
| Schedule | Weekly on Monday 03:00 WIB and on demand, stored as a run with its parameters and model versions |
| Shows | Clusters with size, dominant category and zone, time span, up to four representative thumbnails, members list |
| Actions | Mark two members as duplicates (FR-OFF-05), open a member in the review board |
| Use | Surfaces duplicates that slipped through AI-06 and patterns such as many earbud cases lost at one sports hall |

### AI-12 AI and Matching Monitoring

| Group | Metric | Definition |
|---|---|---|
| Health | Embedding coverage | share of open reports with every embedding for the active versions |
| | Job failures | failed `ai.*` and `match.*` jobs in the period |
| | Match latency | p50 and p95 from report creation to stored candidates |
| | Inference latency | p95 per `ml` endpoint |
| Score shape | Top 1 score distribution | histogram of the best `S` of each new report |
| | Tier share | share of new reports with at least one high, at least one medium |
| | Rank gap | median of top 1 minus top 2 score |
| Outcomes | Online rank of the true match | for claims handed over in the period, the rank the confirmed match held on the lost report when the claim was made. Reported as Recall@1, Recall@5 and Recall@10 |
| | Return path | share of returns that started from a candidate, an officer invitation, the public listing, or a proxy report |
| | Tier precision | per tier, confirmed divided by confirmed plus rejected plus officer dismissed |
| | Owner dismiss rate | per tier |
| Assistant | Acceptance | per field, accepted divided by suggested |
| Drift | Score PSI | population stability index of the top 1 score distribution against the previous 4 weeks. Warning banner above 0.2 |
| | Category mix | share per parent category compared with the previous 4 weeks |
| Versions | Active versions | per component, with activation history and evaluation metrics |

Cells under 3 outcomes are suppressed. Every metric can be filtered by model version so a
newly activated version is compared with the one before.

#### Sprint 4 Data Foundations Available

The database foundation for AI-12 monitoring is prepared through migration 00012. The following views are available for Sprint 4 UI endpoints:

- `ai_monitoring_assist_outcomes`: aggregates assistant field suggestions and outcomes (accepted, edited, ignored, guessed) across lost and found reports.
- `ai_monitoring_dismiss_rates`: computes total matches, owner dismissals, officer dismissals, and dismiss rates per tier.
- `ai_monitoring_confirmed_ranks`: computes confirmed match counts, rank distribution (rank 1, rank <= 5, rank <= 10), and online Recall@1, Recall@5, and Recall@10.
- `ai_monitoring_top1_histogram`: computes counts of top 1 match scores across fixed score buckets (< 0.40, 0.40 to 0.54, 0.55 to 0.71, 0.72 to 0.84, >= 0.85).
- `ai_monitoring_summary`: aggregates all four metric groups in a single unified JSON query row for dashboard consumers.

In addition, Prometheus runtime metrics in `api` (`match_top1_score`, `match_pipeline_seconds`, `match_candidates_stored`, `ml_request_duration_seconds`) and `ml` (`ml_models_loaded`, `ml_inference_seconds`, `ml_request_duration_seconds`) feed operational dashboards.

## Officer and administrator

### AI-13 Semantic Search

| | |
|---|---|
| Where | The search bar in the officer workspace, switch "Cari berdasarkan makna" |
| Logic | The query is embedded with `e5` (`query: ` prefix) and compared with report `e5` vectors inside the active filters. With the switch on and a keyword query present, the keyword ranking and the vector ranking are merged by reciprocal rank fusion with `k = 60` |
| Shows | The usual result list with a "Relevansi" column |
| Fallback | Keyword search only, switch disabled with a tooltip |
| Measured by | Recall@5 on the paraphrase queries of the evaluation fixture |
