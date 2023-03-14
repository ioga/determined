--ALTER TABLE trials
--	ADD COLUMN summary_metrics jsonb DEFAULT NULL;

WITH trial_metrics as (
SELECT
	name,
	trial_id,
	sum(entries) FILTER (WHERE metric_type != 'number') as nonumbers
FROM (
	SELECT
	name,
	jsonb_typeof(steps.metrics->'avg_metrics'->name) as metric_type,
	trial_id,
	count(1) as entries
	FROM (
		SELECT DISTINCT
		jsonb_object_keys(s.metrics->'avg_metrics') as name
		FROM steps s
	) names, steps
	GROUP BY name, metric_type, trial_id
) typed
where metric_type is not null
GROUP BY name, trial_id
ORDER BY trial_id, name
),
numeric_trial_metrics as (
SELECT name, trial_id
FROM trial_metrics
WHERE nonumbers IS null
),
trial_metric_aggs as (
SELECT
	name,
	numeric_trial_metrics.trial_id,
	count(1) as count_agg,
	sum((steps.metrics->'avg_metrics'->>name)::double precision) as sum_agg,
	min((steps.metrics->'avg_metrics'->>name)::double precision) as min_agg,
	max((steps.metrics->'avg_metrics'->>name)::double precision) as max_agg
FROM numeric_trial_metrics INNER JOIN steps
ON steps.trial_id=numeric_trial_metrics.trial_id
WHERE steps.metrics->'avg_metrics'->name IS NOT NULL
GROUP BY 1, 2
),
latest_training AS (
  SELECT s.trial_id,
	unpacked.key as name,
	unpacked.value as latest_value
  FROM (
      SELECT s.*,
        ROW_NUMBER() OVER(
          PARTITION BY s.trial_id
          ORDER BY s.end_time DESC
        ) AS rank
      FROM steps s
    ) s, jsonb_each(s.metrics->'avg_metrics') unpacked
  WHERE s.rank = 1
),
combined_latest_agg as (SELECT
 coalesce(lt.trial_id, tma.trial_id) as trial_id,
 coalesce(lt.name, tma.name) as name,
 tma.count_agg,
 tma.sum_agg,
 tma.min_agg,
 tma.max_agg,
 lt.latest_value
FROM latest_training lt FULL OUTER JOIN trial_metric_aggs tma ON lt.trial_id = tma.trial_id AND lt.name = tma.name
)
SELECT
	trial_id, jsonb_collect(jsonb_build_object(
		name, jsonb_build_object(
			'count', count_agg,
			'sum', sum_agg,
			'min', min_agg,
			'max', max_agg,
			'last', latest_value
		)
	)) as summary
FROM combined_latest_agg
GROUP BY trial_id
;
