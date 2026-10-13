#!/usr/bin/env bash
# ベンチ中の CPU・メモリを1秒ごとに記録して要約する: ./scripts/monitor.sh [秒数]
# 要約は標準出力と /tmp/resource.txt に出す。生データは /tmp/top.log。
set -euo pipefail

DURATION=${1:-75}
RAW=/tmp/top.log
OUT=/tmp/resource.txt

# 1回目のサンプルは起動からの累計値なので集計では捨てる
LC_ALL=C top -b -E m -d 1 -n $((DURATION + 1)) -w 200 > "$RAW"

LC_ALL=C awk -v host="$(cat ~/.server_id 2>/dev/null || hostname -s)" -v cores="$(nproc)" -v dur="$DURATION" '
  /^top - / { sample++; inproc = 0; next }
  sample < 2 { next }
  /^%Cpu/ {
    for (i = 1; i <= NF; i++) {
      if ($(i+1) ~ /^id/) idle = $i
      if ($(i+1) ~ /^wa/) wa = $i
      if ($(i+1) ~ /^st/) st = $i
    }
    use = 100 - idle; n++
    sum_use += use; if (use > max_use) max_use = use
    sum_wa += wa;   if (wa > max_wa) max_wa = wa
    sum_st += st
    next
  }
  /^MiB Mem/ {
    for (i = 1; i <= NF; i++) {
      if ($(i+1) ~ /^total/) total = $i
      if ($(i+1) ~ /^used/) used = $i
    }
    if (used > max_used) max_used = used
    next
  }
  /^MiB Swap/ { for (i = 1; i <= NF; i++) if ($(i+1) ~ /^used/ && $i > max_swap) max_swap = $i; next }
  /^ *PID / { inproc = 1; next }
  inproc && NF >= 12 {
    cmd = $12; cpu[cmd] += $9
    if ($9 > peak[cmd]) peak[cmd] = $9
    if ($10 > mem[cmd]) mem[cmd] = $10
  }
  END {
    if (n == 0) { print "サンプルが取れていない"; exit 1 }
    printf "# %s (%d コア, %.0f MiB) %d 秒\n\n", host, cores, total, n
    printf "CPU 使用率 (全コア合計を100%%): 平均 %.0f%%  最大 %.0f%%\n", sum_use / n, max_use
    printf "iowait: 平均 %.1f%%  最大 %.1f%%   steal: 平均 %.1f%%\n", sum_wa / n, max_wa, sum_st / n
    printf "メモリ使用: 最大 %.0f / %.0f MiB (%.0f%%)   swap 使用: 最大 %.0f MiB\n\n", max_used, total, max_used * 100 / total, max_swap
    printf "プロセス別 CPU (100%% = 1コア分)\n"
    printf "%-20s %8s %8s %8s\n", "COMMAND", "平均%CPU", "最大%CPU", "最大%MEM"
    fflush()
    for (c in cpu) printf "%-20s %8.1f %8.1f %8.1f\n", c, cpu[c] / n, peak[c], mem[c] | "sort -k2 -nr | head -10"
    close("sort -k2 -nr | head -10")
  }
' "$RAW" | tee "$OUT"
