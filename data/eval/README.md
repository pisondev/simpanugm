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
  --val-count 60 \
  --seed 2026
python3 data/scripts/split_eval.py \
  --labels /private/simpanugm-eval/v1/labels.csv \
  --val-count 60 \
  --verify /private/simpanugm-eval/v1/splits.csv
```

The assignment is deterministic and keeps every physical item in one split only. The
generated CSV is private dataset metadata and must not be committed.

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
