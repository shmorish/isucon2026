#!/usr/bin/env bash
# 手元の Mac で実行: ./scripts/reboot-all.sh [s1 s2 s3]
# 再起動試験: 全台を再起動し、戻ってきたらサービスの状態とアプリの応答を確かめる。
# このあと人間がベンチを回し、ブラウザで画面を確認する。
set -uo pipefail

hosts=("$@")
[ ${#hosts[@]} -gt 0 ] || hosts=(s1 s2 s3)

for h in "${hosts[@]}"; do
  ssh -O exit "$h" >/dev/null 2>&1 || true   # 使い回し中の接続を切る
  ssh "$h" 'sudo systemctl reboot' >/dev/null 2>&1 || true
  echo "再起動: $h"
done

sleep 15
for h in "${hosts[@]}"; do
  for _ in $(seq 1 60); do
    ssh -o ConnectTimeout=5 "$h" true >/dev/null 2>&1 && break
    sleep 5
  done
done

failed=0
for h in "${hosts[@]}"; do
  echo "--- $h"
  if ! ssh -o ConnectTimeout=5 "$h" true >/dev/null 2>&1; then
    echo "  戻ってこない"; failed=1; continue
  fi
  ssh "$h" '
    bad=0
    units=$(systemctl list-unit-files --state=enabled --no-legend | awk "{print \$1}" | grep -Ei "isu|nginx|mysql|maria|redis|memcache|postgres|h2o|envoy|pdns")
    for u in $units; do
      st=$(systemctl is-active "$u"); printf "  %-28s %s\n" "$u" "$st"
      [ "$st" = active ] || bad=1
    done
    failed_units=$(systemctl --failed --no-legend --plain | awk "{print \$1}")
    if [ -n "$failed_units" ]; then echo "  失敗しているユニット: $failed_units"; bad=1; fi
    if systemctl is-enabled nginx >/dev/null 2>&1; then
      code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost/)
      echo "  GET / -> $code"
      [ "$code" = 200 ] || bad=1
    fi
    exit $bad
  ' || failed=1
done
[ $failed -eq 0 ] && echo "全台 OK" || echo "異常あり"
exit $failed
