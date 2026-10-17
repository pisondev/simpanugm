# Matching Engine | SimpanUGM

| | |
|---|---|
| Owner | Irfan Fadilah Rafif |
| Reviewers | Pison Golda (integration), Axelle Chandra (data) |
| Fusion version described | `fusion-linear-2026.11.0`, the first production configuration |

The matching engine turns every lost and found report into vectors and attributes, finds
the reports of the opposite type that could be the same physical item, scores each pair
on five signals, fuses them into one score, and stores the ten best candidates per report
with an explanation. It is decision support: it orders the officer's and the owner's
attention, and it never decides ownership.

## 1. Design choices in one table

| Question | Answer | Why |
|---|---|---|
| Where does inference run | `ml`, a Python FastAPI service on the same server | The mature model ecosystem is Python. Go stays the owner of all state (ADR-0003) |
| Where do vectors live | PostgreSQL with pgvector, HNSW indexes | One database, transactional consistency with reports, filters in the same query (ADR-0004) |
| Where is the score computed | Go, in `api/internal/matching` | Deterministic, unit testable, versioned config, no network hop per pair |
| How are signals combined | Weighted mean over available signals, times a category factor | Explainable part by part, robust to missing signals, tunable without retraining (ADR-0008) |
| Owner has no photo | Multilingual CLIP text to image similarity | Most people have no photo of what they lost. This is the common case, not the edge case (ADR-0010) |
| Language | Multilingual encoders | Reports are Indonesian, often mixed with English brand and slang words |
| Hardware | CPU only | The budget is one VPS. Models are chosen to run in milliseconds to a few seconds on CPU |

## 2. Inputs

| From the report | Lost | Found |
|---|---|---|
| Category, leaf and parent | required | required |
| Item name | required | required |
| Description | required | required, public |
| Private identifying details or hidden details | optional | optional |
| Colours, brand | optional, from user or assistant | optional, from user or assistant |
| Photos | 0 to 5 | 1 to 5 |
| Places | 0 to 3 locations or one zone, or unknown | exactly one location |
| Time | window `lost_from` to `lost_until` | instant `found_at` |
| Visible text from photos | extracted, masked | extracted, masked |

Private and hidden details are used for matching inside the system. They are never shown
to the other party (BR-VIS-04).

## 3. Matching text

Text is normalised before embedding: Unicode NFKC, whitespace collapsed, URLs removed,
numbers masked per BR-PRV-03, and slang expanded with the synonym table
`data/seed/synonyms.csv` (for example `hp` adds `ponsel handphone`, `casan` adds `charger`,
`tumbler` adds `botol minum`). Expansion appends canonical terms and never deletes the
user's words.

| Embedding | Built from | Prefix | Max length |
|---|---|---|---|
| Sentence, `e5` | item name, leaf and parent category names, colours, brand, description, private or hidden details, visible text | `query: ` | 512 tokens |
| CLIP text, `clip_text` | item name, leaf category name, colours, brand, description | none | 128 tokens, truncated |

Example for a lost report:

```
query: dompet kulit. Dompet (Dompet dan Tas Kecil). Warna: cokelat. Merek: tidak ada.
Dompet lipat dua, isi KTM dan kartu ATM. Ada gantungan kunci kecil berbentuk kucing.
```

The CLIP text is shorter because that encoder was trained on caption length text and
loses precision on long input.

## 4. Models

| Component | Model | Output | Runtime on 4 vCPU |
|---|---|---|---|
| `image_encoder` | `sentence-transformers/clip-ViT-B-32` | 512 d, L2 normalised | about 120 ms per image |
| `xmodal_text_encoder` | `sentence-transformers/clip-ViT-B-32-multilingual-v1`, aligned to the image encoder's space, covers Indonesian | 512 d, L2 normalised | about 20 ms per text |
| `text_encoder` | `intfloat/multilingual-e5-small` | 384 d, L2 normalised | about 25 ms per text |
| `ocr` | EasyOCR with the `id` and `en` recognisers, input downscaled to 1024 px | text with boxes | 1 to 3 s per image |
| `category_classifier` | Zero shot with `image_encoder` against English prompts per category | top 3 slugs with scores | under 5 ms after the image embedding |
| `colour_extractor` | k-means in CIELAB, rule based naming | up to 2 colour names | under 30 ms |
| `brand_matcher` | Fuzzy match of OCR tokens against `data/seed/brands.csv` with rapidfuzz | brand candidates | under 5 ms |

