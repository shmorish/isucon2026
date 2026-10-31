#!/usr/bin/env bash
# 手元の Mac で実行: ./scripts/deploy-all.sh [s1 s2 s3 s4]
# 未 push の変更がないか確かめてから、指定したサーバー (省略時は s1〜s4) で task deploy を実行する。
# s4 (DB) を先に終わらせ、残りを同時に実行する。
set -uo pipefail

hosts=("$@")
[ ${#hosts[@]} -gt 0 ] || hosts=(s1 s2 s3 s4)

cd "$(dirname "$0")/.."
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  echo "commit していない変更がある。commit・push してから実行する"; exit 1
fi
git fetch -q
if [ "$(git rev-list --count @{u}..@)" -gt 0 ]; then
  echo "push していない commit がある。git push してから実行する"; exit 1
fi

failed=0
# hosts の deploy を同時に実行して結果を出す
run() {
  local hs=("$@") pids=() i h
  for h in "${hs[@]}"; do
    ssh "$h" 'cd ~ && task deploy' > "/tmp/deploy-$h.log" 2>&1 &
    pids+=($!)
  done
  for i in "${!hs[@]}"; do
    h=${hs[$i]}
    if wait "${pids[$i]}"; then
      echo "OK   $h  $(ssh "$h" 'git -C ~ log -1 --format="%h %s"')"
    else
      failed=1
      echo "FAIL $h  (ログ: /tmp/deploy-$h.log)"
      tail -n 15 "/tmp/deploy-$h.log" | sed 's/^/     /'
    fi
  done
}

# DB (s4) が起動しきってからアプリを再起動する
apps=()
for h in "${hosts[@]}"; do
  if [ "$h" = s4 ]; then run s4; else apps+=("$h"); fi
done
[ ${#apps[@]} -eq 0 ] || run "${apps[@]}"
exit $failed
