SELECT calls,
       round(mean_exec_time::numeric, 3) AS mean_ms,
       round(min_exec_time::numeric, 3) AS min_ms,
       round(max_exec_time::numeric, 3) AS max_ms,
       rows,
       shared_blks_hit,
       shared_blks_read,
       temp_blks_read,
       temp_blks_written,
       left(regexp_replace(query, '\s+', ' ', 'g'), 150) AS query
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 10;

SELECT indexrelname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE indexrelname IN (
  'message_active_chat_created_idx',
  'message_active_chat_id_idx',
  'friendship_requester_status_idx',
  'friendship_addressee_status_idx'
)
ORDER BY indexrelname;

SELECT relname, n_live_tup, seq_scan, seq_tup_read, idx_scan, idx_tup_fetch
FROM pg_stat_user_tables
WHERE relname IN ('message', 'friendship')
ORDER BY relname;
