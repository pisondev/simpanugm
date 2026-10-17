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

## Deferred until review

`zones.csv`, `locations.csv`, `posts.yaml`, and `model_versions.yaml` are intentionally
not created in this stage. Run `data/scripts/osm_locations.py` in download mode to save
the raw response and review CSV atomically; the extractor requests named campus-relevant
features only. A human must assign zones, aliases, and an approved status before the
rows can become `locations.csv`:

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
before starting the manual review. It rejects changed source fields and any premature
zone, alias, or approval values.

The promotion command requires the saved raw Overpass response and rechecks every source
field before applying human-reviewed status, aliases, and zones. It skips explicitly
rejected rows and refuses any row still marked `needs_review`, outside the bbox, or
assigned to an unknown zone. The promoter also
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

Run the foundation check from the repository root:

```text
python3 data/scripts/check.py
```

Run the future full-seed check with the agreed campus bbox from the location review:

```text
python3 data/scripts/check.py \
  --require-complete \
  --bbox=SOUTH,WEST,NORTH,EAST
```
