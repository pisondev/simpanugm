# Evaluation Protocol | SimpanUGM

| | |
|---|---|
| Owner | Irfan Fadilah Rafif |
| Data owner | Axelle Chandra |
| Harness | `ml/eval/`, run with `make eval` from Sprint 2 |
| Final report | `docs/ai/evaluation-report.md`, written in Sprint 4 from a tagged run |

A number is a result only if it can be reproduced from the repository, the dataset
version and the seed. This page fixes the task, the data, the metrics, the procedure and
the acceptance gates before any model is run, so the gates cannot be moved after seeing
the scores.

## 1. Tasks

| Task | Question | Primary metric |
|---|---|---|
| T1 Matching | Given a lost report, is its true found report near the top of the ranked list? | Recall@5, MRR |
| T2 Cross modal matching | Same as T1 for lost reports without a photo | Recall@10 |
| T3 Tier calibration | Are high tier candidates almost always right? | Precision of the high tier |
| T4 Category suggestion | Is the true leaf category among the top 3 suggestions for a found photo? | Top 3 accuracy |
| T5 Colour suggestion | Does a suggested colour match the labelled primary colour? | Agreement |
| T6 Brand suggestion | Are suggested brands correct? | Precision, recall |
| T7 Duplicate detection | Are two found reports of the same item flagged, and others not? | Precision and recall at the flag threshold |
| T8 Semantic search | Does a paraphrased query find its report? | Recall@5 |
| T9 Latency | Does the pipeline meet NFR-PERF-03 and NFR-PERF-04 on the reference server? | p95 |

## 2. Dataset

`simpanugm-eval`, built as described in [dataset-collection.md](dataset-collection.md).

| Part | Size in v1 | Content |
|---|---|---|
| Items | 150 physical items | Each with leaf category, primary colour, brand if visible, a distinctiveness label `generic` or `distinctive` |
| Finder photos | 2 to 3 per item | Taken where the item was "found" on campus, by a member who does not own it |
| Owner photos | 0 or 1 per item, about 40 percent have one | Taken at a different time and place, as an owner's phone photo would be |
| Owner descriptions | 1 per item | Written by the owner from memory, without looking at the item |
| Finder descriptions | 1 per item | Written by the finder while looking at it |
| Distractors | 400 found reports | Items with no lost counterpart, same collection protocol, 160 in `val` and 240 in `test` |
| Duplicate pairs | 40 | Items reported twice as found by two finders |
| Paraphrase queries | 50 | Free phrases an officer might type, each with its target report |

Report metadata (places and times) is generated per item by the scenario generator in
section 3, so the photos and texts are real while the metadata is controlled and
reproducible.

The dataset version is the SHA-256 of its manifest file. Every run records it.

## 3. Scenario generator

For each item the generator draws, with a fixed seed:

| Variable | Draw |
|---|---|
| True place `p` | uniform over campus locations, weighted by the expected foot traffic per location kind |
| True loss time `t` | uniform over weekday 07:00 to 21:00 WIB within a 10 week window |
| Owner's places | with probability 0.70 `p` alone, 0.15 `p` plus one or two nearby locations, 0.10 a nearby location without `p`, 0.05 unknown |
| Owner's window | `[t - U(0, 3 h), t + U(0, 2 h)]`, with probability 0.10 shifted by one day to simulate a wrong day |
| Found time | `t + Exponential(mean 6 h)` |
| Found place | `p` with probability 0.85, else a location within 300 m |
| Owner category | the true leaf with probability 0.85, else a sibling leaf or a listed compatible category |
| Distractor metadata | the same distributions, independent of any item |

The generator's parameters are part of the dataset manifest, so changing them is a new
dataset version.

## 4. Splits

| Split | Items | Use |
|---|---|---|
| `val` | 60 | Calibrating bands, weights and thresholds. Training learned fusion with grouped cross validation |
| `test` | 90 | Final numbers. Touched only at an acceptance gate |

Split by physical item, stratified by parent category and distinctiveness. Distractors are
split in the same proportion. Matching for an item in a split searches only among found
reports of the same split, so the candidate pool size is fixed per split and recorded.

A test in `ml/tests/test_split.py` asserts that no item id, image hash or description hash
appears in two splits (NFR-REPRO-04).

## 5. Metrics

| Metric | Definition |
|---|---|
| Recall@K | share of lost reports whose true found report is ranked within the top K |
| MRR | mean of 1 divided by the rank of the true found report, 0 when not retrieved in the top 50 |
| nDCG@10 | with one relevant item, equal to 1 divided by log2 of rank plus 1 within the top 10 |
| Tier precision | among candidates in a tier, the share that are true pairs |
| Tier recall | among true pairs, the share that land in that tier or above |
| ECE | expected calibration error over 10 equal width bins, reported for learned fusion only |

