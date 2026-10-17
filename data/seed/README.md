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
not created in this stage. Run `data/scripts/osm_locations.py` against a saved Overpass
JSON response to create a review CSV; the extractor requests named campus-relevant
features only. A human must assign zones, aliases, and an approved status before the
rows can become `locations.csv`:

```text
python3 data/scripts/promote_locations.py \
  --input /private/simpanugm-locations/locations-review.csv \
  --zones data/seed/zones.csv \
  --output /private/simpanugm-locations/locations.csv \
  --bbox SOUTH,WEST,NORTH,EAST
```

The promotion command skips explicitly rejected rows and refuses any row still marked
`needs_review`, outside the bbox, or assigned to an unknown zone. Posts need the
security-unit pilot list, and model versions must match the inference service lock file.
It also rejects non-finite bbox values, missing OSM provenance, and duplicate OSM elements;
any review error prevents the output seed from being written.
The seed checker supports an explicit complete-mode check once those files are available;
that mode validates file schemas, cross-file references, weekday opening hours, and the
one-active-version-per-component rule instead of checking file presence only.

The complete handoff sequence and evidence checklist is in
[`data/protocol/seed-review-checklist.md`](../protocol/seed-review-checklist.md).

Run the foundation check from the repository root:

```text
python3 data/scripts/check.py
```

Run the future full-seed check with:

```text
python3 data/scripts/check.py --require-complete
```