Weights are pinned to an exact Hugging Face revision hash in `ml/models.lock.json` and
downloaded once into a volume (NFR-REPRO-02). The lock file is the only place a model
revision is named, and every response from `ml` echoes it.

**Zero shot category.** Each leaf category carries English prompts in master data, for
example for `payung`: "a photo of an umbrella", "a folded umbrella", "a compact umbrella".
The score of a category is the softmax, with temperature 0.01, over the maximum cosine of
its prompts. Prompt embeddings are cached in `ml` by the hash of the prompt set the `api`
sends, so a master data change is picked up without a restart.

**Colour.** The image is resized to 128 px, the central 70 percent kept, pixels clustered
into 4 groups in CIELAB. Groups under 10 percent share are dropped, each remaining group
is named by the nearest palette colour under CIEDE2000, and the two largest distinct
names are returned. The palette, defined in `data/seed/colours.csv`, has 14 entries:
hitam, putih, abu-abu, perak, emas, merah, merah muda, oranye, kuning, krem, cokelat,
hijau, biru, ungu.

**Brand.** OCR tokens of at least 3 characters are compared with the brand list using
`rapidfuzz.fuzz.ratio`. A ratio of at least 90 is a candidate. The list holds brands
common on campus: electronics, bags, bottles, apparel, vehicles, watches.

## 5. Candidate retrieval

Retrieval decides which pairs get scored. Scoring every pair would be correct but grows
with the square of open reports, so retrieval narrows the pool to at most about 200 per
report while keeping recall high.

### 5.1 Pool for a lost report

Found reports that are not hidden, with status `reported`, `potential_match`,
`under_verification` or `unclaimed`, and with
`lost_from - 24 h <= found_at <= lost_until + 60 days`.

### 5.2 Pool for a found report

Lost reports that are not hidden, with status `reported`, `potential_match` or
`under_verification`, and the same time condition seen from the other side.

### 5.3 Candidate generators

The union of four generators, deduplicated:

| Generator | Query | Against | k |
|---|---|---|---|
| Image to image | each image embedding of the source | image embeddings of the pool | 50 |
| Cross modal | the source's `clip_text`, or for a found source each image embedding | the pool's image embeddings, or the pool's `clip_text` | 50 |
| Text | the source's `e5` | the pool's `e5` | 50 |
| Structured | same leaf category, ordered by time distance | the pool | 50 |

Vector generators use HNSW indexes with cosine distance and pgvector's iterative index
scan, so the pool filter is applied inside the index search rather than after it.

## 6. Component scores

All component scores are in `[0, 1]`. A raw cosine is mapped with

```
norm(x, lo, hi) = min(1, max(0, (x - lo) / (hi - lo)))
```

because each encoder's cosines live in its own band: unrelated CLIP image pairs already
sit near 0.5, CLIP text to image pairs rarely exceed 0.35, and e5 compresses almost
everything into 0.7 to 0.95. The `lo` and `hi` values below are the starting point and are
calibrated on the validation split in Sprint 2 (evaluation section 6).

| Band | `lo` | `hi` |
|---|---|---|
| `image`, CLIP image to image | 0.60 | 0.95 |
| `cross_modal`, CLIP text to image | 0.18 | 0.33 |
| `e5`, sentence to sentence | 0.78 | 0.93 |

### 6.1 Visual, `s_vis`

| Case | Mode | Score |
|---|---|---|
| Both reports have photos | `image` | max over all photo pairs of `norm(cos, image)` |
| Lost report has no photo | `cross_modal` | max over found photos of `norm(cos(lost clip_text, photo), cross_modal)` |
| Found report has no photo, possible only for legacy or merged data | `cross_modal` | max over lost photos of `norm(cos(found clip_text, photo), cross_modal)` |
| Neither has a photo | `none` | missing |

### 6.2 Text, `s_text`

```
s_sem  = norm(cos(lost e5, found e5), e5)
s_attr = mean of the available attribute scores
s_text = 0.75 * s_sem + 0.25 * s_attr     if any attribute score is available
s_text = s_sem                             otherwise
```

