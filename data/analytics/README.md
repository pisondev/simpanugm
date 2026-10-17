# Analytics reference views

This directory contains the SQL definitions Axelle reviews with the API owner before
they become migrations. The definitions cover the analytics requirements without
including descriptions, private details, reporter identities, or finder identities.

## Views

| View | Purpose |
|---|---|
| `analytics_report_events` | Internal hourly source of location, zone, parent category, timestamp, and report type |
| `analytics_location_daily` | Hotspot map markers, ranked locations, and period filtering |
| `analytics_weekly_time` | Week, weekday, hour, category, and lost/found trend cells |
| `analytics_weekly_outcomes` | Return rate, median hours to return, unclaimed rate, and open cases |

All public aggregate cells use the current `analytics.min_cell_count` threshold of 3.
The SQL returns `NULL` for suppressed cells; the API renders those as `< 3` and never
replaces them with zero. Hidden reports are excluded, while anonymised non-personal
report facts remain eligible for aggregate statistics according to the retention rules.

Timestamps are grouped in `Asia/Jakarta` (WIB). Refresh the materialized views hourly in
the order listed at the bottom of [`views.sql`](views.sql). This directory contains no
participant data and is safe to commit.
