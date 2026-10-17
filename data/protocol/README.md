# Sprint 0 data protocol

This directory contains the preparation material for Axelle's 20-item dataset trial.
It contains no participant data, photographs, or identity documents.

## Files

- [`consent-form.md`](consent-form.md): internal consent-form wording to transfer into
  the team's approved form tool.
- [`mock-identity-card.md`](mock-identity-card.md): production rules for fictitious
  identity-card props.
- [`trial-20-items.md`](trial-20-items.md): collection and review checklist.
- [`trial-intake-template.csv`](trial-intake-template.csv): blank label template for
  `I001` through `I020`.

Validate a completed label sheet from the repository root:

```text
python3 data/scripts/check_trial.py --input /path/to/labels.csv
```

The template is intentionally incomplete and must not be treated as a dataset. The
validator only checks label structure and allowed values; it does not replace the
manual consent, photo, description, privacy, and quality review.

After the trial files are collected outside the repository, prepare them into a private
dataset directory:

```text
python3 data/scripts/intake.py \
  --source /private/simpanugm-eval/raw \
  --output /private/simpanugm-eval/v1 \
  --version trial-1
python3 data/scripts/check_manifest.py \
  --dataset-dir /private/simpanugm-eval/v1
```

The intake requires Pillow from `data/scripts/requirements.txt`. It refuses to write into
a non-empty destination and never deletes an existing dataset.
