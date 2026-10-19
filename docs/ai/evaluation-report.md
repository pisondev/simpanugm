# Model Evaluation Report | SimpanUGM

| | |
|---|---|
| Owner | Irfan Fadilah Rafif |
| Data owner | Axelle Chandra |
| Harness | `ml/eval/run_real.py`, executed via `make eval-real` |
| Status | **TERTUNDA** (pending private dataset freeze on 15 November 2026) |

> [!WARNING]
> **STATUS: TERTUNDA**
> Official model evaluation numbers on the real campus dataset (`simpanugm-eval` v1) remain
> TERTUNDA until data collection, review, and freeze checks are completed on 15 November 2026.
> Synthetic baseline and ablation benchmarks are documented in [evaluation.md](evaluation.md) section 13.

## 1. Locked Production Models

Models are pinned in `ml/models.lock.json` per [model-lifecycle.md](model-lifecycle.md):

| Component | Model ID | Revision | Dimension |
|---|---|---|---|
| `image_encoder` | `sentence-transformers/clip-ViT-B-32` | `327ab6726d33c0e22f920c83f2ff9e4bd38ca37f` | 512 |
| `xmodal_text_encoder` | `sentence-transformers/clip-ViT-B-32-multilingual-v1` | `58edf8cada9e398793dca955574a48cbb7f18be2` | 512 |
| `text_encoder` | `intfloat/multilingual-e5-small` | `614241f622f53c4eeff9890bdc4f31cfecc418b3` | 384 |
| `ocr` | `JaidedAI/EasyOCR` | `craft_mlt_25k+latin_g2` | Text boxes |
| `fusion` | `simpanugm-fusion` | `fusion-linear-2026.11.0.json` | Tier scores |

## 2. Acceptance Gate Status (Real Dataset `test` Split)

All gate thresholds are fixed before seeing the test scores and are never lowered within the semester:

| Gate | Target Threshold | Official Real Value | Gate Status | Note |
|---|---|---|---|---|
| Recall@5 | >= 0.80 | TERTUNDA | TERTUNDA | Pending real dataset freeze |
| Recall@1 | >= 0.55 | TERTUNDA | TERTUNDA | Pending real dataset freeze |
| MRR | >= 0.65 | TERTUNDA | TERTUNDA | Pending real dataset freeze |
| High tier precision | >= 0.85 | TERTUNDA | TERTUNDA | Pending real dataset freeze |
| Recall@10 without lost photo | >= 0.60 | TERTUNDA | TERTUNDA | Cross-modal text to photo matching |
| No slice below Recall@5 0.60 | all >= 0.60 | TERTUNDA | TERTUNDA | Except generic items |
| T4 category top 3 | >= 0.85 | TERTUNDA | TERTUNDA | Zero-shot visual prompt scoring |
| T5 colour agreement | >= 0.75 | TERTUNDA | TERTUNDA | Dominant pixel palette match |
| T6 brand precision | >= 0.90 | TERTUNDA | TERTUNDA | OCR and wordlist matching |
| T6 brand recall | >= 0.50 | TERTUNDA | TERTUNDA | Visible brand extraction |
| T7 duplicate precision | >= 0.80 | TERTUNDA | TERTUNDA | Duplicate found report detection |
| T8 semantic recall@5 | >= 0.80 | TERTUNDA | TERTUNDA | Natural language query retrieval |
| T9 CPU Latency | matching-engine sec 10 | Ref Laptop: PASS; VPS: TERTUNDA | PASS (Ref) / TERTUNDA (VPS) | Reference laptop verified; VPS pending staging |

## 3. Variant Comparison on Real Test Split

| Variant | Description | Recall@1 | Recall@5 | Recall@10 | MRR | nDCG@10 |
|---|---|---|---|---|---|---|
| B0 | Random order floor | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| B1 | Category, time, and location only (no AI) | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| B2 | TF-IDF on descriptions plus B1 | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| B3 | Full engine, calibrated weighted mean | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| B4 | Learned logistic regression fusion | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| A1 | B3 without visual | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| A2 | B3 without text | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| A3 | B3 without location | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| A4 | B3 without time | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| A5 | B3 with hard category filter | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| A6a | B3 location scale 200 m | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| A6b | B3 location scale 800 m | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| A6c | B3 time half life 24 h | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| A6d | B3 time half life 168 h | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |
| A7 | B3 without synonym expansion | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA | TERTUNDA |

## 4. Component CPU Latency Benchmarking (T9)

| Component | Input | Ref Laptop p50 | Ref Laptop p95 | VPS p50 | VPS p95 | Target Budget |
|---|---|---|---|---|---|---|
| `image_encoder` | 1 image | 72 ms | 78 ms | TERTUNDA | TERTUNDA | < 500 ms |
| `image_encoder` | 2 images | 117 ms | 129 ms | TERTUNDA | TERTUNDA | < 1000 ms |
| `xmodal_text_encoder` | 1 text | 24 ms | 28 ms | TERTUNDA | TERTUNDA | < 100 ms |
| `text_encoder` | 1 text | 19 ms | 23 ms | TERTUNDA | TERTUNDA | < 100 ms |
| `ocr` | mock card | 737 ms | 747 ms | TERTUNDA | TERTUNDA | < 3000 ms |
| `colour_extractor` | 1 image | 29 ms | 32 ms | TERTUNDA | TERTUNDA | < 100 ms |
| `brand_matcher` | 1 string | 0.1 ms | 0.1 ms | TERTUNDA | TERTUNDA | < 10 ms |
| `fusion` | 1 pair | < 1 ms | < 1 ms | TERTUNDA | TERTUNDA | < 5 ms |

## 5. Execution Instructions for Official Evaluation

Once the freeze is completed per [dataset-collection.md](dataset-collection.md) section 10:
```bash
make eval-real
```

This automated runner verifies `data/eval/freeze_report.json`, calibrates on `val`, evaluates on `test`,
and outputs `ml/eval/runs/<run-id>/REPORT.md` with status `RESMI`. The verified metrics will then be
transcribed into this document.
