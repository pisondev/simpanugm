# simpanugm-eval

Dataset card for the evaluation dataset. The protocol is
[docs/ai/dataset-collection.md](../../docs/ai/dataset-collection.md). Until the private
dataset is frozen, only this card is in git. `manifest.json` is committed at the v1
freeze with the real file hashes; images and texts live in the team Drive folder and are
fetched with `make eval-data`.

| | |
|---|---|
| Version | v1, in collection |
| Freeze | 15 November 2026 |
| Manifest hash | pending private intake and freeze |
| Owner | Axelle Chandra |
| Consent | team members and friends who filled the consent form, mock identity cards only |
| Intended use | offline evaluation of SimpanUGM matching and report assistance, and the course report |
| Not for | training models that are distributed, any use outside the team |

The Sprint 0 trial preparation is kept in [`data/protocol/`](../protocol/). Completed
consent forms, photographs, descriptions, and review logs stay in the private team Drive
and are never committed to git.

The intake pipeline writes `manifest.json` with each prepared file's relative path, byte
size, SHA-256, and kind. It strips EXIF metadata and normalizes accepted image names
before hashing. It stages the output first, so a failed intake does not leave a partial
dataset directory. The manifest checker requires normalized image paths, approved file
kinds, valid hashes, images at least 1080 px on the short side, unique image content,
and rejects symlinks, including symlink directories. Run
`data/scripts/check_manifest.py` after intake and before a dataset freeze.

After the private review and v1 freeze, publish only that validated manifest with
`data/scripts/publish_manifest.py`. The command copies no images, descriptions, or other
private files into the repository and refuses to replace an existing manifest unless
`--replace` is supplied explicitly.

The quality gate also verifies that each item has at least two finder photos, that the
owner-photo label agrees with the presence of an owner photo, and that normalized
descriptions are not duplicated across items or report roles. It is run on the private
prepared directory, not on git.

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
are private dataset metadata, are written atomically, and must not be committed.

The full collection also requires the following private metadata contracts:

| File | Required columns | Target |
|---|---|---|
| `distractors.csv` | `distractor_id`, `leaf_category`, `primary_colour`, `distinctiveness`, `description` | `D001` through `D400`, with at least one finder photo each |
| `duplicates.csv` | `pair_id`, `item_id`, `first_report_id`, `second_report_id` | `P001` through `P040`, one pair per physical item |
| `queries.csv` | `query_id`, `query_text`, `target_report_id` | `Q001` through `Q050`, unique normalized query text |

The base item and distractor IDs are stable, one-based IDs (`I001` through `I150` and
`D001` through `D400`). The splitter rejects IDs outside those formats before assigning
`val` and `test`.

Run `data/scripts/check_auxiliary.py` after the manifest and base quality checks. These
files remain private and are never committed to git. The auxiliary gate also rechecks
one-based IDs and label-ID uniqueness so it is safe to run independently.

Use `data/scripts/create_eval_templates.py` to create the blank raw collection layout
before data entry. It produces all five metadata files with stable IDs and refuses to
overwrite an existing non-empty directory. Generation is staged first, so an interrupted
or failed write does not leave a partial raw layout.

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
