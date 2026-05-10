\pset pager off

SELECT
	now() AS checked_at,
	current_database() AS database;

SELECT
	pid,
	usename,
	application_name,
	client_addr,
	state,
	wait_event_type,
	wait_event,
	now() - query_start AS query_age,
	left(regexp_replace(query, '\s+', ' ', 'g'), 180) AS query
FROM pg_stat_activity
WHERE datname = current_database()
	AND pid <> pg_backend_pid()
ORDER BY query_age DESC NULLS LAST
LIMIT 30;

SELECT
	blocked_activity.pid AS blocked_pid,
	blocked_activity.usename AS blocked_user,
	blocking_activity.pid AS blocking_pid,
	blocking_activity.usename AS blocking_user,
	now() - blocked_activity.query_start AS blocked_age,
	left(regexp_replace(blocked_activity.query, '\s+', ' ', 'g'), 160) AS blocked_query,
	left(regexp_replace(blocking_activity.query, '\s+', ' ', 'g'), 160) AS blocking_query
FROM pg_locks blocked_locks
JOIN pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_locks blocking_locks
	ON blocking_locks.locktype = blocked_locks.locktype
	AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
	AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
	AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
	AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
	AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
	AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
	AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
	AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
	AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
	AND blocking_locks.pid <> blocked_locks.pid
JOIN pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted
	AND blocking_locks.granted
ORDER BY blocked_activity.query_start;

SELECT
	datname,
	deadlocks,
	conflicts,
	temp_files,
	pg_size_pretty(temp_bytes) AS temp_bytes
FROM pg_stat_database
WHERE datname = current_database();

SELECT
	schemaname,
	relname,
	seq_scan,
	idx_scan,
	n_live_tup,
	n_dead_tup,
	last_autovacuum,
	last_autoanalyze
FROM pg_stat_user_tables
ORDER BY n_dead_tup DESC, seq_scan DESC
LIMIT 20;

SELECT
	schemaname,
	relname,
	indexrelname,
	idx_scan,
	idx_tup_read,
	idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC, relname, indexrelname
LIMIT 20;