Every metric is reported overall and per slice:

| Slice | Values |
|---|---|
| Lost photo | with, without |
| Parent category | each parent with at least 10 items |
| Distinctiveness | generic, distinctive |
| Metadata noise | clean, wrong day, wrong place, wrong category |

Confidence intervals: 95 percent bootstrap over items, 1000 resamples, seed fixed.

## 6. Calibration procedure on `val`

1. **Bands.** For each band, compute cosines of true pairs and of retrieved negatives.
   `lo` is the median of negatives, `hi` the 95th percentile of positives, rounded to two
   decimals.
2. **Weights.** Grid search over weights in steps of 0.05 with `w_vis`, `w_text` each in
   0.20 to 0.50 and `w_loc`, `w_time` each in 0.05 to 0.25, summing to 1, maximising MRR.
   Ties are broken toward the simplest configuration, the one closest to equal weights.
   The cross modal visual weight is searched separately on the subset without lost photos.
3. **Thresholds.** `high` is the smallest score at which tier precision on `val` reaches
   0.85. `medium` is the largest score at which Recall@5, counting only candidates at or
   above `medium`, stays at least 0.80. The floor stays 0.40 unless more than 5 percent of
   true pairs fall below it.
4. **Freeze.** The result becomes a fusion configuration document with a new version id
   and is committed under `ml/eval/configs/`.

## 7. Baselines and ablations

Each is run on the same split with the same pool so differences come from the method only.

| Run | What it shows |
|---|---|
| B0 Random order within the pool | the floor |
| B1 Category, time and location only, no learned model | how far rules alone go, the "no AI" system |
| B2 TF-IDF cosine on descriptions plus B1 | a classical text baseline |
| B3 Full engine, calibrated weighted mean | the v1.0 candidate |
| B4 Learned fusion | the Sprint 4 candidate |
| A1 B3 without visual | value of photos |
| A2 B3 without text | value of descriptions |
| A3 B3 without location, A4 without time | value of metadata |
| A5 B3 with a hard category filter instead of the factor | why the factor is soft |
| A6 B3 with location scale 200, 800 m and half life 24, 168 h | sensitivity of the metadata scores |
| A7 B3 without synonym expansion | value of the slang table |

## 8. Acceptance gates

A fusion or model version may be activated in production only if, on `test`:

| Gate | Threshold |
|---|---|
| Recall@5 | at least 0.80 |
| Recall@1 | at least 0.55 |
| MRR | at least 0.65 |
| High tier precision | at least 0.85 |
| Recall@10 without lost photo | at least 0.60 |
| No slice except `generic` | Recall@5 at least 0.60 |
| Versus the active version | no gate metric worse by more than 0.02 |
| Component tasks | T4 top 3 at least 0.85, T5 at least 0.75, T6 precision at least 0.90 at recall at least 0.50, T7 precision at least 0.80, T8 Recall@5 at least 0.80 |
| Latency on the reference server | T9 within NFR-PERF-03 and NFR-PERF-04 |

When a gate fails, the run is still recorded and the evaluation report states which gate
failed and what was done. A gate is never lowered within the semester.

## 9. Error analysis

After each gate run, the twenty lowest ranked true pairs are reviewed by two members and
tagged with one or more causes: `photo_quality`, `item_small_in_frame`, `generic_item`,
`text_vocabulary`, `metadata_noise`, `category_disagreement`, `other`. The tag counts and
three illustrated examples go into the evaluation report, with the change each finding
leads to.

## 10. Reproducibility

| Requirement | How |
|---|---|
| Seeds | `random`, `numpy` and `torch` seeded with 2026, and `torch.use_deterministic_algorithms(True)` |
| Model revisions | read from `ml/models.lock.json`, recorded in the run |
| Dataset | manifest hash recorded, run refuses to start if the local files do not match the manifest |
| Code | git commit recorded, run refuses to start on a dirty tree unless `--allow-dirty`, which is then recorded |
| Output | `ml/eval/runs/<run-id>/` with `config.yaml`, `metrics.json`, `per_query.csv`, `slices.csv`, `environment.json`. Not committed. The run id and its metrics are copied into the evaluation report |
| Tracking | each run logged to MLflow with the same artefacts |

## 11. Online evaluation after launch

The offline set is small and staged. Once real reports exist, AI-12 tracks the online
counterparts: rank of confirmed matches, tier precision from claim outcomes and officer
dismissals, and drift. Every confirmed or rejected pair is a new label. When at least 100
confirmed returns exist, they become `simpanugm-online-v1` for the next calibration, with
the same grouping and gates.

## 12. Tests for the harness

