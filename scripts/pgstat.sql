\pset pager off
\echo pg_stat_statements: 合計実行時間の上位30件 (bench-prep でリセット)
SELECT
  round((100 * total_exec_time / sum(total_exec_time) OVER ())::numeric, 1) AS "%",
  round((total_exec_time / 1000)::numeric, 2) AS total_s,
  calls,
  round(mean_exec_time::numeric, 2) AS mean_ms,
  round(max_exec_time::numeric, 1) AS max_ms,
  round(rows::numeric / nullif(calls, 0), 1) AS rows_per_call,
  round(100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0), 1) AS hit_pct,
  round((blk_read_time / 1000)::numeric, 2) AS read_s,
  temp_blks_written AS temp_blks,
  left(regexp_replace(query, '\s+', ' ', 'g'), 120) AS query
FROM pg_stat_statements
WHERE dbid <> (SELECT oid FROM pg_database WHERE datname = 'postgres')
ORDER BY total_exec_time DESC
LIMIT 30;

\echo 上位10件の全文 (EXPLAIN するときは $1 などを実際の値に置き換える)
\pset format unaligned
\pset tuples_only on
\pset fieldsep ': '
SELECT row_number() OVER (ORDER BY total_exec_time DESC), regexp_replace(query, '\s+', ' ', 'g')
FROM pg_stat_statements
WHERE dbid <> (SELECT oid FROM pg_database WHERE datname = 'postgres')
ORDER BY total_exec_time DESC
LIMIT 10;
