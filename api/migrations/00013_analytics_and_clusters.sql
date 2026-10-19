-- +goose Up
CREATE TABLE cluster_runs (
    id           uuid PRIMARY KEY,
    params       jsonb NOT NULL DEFAULT '{}',
    versions     jsonb NOT NULL DEFAULT '{}',
    status       text NOT NULL CHECK (status IN ('running', 'done', 'failed')),
    triggered_by uuid REFERENCES users (id),
    started_at   timestamptz NOT NULL,
    finished_at  timestamptz
);

CREATE TABLE cluster_members (
    run_id        uuid NOT NULL REFERENCES cluster_runs (id) ON DELETE CASCADE,
    report_type   text NOT NULL CHECK (report_type IN ('lost', 'found')),
    report_id     uuid NOT NULL,
    cluster_label integer NOT NULL,
    probability   real NOT NULL DEFAULT 1.0,
    PRIMARY KEY (run_id, report_type, report_id)
);

CREATE INDEX cluster_members_run_label ON cluster_members (run_id, cluster_label);

CREATE MATERIALIZED VIEW mv_reports_daily AS
WITH lost_reps AS (
    SELECT
        (lr.created_at AT TIME ZONE 'Asia/Jakarta')::date AS day_wib,
        'lost'::text AS report_type,
        lr.category_id,
        COALESCE(p.location_id, '00000000-0000-0000-0000-000000000000'::uuid) AS location_id
    FROM lost_reports lr
    LEFT JOIN lost_report_places p ON p.lost_report_id = lr.id AND p.position = 1
),
found_reps AS (
    SELECT
        (fr.found_at AT TIME ZONE 'Asia/Jakarta')::date AS day_wib,
        'found'::text AS report_type,
        fr.category_id,
        fr.found_location_id AS location_id
    FROM found_reports fr
),
all_reps AS (
    SELECT * FROM lost_reps
    UNION ALL
    SELECT * FROM found_reps
)
SELECT
    day_wib,
    report_type,
    category_id,
    location_id,
    COUNT(*)::bigint AS count
FROM all_reps
GROUP BY day_wib, report_type, category_id, location_id;

CREATE UNIQUE INDEX mv_reports_daily_pk ON mv_reports_daily (day_wib, report_type, category_id, location_id);

CREATE MATERIALIZED VIEW mv_hour_weekday AS
WITH all_reports AS (
    SELECT
        'lost'::text AS report_type,
        EXTRACT(ISODOW FROM lr.created_at AT TIME ZONE 'Asia/Jakarta')::int AS weekday,
        EXTRACT(HOUR FROM lr.created_at AT TIME ZONE 'Asia/Jakarta')::int AS hour_wib
    FROM lost_reports lr
    UNION ALL
    SELECT
        'found'::text AS report_type,
        EXTRACT(ISODOW FROM fr.found_at AT TIME ZONE 'Asia/Jakarta')::int AS weekday,
        EXTRACT(HOUR FROM fr.found_at AT TIME ZONE 'Asia/Jakarta')::int AS hour_wib
    FROM found_reports fr
)
SELECT
    report_type,
    weekday,
    hour_wib,
    COUNT(*)::bigint AS count
FROM all_reports
GROUP BY report_type, weekday, hour_wib;

CREATE UNIQUE INDEX mv_hour_weekday_pk ON mv_hour_weekday (report_type, weekday, hour_wib);

CREATE MATERIALIZED VIEW mv_return_times AS
SELECT
    fr.id AS found_report_id,
    fr.category_id,
    l.zone_id,
    fr.found_at,
    fr.received_at,
    c.handed_over_at AS returned_at,
    CASE
        WHEN c.handed_over_at IS NOT NULL THEN
            ROUND((EXTRACT(EPOCH FROM (c.handed_over_at - fr.found_at)) / 3600.0)::numeric, 2)
        ELSE NULL
    END AS hours_to_return
FROM found_reports fr
JOIN locations l ON l.id = fr.found_location_id
LEFT JOIN LATERAL (
    SELECT handed_over_at
    FROM claims
    WHERE found_report_id = fr.id AND status = 'handed_over'
    ORDER BY handed_over_at DESC
    LIMIT 1
) c ON true;

CREATE UNIQUE INDEX mv_return_times_pk ON mv_return_times (found_report_id);

CREATE MATERIALIZED VIEW mv_match_outcomes AS
SELECT
    c.id AS claim_id,
    m.tier,
    m.rank_for_lost AS rank_at_claim,
    c.status AS outcome,
    COALESCE(m.fusion_version_id, '00000000-0000-0000-0000-000000000000'::uuid) AS fusion_version_id,
    COALESCE(c.handed_over_at, c.decided_at, c.updated_at) AS closed_at
FROM claims c
LEFT JOIN matches m ON m.id = c.match_id
WHERE c.status IN ('handed_over', 'rejected', 'withdrawn', 'expired');

CREATE UNIQUE INDEX mv_match_outcomes_pk ON mv_match_outcomes (claim_id);

-- +goose Down
DROP MATERIALIZED VIEW IF EXISTS mv_match_outcomes;
DROP MATERIALIZED VIEW IF EXISTS mv_return_times;
DROP MATERIALIZED VIEW IF EXISTS mv_hour_weekday;
DROP MATERIALIZED VIEW IF EXISTS mv_reports_daily;
DROP TABLE IF EXISTS cluster_members;
DROP TABLE IF EXISTS cluster_runs;
