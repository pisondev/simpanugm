# Development seed overlay

`zones.csv`, `locations.csv` and `posts.yaml` for local development, end to end tests and
the staging demo, until the reviewed files exist in [`data/seed/`](../../data/seed/).

| File | Content |
|---|---|
| `zones.csv` | the five campus zones, centroids computed from the locations below |
| `locations.csv` | 41 campus landmarks with coordinates from OpenStreetMap, extracted 17 October 2026 |
| `posts.yaml` | six pilot posts, one per zone plus the central library, with placeholder hours and no phone numbers |

These rows have not been through the location review of
[`data/protocol/seed-review-checklist.md`](../../data/protocol/seed-review-checklist.md).
They pass the schema checks of `data/scripts/check.py`, which is all a demo needs.

## How it is used

`simpan seed --dir data/seed --overlay deploy/dev-seed` reads every file from `data/seed`
first and takes a file from the overlay only when `data/seed` does not have it. As soon as
the reviewed `locations.csv` and friends are promoted into `data/seed`, the overlay copies
are ignored without any change here.

`simpan seed` refuses `--overlay` when `APP_ENV=production`, so production can only ever
run on reviewed master data.
