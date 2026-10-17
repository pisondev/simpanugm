# Analytics reference views

This directory contains the SQL definitions Axelle reviews with the API owner before
they become migrations. The definitions follow the canonical materialized-view contract
in `docs/architecture/data-model.md` without including descriptions, private details,
reporter identities, or finder identities.

## Views

| View | Grain and purpose |
|---|---|
| `mv_reports_daily` | WIB day, report type, leaf category, place, and report count for hotspots |
| `mv_hour_weekday` | Report type, ISO weekday, hour, and report count for time patterns |
| `mv_return_times` | One found report with category, zone, custody timestamps, and return duration |
| `mv_match_outcomes` | One terminal claim with match tier, rank, outcome, and fusion version |

The views do not apply the privacy threshold themselves. The API query layer reads the
current `analytics.min_cell_count` setting and suppresses rows whose count is below it,
rendering them as `< 3` rather than zero. This keeps the setting changeable without
rebuilding the materialized views. Hidden reports are excluded, while anonymised
non-personal report facts remain eligible for aggregate statistics according to the
retention rules.

Timestamps are grouped in `Asia/Jakarta` (WIB). Each view has a unique index required by
`REFRESH MATERIALIZED VIEW CONCURRENTLY`. Refresh them hourly in the order listed at the
bottom of [`views.sql`](views.sql). This directory contains no participant data and is
safe to commit.