| Attribute | Available when | Score |
|---|---|---|
| Colour | both sides have at least one colour | 1 if any colour is shared, 0.5 if any pair is adjacent in the palette adjacency list, else 0 |
| Brand | both sides have a brand, from the user or the brand matcher | 1 if equal after case folding and trimming, else 0 |

Palette adjacency: hitam and abu-abu, abu-abu and perak, perak and putih, putih and krem,
krem and cokelat, cokelat and oranye, oranye and merah, merah and merah muda, oranye and
kuning, kuning and emas, emas and cokelat, biru and ungu, biru and hijau.

**Identity name rule.** When the found category has sensitivity `identity` and the found
report's visible text contains the lost reporter's full name, or the recorded visitor name
for a proxy report, with a token set ratio of at least 85 (the measure defined by
rapidfuzz's `token_set_ratio`, ported to Go in `api/internal/matching/fuzzy` and tested
against rapidfuzz outputs on a shared fixture), then
`s_text = max(s_text, 0.95)` and the match carries the flag `name_hit`. A lost KTM is the
clearest case: the card says who owns it.

### 6.3 Location, `s_loc`

```
d(l, f) = haversine distance in metres between location coordinates
s_loc   = max over lost places l of exp(-d(l, f) / 400)
```

At 0 m the score is 1, at 400 m about 0.37, at 1 km about 0.08. A lost place given only
as a zone uses the zone centroid and the score is capped at 0.6. A lost report with
"tidak tahu lokasinya" has no location signal, which is missing, not zero.

### 6.4 Time, `s_time`

```
if found_at < lost_from - 2 h:              s_time = 0
if lost_from - 2 h <= found_at <= lost_until: s_time = 1
if found_at > lost_until:                   s_time = 0.5 ^ ((found_at - lost_until) / 72 h)
```

Three days after the window the score is 0.5, a week after about 0.2. Found times more
than 24 hours before the window never reach scoring because the pool excludes them.

### 6.5 Category factor, `g`

| Relation of the two leaf categories | `g` |
|---|---|
| Same leaf | 1.00 |
| Listed compatible pair in master data | the pair's factor, for example `dompet` and `tas-kecil` 0.70 |
| Same parent | 0.85 |
| Anything else | 0.35 |

When a pair is both listed and in the same parent, the larger factor wins. The factor
multiplies rather than adds, because a wrong category should pull every other signal down
without zeroing it: finders and owners disagree on categories often enough that a hard
filter would lose real matches.

## 7. Fusion and tiers

```
A     = components available among {vis, text, loc, time}
w_vis = 0.35 in mode image, 0.25 in mode cross_modal
w_text = 0.35, w_loc = 0.15, w_time = 0.15

S = g * sum(w_i * s_i for i in A) / sum(w_i for i in A)
```

`S` is rounded to four decimals and stored with every component.

| Tier | Rule | Who sees it | What happens |
|---|---|---|---|
| `high` | `S >= 0.72` | owner, officer | owner notified (BR-NTF-01), officer queue |
| `medium` | `0.55 <= S < 0.72` | owner, officer | listed on the report, digest |
| `low` | `0.40 <= S < 0.55` | officer on request | review board only |
| not stored | `S < 0.40` | nobody | dropped |

For each report the ten highest `S` at or above the floor are kept. A match row is
`active` while it is in the top ten of either side. Rank on each side is stored, and is
empty when the pair is outside that side's top ten.

The tier thresholds are calibrated so that the high tier reaches a precision of at least
0.85 on the validation split (evaluation section 6), and the medium tier floor so that
Recall@5 within medium and high stays at least 0.80.

### 7.1 The fusion configuration

Every number in sections 5 to 7 is one versioned JSON document, stored in `model_versions`
with component `fusion` and loaded by `api` at start and on activation:

