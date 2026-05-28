SELECT calls,
       round(mean_exec_time::numeric, 3) AS mean_ms,
       round(min_exec_time::numeric, 3) AS min_ms,
       round(max_exec_time::numeric, 3) AS max_ms,
       rows,
       shared_blks_hit,
       shared_blks_read,
       temp_blks_read,
       temp_blks_written,
       left(regexp_replace(query, '\s+', ' ', 'g'), 130) AS query
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 8;

SELECT indexrelname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE indexrelname IN ('post_active_public_feed_idx', 'post_active_author_feed_idx')
ORDER BY indexrelname;

SELECT relname, n_live_tup, seq_scan, seq_tup_read, idx_scan, idx_tup_fetch
FROM pg_stat_user_tables
WHERE relname = 'post';
