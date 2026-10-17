# simpanugm-eval

Dataset card for the evaluation dataset. The protocol is
[docs/ai/dataset-collection.md](../../docs/ai/dataset-collection.md). Only this file and
`manifest.json` are in git. The images and texts live in the team Drive folder and are
fetched with `make eval-data`.

| | |
|---|---|
| Version | v1, in collection |
| Freeze | 15 November 2026 |
| Manifest hash | published here at freeze |
| Owner | Axelle Chandra |
| Consent | team members and friends who filled the consent form, mock identity cards only |
| Intended use | offline evaluation of SimpanUGM matching and report assistance, and the course report |
| Not for | training models that are distributed, any use outside the team |

The Sprint 0 trial preparation is kept in [`data/protocol/`](../protocol/). Completed
consent forms, photographs, descriptions, and review logs stay in the private team Drive
and are never committed to git.

The intake pipeline writes `manifest.json` with each prepared file's relative path, byte
size, SHA-256, and kind. It strips EXIF metadata and normalizes accepted image names
before hashing. Run `data/scripts/check_manifest.py` after intake and before a dataset
freeze.

The quality gate also verifies that each item has at least two finder photos, that the
owner-photo label agrees with the presence of an owner photo, and that descriptions are
not duplicated across items. It is run on the private prepared directory, not on git.

After labels pass the quality gate, create and verify the grouped split:

```text
python3 data/scripts/split_eval.py \
  --labels /private/simpanugm-eval/v1/labels.csv \
  --output /private/simpanugm-eval/v1/splits.csv \
  --distractors /private/simpanugm-eval/v1/distractors.csv \
  --distractor-output /private/simpanugm-eval/v1/distractor-splits.csv \
  --val-count 60 \
  --seed 2026
python3 data/scripts/split_eval.py \
  --labels /private/simpanugm-eval/v1/labels.csv \
  --val-count 60 \
  --verify /private/simpanugm-eval/v1/splits.csv \
  --distractors /private/simpanugm-eval/v1/distractors.csv \
  --verify-distractors /private/simpanugm-eval/v1/distractor-splits.csv
```

The assignments are deterministic, keep every physical item in one split only, and
place 160 of the 400 distractors in `val` for the v1 target. The generated CSV files
are private dataset metadata and must not be committed.

The full collection also requires the following private metadata contracts:

| File | Required columns | Target |
|---|---|---|
| `distractors.csv` | `distractor_id`, `leaf_category`, `primary_colour`, `distinctiveness`, `description` | `D001` through `D400`, with at least one finder photo each |
| `duplicates.csv` | `pair_id`, `item_id`, `first_report_id`, `second_report_id` | `P001` through `P040`, one pair per physical item |
| `queries.csv` | `query_id`, `query_text`, `target_report_id` | `Q001` through `Q050`, unique normalized query text |

Run `data/scripts/check_auxiliary.py` after the manifest and base quality checks. These
files remain private and are never committed to git.

Use `data/scripts/create_eval_templates.py` to create the blank raw collection layout
before data entry. It produces all five metadata files with stable IDs and refuses to
overwrite an existing non-empty directory.

## Contents at freeze

| Part | Target |
|---|---|
| Items | 150 |
| Finder photos | 2 to 3 per item |
| Owner photos | about 60 items |
| Distractor found reports | 400 |
| Duplicate pairs | 40 |
| Paraphrase queries | 50 |

## Version history

| Version | Date | Change |
|---|---|---|
| v1 | in progress | initial collection |