```json
{
  "id": "fusion-linear-2026.11.0",
  "kind": "weighted_mean",
  "weights": { "visual_image": 0.35, "visual_cross_modal": 0.25, "text": 0.35, "location": 0.15, "time": 0.15 },
  "bands": { "image": [0.60, 0.95], "cross_modal": [0.18, 0.33], "e5": [0.78, 0.93] },
  "text_mix": { "semantic": 0.75, "attributes": 0.25 },
  "colour_partial": 0.5,
  "location": { "scale_m": 400, "zone_cap": 0.6 },
  "time": { "tolerance_before_h": 2, "half_life_h": 72, "pool_before_h": 24, "pool_after_d": 60 },
  "category": { "same_parent": 0.85, "other": 0.35 },
  "identity_name_hit": { "min_ratio": 85, "text_floor": 0.95 },
  "tiers": { "high": 0.72, "medium": 0.55, "floor": 0.40 },
  "top_k": 10,
  "ann_k": 50
}
```

Changing any value means a new fusion version, an evaluation run, and an activation,
exactly like a model change. That keeps every stored score explainable by the version id
next to it.

### 7.2 Learned fusion, Sprint 4

The weighted mean is the baseline. In Sprint 4 a logistic regression is trained on the
evaluation set with features `s_vis`, a mode indicator, `s_sem`, `s_attr`, `s_loc`,
`s_time`, `g`, and one missing flag per component. Positives are true pairs. Negatives are
the hard negatives that retrieval returns for each lost report. Training uses grouped
cross validation by physical item. Its output probability replaces `S` with
`kind: "logistic"` and the coefficients in the same JSON document, evaluated in Go as a
sigmoid. It is activated only if, on the test split, it improves both Recall@5 and MRR by
at least 0.02 absolute and keeps high tier precision at or above 0.85. Otherwise the
weighted mean stays and the result is recorded in the evaluation report.

## 8. Duplicate detection, AI-06

| Pair | Gate | Score | Flag at |
|---|---|---|---|
| Found and found | same parent category, `abs(found_at difference) <= 72 h`, distance `<= 500 m`, both open | `S_dup = 0.6 * max norm(image cos) + 0.4 * s_sem`, or 1.0 when any perceptual hash pair differs by at most 6 bits | 0.80 |
| Lost and lost | same owner, same parent category, overlapping windows, both open | `s_sem` | 0.80, BR-LST-05 |

Found pairs above the flag are shown to the finder before submission and listed for
officers. Nothing is merged automatically.

## 9. Inference API

Internal only, bearer token `ML_INTERNAL_TOKEN`, JSON unless noted. The OpenAPI document is
generated by FastAPI and committed as `ml/openapi.json`, which is the contract.

| Method and path | Request | Response | `api` timeout |
|---|---|---|---|
| `GET /healthz` | none | `{"status":"ok"}` | 2 s |
| `GET /readyz` | none | 200 when every model is loaded, 503 while loading | 2 s |
| `GET /v1/models` | none | per component: model id, revision, dimension, loaded at | 2 s |
| `POST /v1/embed/image` | multipart, 1 to 8 JPEG or WebP files | `{model, revision, embeddings: [[512 floats]]}` | 10 s |
| `POST /v1/embed/text` | `{texts: [...], kinds: ["e5","clip_text"]}`, at most 32 texts | per kind `{model, revision, embeddings}` | 5 s |
| `POST /v1/analyze/image` | multipart, one file, plus `categories` JSON `[{slug, prompts}]` and its `prompt_set_hash` | `{categories: [{slug, score}], colours: [{name, hex, share}], text: {value, masked}, brands: [{name, score}], versions}` | 8 s |
| `POST /v1/cluster` | `{items: [{id, vector}], min_cluster_size, min_samples}` | `{labels: [{id, cluster, probability}], version}` | 30 s |

Errors use problem details with codes `invalid_image`, `too_large`, `too_many_items`,
`not_ready`. The service runs one Uvicorn worker with a bounded thread pool. The `api`
side limits concurrency through the River queue `ml`, which has at most 2 workers, so the
CPU is never oversubscribed.

## 10. Jobs and triggers