- Metric functions against hand computed toy rankings.
- The split leakage test of section 4.
- Determinism: two runs on a 10 item fixture produce identical `metrics.json`.
- The scenario generator with a fixed seed produces the manifest hash recorded in the
  fixture.

## 13. Results (Synthetic Evaluation | SINTETIS)

> [!WARNING]
> SINTETIS: The results below were produced on synthetic evaluation data generated by
> `ml/tests/synth.py` (10 items, 6 distractors) using the deterministic stand in backend.
> These are not official results and do not represent model performance on real campus
> reports. Real evaluation results remain TERTUNDA until the dataset freeze on 15 November 2026.

### 13.1 Variant comparison on synthetic val split

| Variant | Description | Recall@1 | Recall@5 | Recall@10 | MRR | nDCG@10 |
|---|---|---|---|---|---|---|
| B0 | Random order floor | 0.100 | 0.800 | 0.900 | 0.387 | 0.507 |
| B1 | Category, time, and location only (no AI) | 0.900 | 1.000 | 1.000 | 0.950 | 0.963 |
| B2 | TF-IDF on descriptions plus B1 | 0.900 | 0.900 | 0.900 | 0.900 | 0.900 |
| B3 | Full engine, calibrated weighted mean | 0.400 | 0.400 | 0.400 | 0.400 | 0.400 |
| A1 | B3 without visual | 0.900 | 0.900 | 0.900 | 0.900 | 0.900 |
| A2 | B3 without text | 0.900 | 0.900 | 0.900 | 0.900 | 0.900 |
| A3 | B3 without location | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| A4 | B3 without time | 0.500 | 0.500 | 0.500 | 0.500 | 0.500 |
| A5 | B3 with hard category filter | 0.400 | 0.400 | 0.400 | 0.400 | 0.400 |
| A6a | B3 location scale 200 m | 0.400 | 0.400 | 0.400 | 0.400 | 0.400 |
| A6b | B3 location scale 800 m | 0.400 | 0.400 | 0.400 | 0.400 | 0.400 |
| A6c | B3 time half life 24 h | 0.400 | 0.400 | 0.400 | 0.400 | 0.400 |
| A6d | B3 time half life 168 h | 0.400 | 0.400 | 0.400 | 0.400 | 0.400 |
| A7 | B3 without synonym expansion | 0.400 | 0.400 | 0.400 | 0.400 | 0.400 |

### 13.2 Acceptance gate status (Synthetic Run)

| Gate | Threshold | Synthetic Value | Status | Note |
|---|---|---|---|---|
| Recall@5 | >= 0.80 | 0.400 | FAIL (SINTETIS) | Synthetic stand in embeddings have random projection; TERTUNDA for real data |
| Recall@1 | >= 0.55 | 0.400 | FAIL (SINTETIS) | TERTUNDA for real data |
| MRR | >= 0.65 | 0.400 | FAIL (SINTETIS) | TERTUNDA for real data |
| High tier precision | >= 0.85 | null | FAIL (SINTETIS) | No candidates met high tier score in synthetic run; TERTUNDA |
| Recall@10 without lost photo | >= 0.60 | 0.571 | FAIL (SINTETIS) | TERTUNDA for real data |
| No slice below Recall@5 0.60 | all >= 0.60 | distinctive=0.20, with_photo=0.00 | FAIL (SINTETIS) | Slices fail due to synthetic random features; TERTUNDA |
| T4 category top 3 | >= 0.85 | 0.000 | FAIL (SINTETIS) | Component prompts not evaluated against real zero shot weights; TERTUNDA |
| T5 colour | >= 0.75 | 0.000 | FAIL (SINTETIS) | SINTETIS; TERTUNDA |
| T6 brand precision | >= 0.90 | 0.000 | FAIL (SINTETIS) | SINTETIS; TERTUNDA |
| T6 brand recall | >= 0.50 | 0.000 | FAIL (SINTETIS) | SINTETIS; TERTUNDA |
| T7 duplicate precision | >= 0.80 | 0.000 | FAIL (SINTETIS) | SINTETIS; TERTUNDA |
| T8 semantic recall@5 | >= 0.80 | 0.000 | FAIL (SINTETIS) | SINTETIS; TERTUNDA |
| T9 CPU Latency | matching-engine sec 10 | p50 19 to 737 ms, p95 23 to 747 ms | PASS | Verified with `make eval-latency` on reference laptop |

### 13.3 Learned fusion experiment (B4 vs B3 on synthetic data | SINTETIS)

> [!NOTE]
> SINTETIS: Evaluated on synthetic fixture (`ml/eval/learned_fusion.py`) comparing the active
> calibrated linear weighted mean (B3) against experimental logistic regression fusion (B4)
> trained on 13 multimodal features using 5-fold GroupKFold cross-validation grouped by item.

