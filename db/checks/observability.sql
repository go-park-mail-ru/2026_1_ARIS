\pset pager off

SELECT
	name,
	setting,
	unit
FROM pg_settings
WHERE name IN (
	'log_min_duration_statement',
	'log_lock_waits',
	'deadlock_timeout',
	'track_io_timing',
	'shared_preload_libraries',
	'pg_stat_statements.track'
)
ORDER BY name;

SELECT
	extname,
	extversion
FROM pg_extension
WHERE extname IN ('pg_stat_statements');

SELECT
	datname,
	numbackends,
	xact_commit,
	xact_rollback,
	deadlocks,
	blk_read_time,
	blk_write_time,
	temp_files,
	pg_size_pretty(temp_bytes) AS temp_bytes
FROM pg_stat_database
WHERE datname = current_database();

SELECT to_regclass('public.pg_stat_statements') IS NOT NULL AS has_pg_stat_statements \gset
\if :has_pg_stat_statements
SELECT
	calls,
	round(total_exec_time::numeric, 2) AS total_exec_ms,
	round(mean_exec_time::numeric, 2) AS mean_exec_ms,
	rows,
	left(regexp_replace(query, '\s+', ' ', 'g'), 180) AS query
FROM pg_stat_statements
WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
ORDER BY total_exec_time DESC
LIMIT 10;
\else
SELECT 'pg_stat_statements view is not available' AS warning;
\endif
