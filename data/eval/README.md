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
