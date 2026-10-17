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
JSON response to create a review CSV; a human must assign zones and aliases before the
rows can become `locations.csv`. Posts need the security-unit pilot list, and model
versions must match the inference service lock file. The seed checker supports an
explicit complete-mode check once those files are available.

Run the foundation check from the repository root:

```text
python3 data/scripts/check.py
```

Run the future full-seed check with:

```text
python3 data/scripts/check.py --require-complete
```
