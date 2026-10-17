# Sprint 0 data protocol

This directory contains the preparation material for Axelle's 20-item dataset trial.
It contains no participant data, photographs, or identity documents.

## Files

- [`consent-form.md`](consent-form.md): internal consent-form wording to transfer into
  the team's approved form tool.
- [`mock-identity-card.md`](mock-identity-card.md): production rules for fictitious
  identity-card props.
- [`trial-20-items.md`](trial-20-items.md): collection and review checklist.
- [`seed-review-checklist.md`](seed-review-checklist.md): handoff checklist for the
  deferred campus and inference seed files.
- [`trial-intake-template.csv`](trial-intake-template.csv): blank label template for
  `I001` through `I020`.
- [`trial-review-log-template.csv`](trial-review-log-template.csv): blank exit-review
  log for the same 20 trial items.

Validate a completed label sheet from the repository root:

```text
python3 data/scripts/check_trial.py --input /path/to/labels.csv
```

The template is intentionally incomplete and must not be treated as a dataset. The
validator only checks label structure and allowed values; it does not replace the
manual consent, photo, description, privacy, and quality review.

For the two-person label review, keep both private sheets separate and generate only an
aggregate agreement report:

```text
python3 data/scripts/check_agreement.py \
  --first /private/simpanugm-eval/trial-20/labels-a.csv \
  --second /private/simpanugm-eval/trial-20/labels-b.csv \
  --expected-count 20 \
  --output /private/simpanugm-eval/trial-20/agreement.md
```

The report contains Cohen's kappa for category and distinctiveness. Do not commit either
label sheet or the report when it is tied to private collection activity.

Create the private review workspace before collection; it contains two blank annotation
sheets and a review log with stable IDs:

```text
python3 data/scripts/create_trial_review_workspace.py \
  --output /private/simpanugm-eval/trial-20/review
```

The committed [`trial-review-log-template.csv`](trial-review-log-template.csv) documents
the review-log columns. The generated sheets record no participant details until the
private collection begins; keep all completed files and their Drive path outside git.

## Full collection templates

When the 20-item trial is accepted, create blank private templates for the full target:

```text
python3 data/scripts/create_eval_templates.py \
  --output /private/simpanugm-eval/raw-v1
```

This creates templates for 150 items, 400 distractors, 40 duplicate pairs, and 50
paraphrase queries, plus an `images/` directory. The generator refuses to write into a
non-empty directory and stages the output so a failed generation leaves no partial
templates. Fill the CSVs and photos privately, then run `intake.py`; none of these filled
files belong in git.

Verify a blank full template before data entry:

```text
python3 data/scripts/check_templates.py \
  --input /private/simpanugm-eval/raw-v1 \
  --profile full
```

To prepare the trial workspace itself, use the trial profile:

```text
python3 data/scripts/create_eval_templates.py \
  --profile trial \
  --output /private/simpanugm-eval/trial-20/raw
```

The trial profile creates only `items.csv`, `labels.csv`, and `images/`; it does not
create distractor, duplicate, or query files.

Verify the blank trial scaffold with:

```text
python3 data/scripts/check_templates.py \
  --input /private/simpanugm-eval/trial-20/raw \
  --profile trial
```

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
a non-empty destination and never deletes an existing dataset. Intake writes to a private
staging directory first, so an invalid image or filename leaves no partial output. Source
metadata and images must be regular files, not symlinks.

After the private dataset is approved and frozen, publish only its validated manifest to
the repository. This copies no images or descriptions:

```text
python3 data/scripts/publish_manifest.py \
  --dataset-dir /private/simpanugm-eval/v1 \
  --output data/eval/manifest.json
```

Commit `data/eval/manifest.json` together with the updated manifest hash in its dataset
card. Use `--replace` only when publishing a reviewed new dataset version.

Run the quality gate after manifest verification:

```text
python3 data/scripts/check_dataset.py \
  --dataset-dir /private/simpanugm-eval/v1 \
  --expected-count 20
```

It checks metadata completeness, image coverage, owner-photo labels, and duplicate
descriptions across items and report roles. It does not decide whether an object is safe
to photograph; that remains a manual consent and privacy review.
