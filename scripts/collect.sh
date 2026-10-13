#!/usr/bin/env bash
# サーバーの状況を1ファイルにまとめる: ./scripts/collect.sh → <SERVER_ID>/collect.md
# 失敗したコマンドは飛ばして続ける (set -e しない)
set -u

SERVER_ID=${1:-$(cat ~/.server_id 2>/dev/null || hostname -s)}
OUT="$(dirname "$0")/../${SERVER_ID}/collect.md"
mkdir -p "$(dirname "$OUT")"

# パスワードやトークンらしき値を伏せる
mask() { sed -E 's/((PASS|PASSWORD|SECRET|TOKEN|KEY)[A-Z_]*[=:][[:space:]]*).*/\1****/I'; }

section() {
  local title=$1; shift
  printf '\n## %s\n\n```\n' "$title"
  bash -c "$*" 2>&1 | mask | head -n "${LIMIT:-300}"
  printf '```\n'
}

{
  echo "# ${SERVER_ID} ($(hostname)) $(date '+%F %T')"

  section "スペック" 'nproc; lscpu | grep -E "Model name|^CPU\(s\)"; free -h; df -h /; uname -r; grep PRETTY /etc/os-release'
  section "動いているサービス" 'systemctl list-units --type=service --state=running --no-pager --no-legend | grep -Eiv "systemd|getty|ssh|cron|dbus|polkit|udisks|snap|networkd|resolved|journald|rsyslog|unattended|multipathd|amazon|chrony|irqbalance|packagekit|user@|serial|acpid|atd|lvm"'
  section "isu 関連の unit 一覧" 'systemctl list-unit-files --no-pager | grep -Ei "isu|nginx|mysql|maria|redis|memcache|postgres|h2o|envoy|powerdns|pdns"'
  section "isu 関連の unit 定義" 'for u in $(systemctl list-unit-files --no-legend | awk "{print \$1}" | grep -i isu); do systemctl cat "$u"; echo; done'
  section "EnvironmentFile の中身 (値は一部伏せ字)" 'for f in $(systemctl show -p EnvironmentFiles $(systemctl list-unit-files --no-legend | awk "{print \$1}" | grep -i isu) 2>/dev/null | sed -E "s/^EnvironmentFiles=//; s/ \(.*//" | sort -u); do echo "--- $f"; cat "$f"; done'
  section "listen しているポート" 'sudo ss -tlnp'
  section "バージョン" 'nginx -v 2>&1; mysql --version; go version; redis-server --version; h2o --version 2>&1 | head -1'
  section "ホーム / webapp の構成" 'ls -la ~; find ~/webapp -maxdepth 2 -not -path "*/node_modules*" -not -path "*/.git*" | sort'
  section "大きいファイル" 'du -ah ~ 2>/dev/null | sort -rh | head -20'

  LIMIT=600 section "nginx の実効設定 (nginx -T)" 'sudo nginx -T 2>/dev/null | grep -Ev "^\s*(#|$)"'

  section "MySQL の主要変数" 'sudo mysql -N -e "SHOW VARIABLES WHERE Variable_name IN (\"version\",\"innodb_buffer_pool_size\",\"max_connections\",\"slow_query_log\",\"long_query_time\",\"log_bin\",\"innodb_flush_log_at_trx_commit\",\"innodb_doublewrite\",\"performance_schema\",\"bind_address\")"'
  section "テーブルの件数とサイズ" 'sudo mysql -e "SELECT table_schema, table_name, table_rows, ROUND((data_length+index_length)/1024/1024,1) AS mb FROM information_schema.tables WHERE table_schema NOT IN (\"mysql\",\"sys\",\"information_schema\",\"performance_schema\") ORDER BY data_length+index_length DESC"'
  LIMIT=2000 section "テーブル定義 (SHOW CREATE TABLE)" 'for db in $(sudo mysql -N -e "SHOW DATABASES" | grep -Ev "^(mysql|sys|information_schema|performance_schema)$"); do for t in $(sudo mysql -N -e "SHOW TABLES" "$db"); do echo "-- $db.$t"; sudo mysql -N -e "SHOW CREATE TABLE \`$t\`\G" "$db" | sed 1,2d; echo; done; done'
} > "$OUT"

echo "wrote $OUT ($(wc -l < "$OUT") lines)"
