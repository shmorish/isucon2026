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
  section "バージョン" 'nginx -v 2>&1; psql --version; ~/local/go/bin/go version'
  section "ホーム / webapp の構成" 'ls -la ~; find ~/webapp -maxdepth 2 -not -path "*/node_modules*" -not -path "*/.git*" | sort'
  section "大きいファイル" 'du -ah ~ 2>/dev/null | sort -rh | head -20'

  LIMIT=600 section "nginx の実効設定 (nginx -T)" 'sudo nginx -T 2>/dev/null | grep -Ev "^\s*(#|$)"'

  if systemctl is-active -q postgresql; then
    PSQL='sudo -u postgres psql -X -d isureal'
    section "PostgreSQL の主要設定" "$PSQL -c \"SELECT name, setting, unit FROM pg_settings WHERE name IN ('server_version','shared_buffers','work_mem','effective_cache_size','max_connections','synchronous_commit','fsync','wal_level','listen_addresses','log_min_duration_statement','shared_preload_libraries','max_wal_size','checkpoint_timeout','random_page_cost')\""
    section "pg_hba.conf (コメント除く)" 'sudo grep -Ev "^\s*(#|$)" $(sudo -u postgres psql -XtAc "SHOW hba_file")'
    section "テーブルの件数とサイズ" "$PSQL -c \"SELECT relname, n_live_tup, pg_size_pretty(pg_total_relation_size(relid)) AS total FROM pg_stat_user_tables ORDER BY pg_total_relation_size(relid) DESC\""
    LIMIT=3000 section "テーブル定義 (\\d)" "for t in \$($PSQL -XtAc \"SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY 1\"); do $PSQL -c \"\\\\d \$t\"; done"
  fi
} > "$OUT"

echo "wrote $OUT ($(wc -l < "$OUT") lines)"
