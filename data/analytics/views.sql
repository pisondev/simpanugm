-- Reference definitions for the analytics materialized views.
--
-- These are reviewed SQL definitions, not migrations. They follow the canonical
-- view names and grains in docs/architecture/data-model.md. Suppression for
-- BR-PRV-05 belongs in the API query layer so the runtime setting remains live.
-- All timestamps are grouped in WIB, while PostgreSQL stores instants in UTC.

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_reports_daily AS
WITH report_events AS (
    SELECT
        'lost'::text AS report_type,
        lr.lost_from AS event_at,
        lr.category_id,
        leaf.name AS category_name,
        lrp.location_id,
        location.name AS location_name,
        COALESCE(lrp.zone_id, location.zone_id) AS zone_id,
        zone.name AS zone_name,
        CASE
            WHEN lrp.location_id IS NOT NULL THEN 'location:' || lrp.location_id::text
            ELSE 'zone:' || lrp.zone_id::text
        END AS place_key
    FROM lost_reports AS lr
    JOIN lost_report_places AS lrp ON lrp.lost_report_id = lr.id
    JOIN categories AS leaf ON leaf.id = lr.category_id
    LEFT JOIN locations AS location ON location.id = lrp.location_id
    LEFT JOIN zones AS zone ON zone.id = COALESCE(lrp.zone_id, location.zone_id)
    WHERE lr.hidden_at IS NULL
    UNION ALL
    SELECT
        'found'::text AS report_type,
        fr.found_at AS event_at,
        fr.category_id,
        leaf.name AS category_name,
        fr.found_location_id AS location_id,
        location.name AS location_name,
        location.zone_id,
        zone.name AS zone_name,
        'location:' || fr.found_location_id::text AS place_key
    FROM found_reports AS fr
    JOIN categories AS leaf ON leaf.id = fr.category_id
    JOIN locations AS location ON location.id = fr.found_location_id
    JOIN zones AS zone ON zone.id = location.zone_id
    WHERE fr.hidden_at IS NULL
)
SELECT
    (event_at AT TIME ZONE 'Asia/Jakarta')::date AS event_date,
    report_type,
    category_id,
    category_name,
    location_id,
    location_name,
    zone_id,
    zone_name,
    place_key,
    COUNT(*) AS report_count
FROM report_events
GROUP BY
    1, 2, 3, 4, 5, 6, 7, 8, 9;

CREATE UNIQUE INDEX IF NOT EXISTS mv_reports_daily_key
    ON mv_reports_daily (event_date, report_type, category_id, place_key);

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_hour_weekday AS
WITH report_events AS (
    SELECT 'lost'::text AS report_type, lr.lost_from AS event_at
    FROM lost_reports AS lr
    WHERE lr.hidden_at IS NULL
    UNION ALL
    SELECT 'found'::text AS report_type, fr.found_at AS event_at
    FROM found_reports AS fr
    WHERE fr.hidden_at IS NULL
)
SELECT
    report_type,
    EXTRACT(ISODOW FROM event_at AT TIME ZONE 'Asia/Jakarta')::smallint AS weekday,
    EXTRACT(HOUR FROM event_at AT TIME ZONE 'Asia/Jakarta')::smallint AS hour_of_day,
    COUNT(*) AS report_count
FROM report_events
GROUP BY 1, 2, 3;

CREATE UNIQUE INDEX IF NOT EXISTS mv_hour_weekday_key
    ON mv_hour_weekday (report_type, weekday, hour_of_day);

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_return_times AS
SELECT
    fr.id AS found_report_id,
    fr.category_id,
    category.name AS category_name,
    location.zone_id,
    zone.name AS zone_name,
    fr.found_at,
    fr.received_at,
    returned.returned_at,
    EXTRACT(EPOCH FROM (returned.returned_at - fr.found_at)) / 3600.0 AS hours_to_return
FROM found_reports AS fr
JOIN categories AS category ON category.id = fr.category_id
JOIN locations AS location ON location.id = fr.found_location_id
JOIN zones AS zone ON zone.id = location.zone_id
LEFT JOIN LATERAL (
    SELECT MIN(sh.created_at) AS returned_at
    FROM status_history AS sh
    WHERE sh.entity_type = 'found_report'
      AND sh.entity_id = fr.id
      AND sh.to_status = 'returned'
) AS returned ON TRUE
WHERE fr.hidden_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS mv_return_times_key
    ON mv_return_times (found_report_id);

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_match_outcomes AS
SELECT
    claim.id AS claim_id,
    COALESCE(claim.handed_over_at, claim.decided_at, claim.updated_at) AS closed_at,
    m.tier,
    m.rank_for_lost AS rank_at_claim,
    claim.status AS outcome,
    m.fusion_version_id,
    fusion.name AS fusion_version_name
FROM claims AS claim
JOIN lost_reports AS lost ON lost.id = claim.lost_report_id
JOIN found_reports AS found ON found.id = claim.found_report_id
LEFT JOIN matches AS m ON m.id = claim.match_id
LEFT JOIN model_versions AS fusion ON fusion.id = m.fusion_version_id
WHERE claim.status IN ('handed_over', 'rejected', 'withdrawn')
  AND lost.hidden_at IS NULL
  AND found.hidden_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS mv_match_outcomes_key
    ON mv_match_outcomes (claim_id);

-- Refresh from the hourly worker after migrations have created the views and indexes:
--   REFRESH MATERIALIZED VIEW CONCURRENTLY mv_reports_daily;
--   REFRESH MATERIALIZED VIEW CONCURRENTLY mv_hour_weekday;
--   REFRESH MATERIALIZED VIEW CONCURRENTLY mv_return_times;
--   REFRESH MATERIALIZED VIEW CONCURRENTLY mv_match_outcomes;
