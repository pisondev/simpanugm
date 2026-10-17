-- Reference definitions for the analytics materialized views.
--
-- These are reviewed SQL definitions, not migrations. The API must expose only the
-- suppressed result columns from the public views below. The literal threshold 3 is
-- the current value of analytics.min_cell_count in data/seed/settings.yaml.
-- All timestamps are grouped in WIB, while PostgreSQL stores instants in UTC.

CREATE MATERIALIZED VIEW IF NOT EXISTS analytics_report_events AS
SELECT
    'lost'::text AS report_type,
    lr.lost_from AS event_at,
    l.id AS location_id,
    l.name AS location_name,
    l.zone_id,
    parent.id AS parent_category_id,
    parent.name AS parent_category_name
FROM lost_reports AS lr
JOIN lost_report_places AS lrp ON lrp.lost_report_id = lr.id
JOIN locations AS l ON l.id = lrp.location_id
JOIN categories AS leaf ON leaf.id = lr.category_id
JOIN categories AS parent ON parent.id = leaf.parent_id
WHERE lr.hidden_at IS NULL
UNION ALL
SELECT
    'found'::text AS report_type,
    fr.found_at AS event_at,
    l.id AS location_id,
    l.name AS location_name,
    l.zone_id,
    parent.id AS parent_category_id,
    parent.name AS parent_category_name
FROM found_reports AS fr
JOIN locations AS l ON l.id = fr.found_location_id
JOIN categories AS leaf ON leaf.id = fr.category_id
JOIN categories AS parent ON parent.id = leaf.parent_id
WHERE fr.hidden_at IS NULL;

CREATE MATERIALIZED VIEW IF NOT EXISTS analytics_location_daily AS
WITH grouped AS (
    SELECT
        (event_at AT TIME ZONE 'Asia/Jakarta')::date AS event_date,
        location_id,
        location_name,
        zone_id,
        parent_category_id,
        parent_category_name,
        COUNT(*) FILTER (WHERE report_type = 'lost') AS lost_total,
        COUNT(*) FILTER (WHERE report_type = 'found') AS found_total,
        COUNT(*) AS all_total
    FROM analytics_report_events
    GROUP BY 1, 2, 3, 4, 5, 6
)
SELECT
    event_date,
    location_id,
    location_name,
    zone_id,
    parent_category_id,
    parent_category_name,
    CASE WHEN lost_total >= 3 THEN lost_total ELSE NULL END AS lost_count,
    CASE WHEN found_total >= 3 THEN found_total ELSE NULL END AS found_count,
    CASE WHEN all_total >= 3 THEN all_total ELSE NULL END AS report_count
FROM grouped;

CREATE MATERIALIZED VIEW IF NOT EXISTS analytics_weekly_time AS
WITH grouped AS (
    SELECT
        date_trunc('week', event_at AT TIME ZONE 'Asia/Jakarta')::date AS week_start,
        report_type,
        parent_category_id,
        parent_category_name,
        EXTRACT(ISODOW FROM event_at AT TIME ZONE 'Asia/Jakarta')::smallint AS weekday,
        EXTRACT(HOUR FROM event_at AT TIME ZONE 'Asia/Jakarta')::smallint AS hour_of_day,
        COUNT(*) AS report_total
    FROM analytics_report_events
    GROUP BY 1, 2, 3, 4, 5, 6
)
SELECT
    week_start,
    report_type,
    parent_category_id,
    parent_category_name,
    weekday,
    hour_of_day,
    CASE WHEN report_total >= 3 THEN report_total ELSE NULL END AS report_count
FROM grouped;

CREATE MATERIALIZED VIEW IF NOT EXISTS analytics_weekly_outcomes AS
WITH lost AS (
    SELECT
        date_trunc('week', lr.created_at AT TIME ZONE 'Asia/Jakarta')::date AS week_start,
        COUNT(*) AS lost_total,
        COUNT(*) FILTER (WHERE lr.status = 'returned') AS returned_total,
        COUNT(*) FILTER (
            WHERE lr.status IN ('reported', 'potential_match', 'under_verification', 'claimed')
        ) AS open_total,
        percentile_cont(0.5) WITHIN GROUP (
            ORDER BY EXTRACT(EPOCH FROM (returned_at - lr.created_at)) / 3600.0
        ) FILTER (WHERE returned_at IS NOT NULL) AS median_return_hours
    FROM lost_reports AS lr
    LEFT JOIN LATERAL (
        SELECT MIN(sh.created_at) AS returned_at
        FROM status_history AS sh
        WHERE sh.entity_type = 'lost_report'
          AND sh.entity_id = lr.id
          AND sh.to_status = 'returned'
    ) AS returned ON TRUE
    WHERE lr.hidden_at IS NULL
    GROUP BY 1
), found AS (
    SELECT
        date_trunc('week', fr.created_at AT TIME ZONE 'Asia/Jakarta')::date AS week_start,
        COUNT(*) AS found_total,
        COUNT(*) FILTER (WHERE fr.status = 'unclaimed') AS unclaimed_total,
        COUNT(*) FILTER (
            WHERE fr.status IN ('reported', 'potential_match', 'under_verification', 'claimed')
        ) AS open_total
    FROM found_reports AS fr
    WHERE fr.hidden_at IS NULL
    GROUP BY 1
), grouped AS (
    SELECT
        COALESCE(lost.week_start, found.week_start) AS week_start,
        COALESCE(lost.lost_total, 0) AS lost_total,
        COALESCE(lost.returned_total, 0) AS returned_total,
        COALESCE(lost.open_total, 0) + COALESCE(found.open_total, 0) AS open_total,
        COALESCE(found.found_total, 0) AS found_total,
        COALESCE(found.unclaimed_total, 0) AS unclaimed_total,
        lost.median_return_hours
    FROM lost
    FULL OUTER JOIN found USING (week_start)
)
SELECT
    week_start,
    CASE WHEN lost_total >= 3 THEN lost_total ELSE NULL END AS lost_count,
    CASE WHEN found_total >= 3 THEN found_total ELSE NULL END AS found_count,
    CASE WHEN lost_total >= 3 THEN returned_total::numeric / lost_total ELSE NULL END AS return_rate,
    CASE WHEN lost_total >= 3 THEN median_return_hours ELSE NULL END AS median_return_hours,
    CASE WHEN found_total >= 3 THEN unclaimed_total::numeric / found_total ELSE NULL END AS unclaimed_rate,
    CASE WHEN lost_total + found_total >= 3 THEN open_total ELSE NULL END AS open_case_count
FROM grouped;

-- Refresh in dependency order from the hourly worker:
--   REFRESH MATERIALIZED VIEW analytics_report_events;
--   REFRESH MATERIALIZED VIEW analytics_location_daily;
--   REFRESH MATERIALIZED VIEW analytics_weekly_time;
--   REFRESH MATERIALIZED VIEW analytics_weekly_outcomes;
