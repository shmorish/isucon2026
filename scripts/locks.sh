#!/usr/bin/env bash
# s4 で実行。ベンチ中のロック待ちを0.5秒ごとに記録し、待たせている側の状態とクエリで集計する: ./scripts/locks.sh [秒数]
# 要約は標準出力と /tmp/locks.txt に出す。生データは /tmp/locks.log。
# holder の state が "idle in transaction" なら、ロックを持ったままアプリとの往復を待っている (クエリ欄はそのトランザクションで最後に打ったもの)
set -euo pipefail

DURATION=${1:-75}
RAW=/tmp/locks.log
OUT=/tmp/locks.txt
PSQL=(sudo -u postgres /usr/lib/postgresql/16/bin/psql -X -q -At -F '|' -d isureal)

: > "$RAW"
end=$((SECONDS + DURATION))
while [ $SECONDS -lt $end ]; do
  "${PSQL[@]}" -c "
    SELECT h.client_addr, h.state, round(extract(epoch FROM now() - h.xact_start)::numeric, 2),
           left(regexp_replace(h.query, '\s+', ' ', 'g'), 90), left(regexp_replace(w.query, '\s+', ' ', 'g'), 90), w.wait_event
    FROM pg_stat_activity w CROSS JOIN LATERAL unnest(pg_blocking_pids(w.pid)) AS b(pid)
    JOIN pg_stat_activity h ON h.pid = b.pid
    WHERE w.wait_event_type = 'Lock'" >> "$RAW" || true
  sleep 0.5
done

{
  echo "ロック待ち: $((DURATION * 2)) 回のサンプルで見えた (待つ側, 待たせる側) の組 $(wc -l < "$RAW") 件"
  echo "--- 待たせる側の接続元・状態 (件数)"
  cut -d'|' -f1,2 "$RAW" | sort | uniq -c | sort -rn
  echo "--- 待たせる側の接続元|状態|最後のクエリ (件数, トランザクション経過秒の平均)"
  awk -F'|' '{ k = $1 "|" $2 "|" $4; n[k]++; s[k] += $3 } END { for (k in n) printf "%6d %6.2fs %s\n", n[k], s[k] / n[k], k }' "$RAW" | sort -rn | head -20
  echo "--- 待つ側のクエリ (件数)"
  cut -d'|' -f5 "$RAW" | sort | uniq -c | sort -rn | head -10
  echo "--- 待つ側のロックの種類 (件数。extend はテーブル拡張、transactionid/tuple は行ロック)"
  cut -d'|' -f6 "$RAW" | sort | uniq -c | sort -rn
} | tee "$OUT"