| Metric | B3 (Linear Fusion) | B4 (Learned Fusion) | Delta (B4 - B3) | Gate Criterion | Gate Status |
|---|---|---|---|---|---|
| Recall@1 | 0.4000 | 0.9000 | +0.5000 | >= +0.02 | PASS (SINTETIS) |
| Recall@5 | 0.4000 | 1.0000 | +0.6000 | >= +0.02 | PASS (SINTETIS) |
| Recall@10 | 0.4000 | 1.0000 | +0.6000 | - | - |
| MRR | 0.4000 | 0.9500 | +0.5500 | >= +0.02 | PASS (SINTETIS) |
| nDCG@10 | 0.4000 | 0.9631 | +0.5631 | - | - |
| High tier precision | None | 1.0000 | - | >= 0.85 | PASS (SINTETIS) |
| ECE | N/A | 0.0372 | - | - | Well-calibrated (SINTETIS) |

#### Learned Logistic Regression Feature Weights (SINTETIS)
- `s_vis`: +0.0000
- `vis_missing`: -0.5750
- `mode_image`: +0.0000
- `mode_cross`: +0.0000
- `s_sem`: +0.0000
- `sem_missing`: -0.5750
- `s_attr`: +0.0000
- `attr_missing`: -0.5750
- `s_loc`: +0.8173
- `loc_missing`: +0.0000
- `s_time`: +0.8353
- `time_missing`: +0.0000
- `category_factor`: +0.8491
- `intercept`: -0.5750

#### Activation Recommendation
While the learned fusion model (B4) passes all synthetic acceptance gate thresholds on the
synthetic fixture, per the policy defined in `docs/ai/matching-engine.md` section 7.2 and
project constraints, learned fusion CANNOT be activated in production based on synthetic data.
The active production configuration remains `fusion-linear-2026.11.0.json`. Official activation
is TERTUNDA pending evaluation on the frozen real campus dataset (`simpanugm-eval` v1).

## 14. Real Evaluation Runbook (for Irfan and Axelle)

This one-page runbook guides Irfan and Axelle to execute official model evaluation once
the real dataset (`simpanugm-eval` v1) is collected and frozen on campus.

### 14.1 Prerequisites
1. Axelle completes private data collection and verifies checklist in [dataset-collection.md](dataset-collection.md) section 10.
2. Axelle runs freeze checks and exports the freeze report:
   ```bash
   python3 data/scripts/check_freeze.py \
     --dataset-dir /private/simpanugm-eval/v1 \
     --profile full \
     --split /private/simpanugm-eval/v1/splits.csv \
     --distractor-split /private/simpanugm-eval/v1/distractor-splits.csv \
     --val-count 60 \
     --output /private/simpanugm-eval/v1/freeze_report.json
   ```
3. Public manifest published to `data/eval/manifest.json`.
4. Irfan downloads evaluation data into `data/eval/` and copies `freeze_report.json`:
   ```bash
   python3 data/scripts/fetch.py --manifest data/eval/manifest.json --source /private/simpanugm-eval/v1 --output data/eval
   cp /private/simpanugm-eval/v1/freeze_report.json data/eval/freeze_report.json
   ```

### 14.2 Running Official Evaluation
From repository root, execute one command:
```bash
make eval-real
```

The runner (`ml/eval/run_real.py`):
1. Verifies `data/eval/freeze_report.json`. If missing, invalid, or failed, it exits immediately with error.
2. Calibrates component bands, weights, and tiers on the `val` split (60 items) using seed 2026.
3. Evaluates all variants (B0 to B3, A1 to A7) and acceptance gates on the `test` split (90 items).
4. Packages run artifacts under `ml/eval/runs/<run-id>/`:
   - `config.yaml`: Calibrated parameters and pool sizes.
   - `environment.json`: Git commit, seed, dataset version, model revisions.
   - `metrics.json`: Full metric outputs including CI95 and component gates.
   - `per_query.csv`: Query-level rank and true score logs.
   - `slices.csv`: Performance across subgroups.
   - `REPORT.md`: Authoritative markdown evaluation report labeled `RESMI` (or `SINTETIS` if run on non-frozen/synthetic data).

### 14.3 Post-Evaluation Decision
1. Check `ml/eval/runs/<run-id>/REPORT.md` Section 3 (Acceptance Gate Results).
2. If all gates pass with status `PASS (RESMI)`:
   - Commit calibrated config `ml/eval/configs/fusion-linear-<run-id>.json`.
   - Update `docs/ai/evaluation-report.md` with official numbers and run ID.
   - Proceed with candidate registration in admin UI.
3. If any gate fails:
   - Record failed gates and slice diagnostics in `docs/ai/evaluation-report.md`.
   - Do not activate the version in production.

