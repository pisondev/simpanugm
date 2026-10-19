-- +goose Up
CREATE OR REPLACE VIEW ai_monitoring_assist_outcomes AS
WITH combined AS (
    SELECT id, 'lost' AS report_type, assist_outcome FROM lost_reports WHERE assist_outcome IS NOT NULL
    UNION ALL
    SELECT id, 'found' AS report_type, assist_outcome FROM found_reports WHERE assist_outcome IS NOT NULL
),
expanded AS (
    SELECT report_type, id, (kv).key AS field_name, (kv).value #>> '{}' AS outcome
    FROM combined,
    LATERAL jsonb_each(assist_outcome) AS kv
)
SELECT
    field_name,
    outcome,
    COUNT(*)::bigint AS count
FROM expanded
GROUP BY field_name, outcome
ORDER BY field_name, outcome;

CREATE OR REPLACE VIEW ai_monitoring_dismiss_rates AS
SELECT
    tier,
    COUNT(*)::bigint AS total_matches,
    COUNT(*) FILTER (WHERE owner_dismissed_at IS NOT NULL)::bigint AS owner_dismissed_count,
    COUNT(*) FILTER (WHERE status = 'dismissed')::bigint AS officer_dismissed_count,
    CASE WHEN COUNT(*) > 0 THEN
        ROUND(COUNT(*) FILTER (WHERE owner_dismissed_at IS NOT NULL)::numeric / COUNT(*)::numeric, 4)
    ELSE 0 END AS owner_dismiss_rate
FROM matches
WHERE tier IS NOT NULL
GROUP BY tier
ORDER BY tier;

CREATE OR REPLACE VIEW ai_monitoring_confirmed_ranks AS
SELECT
    COUNT(*)::bigint AS confirmed_total,
    COUNT(*) FILTER (WHERE rank_for_lost = 1)::bigint AS rank_1_count,
    COUNT(*) FILTER (WHERE rank_for_lost <= 5)::bigint AS rank_5_count,
    COUNT(*) FILTER (WHERE rank_for_lost <= 10)::bigint AS rank_10_count,
    CASE WHEN COUNT(*) > 0 THEN
        ROUND(COUNT(*) FILTER (WHERE rank_for_lost = 1)::numeric / COUNT(*)::numeric, 4)
    ELSE 0 END AS recall_at_1,
    CASE WHEN COUNT(*) > 0 THEN
        ROUND(COUNT(*) FILTER (WHERE rank_for_lost <= 5)::numeric / COUNT(*)::numeric, 4)
    ELSE 0 END AS recall_at_5,
    CASE WHEN COUNT(*) > 0 THEN
        ROUND(COUNT(*) FILTER (WHERE rank_for_lost <= 10)::numeric / COUNT(*)::numeric, 4)
    ELSE 0 END AS recall_at_10
FROM matches
WHERE status = 'confirmed';

CREATE OR REPLACE VIEW ai_monitoring_top1_histogram AS
WITH top_matches AS (
    SELECT overall
    FROM matches
    WHERE rank_for_lost = 1
),
buckets AS (
    SELECT bucket_floor, bucket_ceil, label FROM (
        VALUES
            (0.00::numeric, 0.40::numeric, '< 0.40'),
            (0.40::numeric, 0.55::numeric, '0.40 - 0.54 (low)'),
            (0.55::numeric, 0.72::numeric, '0.55 - 0.71 (medium)'),
            (0.72::numeric, 0.85::numeric, '0.72 - 0.84 (high)'),
            (0.85::numeric, 1.01::numeric, '>= 0.85 (very high)')
    ) AS b(bucket_floor, bucket_ceil, label)
)
SELECT
    b.label AS bucket_label,
    b.bucket_floor,
    b.bucket_ceil,
    COUNT(t.overall)::bigint AS count
FROM buckets b
LEFT JOIN top_matches t ON t.overall >= b.bucket_floor AND t.overall < b.bucket_ceil
GROUP BY b.label, b.bucket_floor, b.bucket_ceil
ORDER BY b.bucket_floor;

CREATE OR REPLACE VIEW ai_monitoring_summary AS
SELECT
    COALESCE((SELECT jsonb_agg(jsonb_build_object('field', field_name, 'outcome', outcome, 'count', count)) FROM ai_monitoring_assist_outcomes), '[]'::jsonb) AS assist_outcomes,
    COALESCE((SELECT jsonb_agg(jsonb_build_object('tier', tier, 'total', total_matches, 'owner_dismissed', owner_dismissed_count, 'officer_dismissed', officer_dismissed_count, 'dismiss_rate', owner_dismiss_rate)) FROM ai_monitoring_dismiss_rates), '[]'::jsonb) AS dismiss_rates,
    COALESCE((SELECT row_to_json(r)::jsonb FROM ai_monitoring_confirmed_ranks r), '{}'::jsonb) AS confirmed_ranks,
    COALESCE((SELECT jsonb_agg(jsonb_build_object('bucket', bucket_label, 'count', count)) FROM ai_monitoring_top1_histogram), '[]'::jsonb) AS top1_histogram;

-- +goose Down
DROP VIEW IF EXISTS ai_monitoring_summary;
DROP VIEW IF EXISTS ai_monitoring_top1_histogram;
DROP VIEW IF EXISTS ai_monitoring_confirmed_ranks;
DROP VIEW IF EXISTS ai_monitoring_dismiss_rates;
DROP VIEW IF EXISTS ai_monitoring_assist_outcomes;
