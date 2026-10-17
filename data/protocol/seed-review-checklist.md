# Complete seed handoff checklist

Use this checklist before adding the four deferred files to `data/seed/`. The files
must come from campus review or the inference-service lock file; do not fill them with
placeholders and do not include participant data.

## 1. Zones

- [ ] Confirm the five official slugs: `saintek`, `soshum`, `agro`, `medika`, `pusat`.
- [ ] Record the approved display names and representative centroids in `zones.csv`.
- [ ] Check that every coordinate is within the valid latitude/longitude ranges.
- [ ] Keep the source or review note outside git if it contains internal contact details.

## 2. Locations

- [ ] Save the reviewed Overpass response privately and generate a review CSV:

  ```text
  python3 data/scripts/osm_locations.py \
    --download \
    --bbox SOUTH,WEST,NORTH,EAST \
    --raw-output /private/simpanugm-locations/overpass.json \
    --output /private/simpanugm-locations/locations-review.csv
  ```

  When a raw response has already been saved, use:

  ```text
  python3 data/scripts/osm_locations.py \
    --input /private/simpanugm-locations/overpass.json \
    --output /private/simpanugm-locations/locations-review.csv
  ```

- [ ] Verify the generated CSV against the saved source before editing it:

  ```text
  python3 data/scripts/check_location_review.py \
    --raw /private/simpanugm-locations/overpass.json \
    --review /private/simpanugm-locations/locations-review.csv \
    --bbox SOUTH,WEST,NORTH,EAST
  ```

- [ ] For every promoted row, confirm the official name, student aliases, zone, kind,
  main-entrance coordinate, and `review_status=approved` by hand.
- [ ] Promote only the approved rows inside the agreed campus bounding box:

  ```text
  python3 data/scripts/promote_locations.py \
    --input /private/simpanugm-locations/locations-review.csv \
    --zones data/seed/zones.csv \
    --output /private/simpanugm-locations/locations.csv \
    --bbox SOUTH,WEST,NORTH,EAST
  ```

- [ ] Copy the reviewed output to `data/seed/locations.csv` only after a second-person
  spot check confirms names, aliases, zones, kinds, and coordinates.

## 3. Security posts

- [ ] Obtain the pilot list from the security unit, with one post per zone and one post
  at the central library unless the approved pilot list specifies a different layout.
- [ ] Tie every post to an approved `location_slug`.
- [ ] Confirm the phone string, all seven weekday keys, and each opening-hour interval.
- [ ] Remove personal phone numbers or names that are not part of the operational post
  contact before committing the seed.

## 4. Model versions

- [ ] Obtain the inference-service lock file and record its exact model identifiers,
  revisions, dimensions, and library versions in `model_versions.yaml`.
- [ ] Include all eight components: `image_encoder`, `xmodal_text_encoder`,
  `text_encoder`, `ocr`, `category_classifier`, `colour_extractor`, `brand_matcher`,
  and `fusion`.
- [ ] Set exactly one version per component to `active`; keep the fusion configuration
  and its hash with the seed entry.
- [ ] Record the lock-file revision or release identifier in the review record outside
  git so the seed can be reproduced.

## 5. Final gate

Run from the repository root after all four files are present:

```text
python3 data/scripts/check.py --require-complete
python3 -m unittest discover -s data/scripts -p 'test_*.py'
```

Do not merge the four files when either command fails. The private Overpass response,
security-unit source, inference lock file, and review notes remain outside git unless
the team explicitly approves a non-sensitive source artifact.