| Job | Queue | Trigger | Does | Idempotency |
|---|---|---|---|---|
| `ai.embed_image` | `ml` | photo attached to a report. Variants and hashes already exist, they are produced during the upload request | stores image embedding | upsert by image id and model version |
| `ai.analyze_image` | `ml` | photo attached to a report | stores attributes and masked text | upsert by image id and model version |
| `ai.embed_text` | `ml` | report created or text changed, and after every `ai.analyze_image` of the report | stores both text embeddings | upsert by report and model version, skipped if the text hash is unchanged |
| `match.report` | `default` | after `ai.embed_text`, or on demand | retrieval, scoring, upsert of matches, status effects | unique per report while pending, so bursts of edits coalesce |
| `match.notify` | `default` | after `match.report` produced a new high candidate | applies BR-NTF-01 and creates notifications | by match id and throttle window |
| `match.sweep` | `default` | daily 02:00 WIB, and on model activation | enqueues `match.report` for every open report | periodic |
| `ai.backfill` | `ml` | model activation | re-embeds every open report and image with the new version, then a sweep | resumable by cursor |

`match.report` waits for the report's embeddings: when an embedding job is still pending
it snoozes for 10 s, up to 6 times, then scores with what exists and records the missing
components. The budget behind NFR-PERF-03, for a report with two photos:

| Step | Typical |
|---|---|
| Upload processing | 1 to 2 s |
| Image embedding, 2 photos | under 1 s |
| Image analysis with OCR, 2 photos | 3 to 6 s |
| Text embedding | under 0.1 s |
| Retrieval and scoring of about 200 pairs | under 0.5 s |
| Queue waits | 1 to 10 s |

## 11. Explanations, AI-02

Each component has a reason line generated from its value with fixed templates. Owners see
the reason lines and the bars. Officers also see the numbers, the mode, the versions and
the `name_hit` flag.

| Component | Condition | Reason line |
|---|---|---|
| Visual, `image` | `>= 0.8`, `>= 0.5`, else | Foto sangat mirip. Foto cukup mirip. Foto kurang mirip. |
| Visual, `cross_modal` | `>= 0.8`, `>= 0.5`, else | Deskripsimu sangat cocok dengan foto barang temuan. Cukup cocok. Kurang cocok. |
| Text | `>= 0.8`, `>= 0.5`, else, plus chips | Deskripsi sangat mirip. Cukup mirip. Berbeda. Chips: Warna sama: hitam. Merek sama: Eiger. |
| Text, `name_hit` | flag set | Nama pada barang sesuai dengan namamu. |
| Location | under 50 m, up to 1500 m, beyond, missing | Ditemukan di lokasi yang sama. Ditemukan sekitar {d} m dari {place}. Lokasi berjauhan. Lokasi kehilangan tidak diketahui. |
| Time | in window, after, before | Ditemukan dalam rentang waktu kehilangan. Ditemukan {n} jam atau {n} hari setelah perkiraan waktu hilang. Ditemukan sebelum perkiraan waktu hilang. |
| Category | same leaf, else | Kategori sama. Kategori berbeda: {a} dan {b}. |

Distances are rounded to 50 m. Hours are used under 48 hours, days beyond.

## 12. Failure modes

| Failure | Effect | Mitigation |
|---|---|---|
| The item is small in a busy photo | Image embedding describes the background | Camera hint in the form "Foto barang dari dekat, isi sebagian besar layar". Centre crop in colour extraction. Object cropping is a v1.1 candidate |
| Generic items: black umbrellas, clear bottles | Many near identical visual scores | Location and time carry the ranking. Verification questions decide. Monitoring tracks the score gap between rank 1 and rank 2 |
| Owner remembers time or place wrongly | Lower time or location score | Soft scores, a 2 hour tolerance, a 24 hour pool margin, up to three places |
| Slang and mixed language | Lower text similarity | Synonym expansion, multilingual encoders |
| Inference service down | No new embeddings | Reports still save, jobs retry, sweep catches up (NFR-AVAIL-02) |
| Model update changes score bands | Thresholds no longer mean the same | New fusion version calibrated with the model, backfill before activation, rollback in one click |
| A person writes vague lost reports to see found photos | Exposure of found item appearance | Only matched candidates of medium or high tier are visible, at most ten per report, candidate photo views are logged and rate limited, ownership is proven with hidden details and ID in person, BR-CLM-04 |
| Seasonal drift, for example the rainy season and umbrellas | Score distribution shifts | PSI on the top 1 score distribution in AI-12, weekly review |

## 13. What this page does not cover

How the numbers are measured and accepted: [evaluation.md](evaluation.md). How a model or
fusion version moves from experiment to production: [model-lifecycle.md](model-lifecycle.md).
Per role features built on this engine: [ai-features.md](ai-features.md).
