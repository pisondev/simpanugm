# Seed foundation

This directory contains the deterministic master data that can be prepared before the
campus review and dataset collection.

## Ready in this stage

- category tree and verification questions;
- fourteen-colour palette;
- category compatibility factors;
- initial brand aliases;
- Indonesian campus-report synonyms;
- runtime setting defaults from the business rules.

## MVP baseline and production review

The four formerly deferred files now contain a demo-aligned public-source baseline. It is
reviewed for demo, production review pending:

- `zones.csv` contains five project-level groups and centroids calculated from the selected
  location rows.
- `locations.csv` contains the 25 baseline rows and the 41 OSM landmarks from the demo
  overlay. The overlay rows retain their demo slugs and aliases inside
  `-7.776005,110.3716393,-7.764283,110.3863`.
- `posts.yaml` contains the central K5L office and the six demo pilot posts. The K5L office
  keeps the emergency number published on [UGM's contact page](https://ugm.ac.id/id/kontak/);
  demo pilot phone values remain empty until official contacts are reviewed.
- `model_versions.yaml` records the locked model identities used by the current AI
  documentation. The locked entries match `ml/models.lock.json`. It is an MVP baseline, not a replacement for
  `ml/models.lock.json`.

The bbox covers the combined public-source and demo-overlay rows. The selected rows and the
generated review evidence were kept outside Git; only the derived seed is committed. For
production, replace or re-review all four files together using the raw Overpass response,
official zone and post inputs, and the inference-service lock file. The existing review and
promotion flow remains available:

```text
python3 data/scripts/promote_locations.py \
  --input /private/simpanugm-locations/locations-review.csv \
  --raw /private/simpanugm-locations/overpass.json \
  --zones data/seed/zones.csv \
  --output /private/simpanugm-locations/locations.csv \
  --bbox=SOUTH,WEST,NORTH,EAST
```

The download retries transient Overpass errors twice by default; use `--retries 0` for a
single request or `--timeout` to adjust the per-request limit.

Run `data/scripts/check_location_review.py` against the saved response and generated CSV
before starting the manual review. It rejects changed source fields, malformed CSV rows,
and any premature zone, alias, or approval values.

The promotion command requires the saved raw Overpass response and rechecks every source
field before applying reviewed status, aliases, and zones. It skips explicitly rejected
rows and refuses any row still marked `needs_review`, outside the bbox, or assigned to an
unknown zone. The promoter also
requires valid positive OSM provenance and at least one approved row, so a fully
rejected review cannot produce an empty location seed. Posts need the
security-unit pilot list, and model versions must match the inference service lock file.
It also rejects non-finite bbox values, missing OSM provenance, and duplicate OSM elements;
any review error prevents the output seed from being written. Review and promoted CSV
writes are atomic, so a failed write does not replace an existing output with partial data.
The seed checker supports an explicit complete-mode check once those files are available;
that mode validates file schemas, cross-file references, weekday opening hours, and the
one-active-version-per-component rule instead of checking file presence only.

The complete handoff sequence and evidence checklist is in
[`data/protocol/seed-review-checklist.md`](../protocol/seed-review-checklist.md).

The foundation check rejects a partial addition of any deferred file. Once the first
reviewed complete-seed file is available, run the complete check with all four files and
the agreed campus bbox; do not rely on foundation mode for an incomplete handoff. Seed
roots and files must be regular paths, not symlinks.

Run the complete MVP seed check from the repository root:

```text
python3 data/scripts/check.py \
  --require-complete \
  --bbox=-7.776005,110.3716393,-7.764283,110.3863
```

For a future production review, run the same complete check with the agreed campus bbox:

```text
python3 data/scripts/check.py \
  --require-complete \
  --bbox=SOUTH,WEST,NORTH,EAST
```
