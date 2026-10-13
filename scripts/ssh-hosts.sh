#!/usr/bin/env bash
# 手元の Mac で実行: ./scripts/ssh-hosts.sh <s1のIP> <s2のIP> <s3のIP>
# ~/.ssh/config の ISUCON ブロックを作り直す (無ければ末尾に追加)。台数は引数の数で決まる。
# ユーザー名を変えるとき: ISU_USER=ubuntu ./scripts/ssh-hosts.sh ...
set -euo pipefail

[ $# -ge 1 ] || { echo "usage: $0 <s1のIP> [s2のIP] [s3のIP] ..."; exit 1; }
for ip in "$@"; do
  [[ $ip =~ ^[0-9]{1,3}(\.[0-9]{1,3}){3}$ ]] || { echo "IP の形式ではありません: $ip"; exit 1; }
done

CONFIG=~/.ssh/config
BEGIN='# --- ISUCON (ssh-hosts.sh が管理) ---'
END='# --- END ISUCON ---'
USER_NAME=${ISU_USER:-isucon}
KEY=${ISU_KEY:-~/.ssh/id_ed25519}

mkdir -p ~/.ssh && chmod 700 ~/.ssh
touch "$CONFIG" && chmod 600 "$CONFIG"
cp -p "$CONFIG" "$CONFIG.bak-isucon"

# 古い接続の使い回しを切る (IP が変わっても前のサーバーに繋がるのを防ぐ)
names=()
for i in $(seq 1 $#); do
  names+=("s$i")
  ssh -O exit "s$i" >/dev/null 2>&1 || true
done

# 既存の ISUCON ブロックを除いてから、新しいブロックを末尾に付ける
tmp=$(mktemp)
awk -v b="$BEGIN" -v e="$END" '
  index($0, "# --- ISUCON") == 1 { skip = 1 }
  !skip { print }
  skip && $0 == e { skip = 0 }
' "$CONFIG" > "$tmp"

{
  echo ""
  echo "$BEGIN"
  i=1
  for ip in "$@"; do
    echo "Host s$i"
    echo "  HostName $ip"
    [ $i -eq 1 ] && echo "  LocalForward 1080 localhost:1080"
    i=$((i + 1))
  done
  echo ""
  echo "Host ${names[*]}"
  echo "  User $USER_NAME"
  echo "  IdentityFile $KEY"
  echo "  AddKeysToAgent yes"
  echo "  UseKeychain yes"
  echo "  ForwardAgent yes"
  echo "  StrictHostKeyChecking accept-new"
  echo "  ServerAliveInterval 30"
  echo "  ControlMaster auto"
  echo "  ControlPath ~/.ssh/cm-%r@%h:%p"
  echo "  ControlPersist 10m"
  echo "$END"
} >> "$tmp"

# 連続した空行を1つにまとめて書き戻す
cat -s "$tmp" > "$CONFIG"
rm -f "$tmp"

# 同じ IP が前回別のサーバーだった場合の「host key が変わった」エラーを防ぐ
for ip in "$@"; do
  ssh-keygen -R "$ip" >/dev/null 2>&1 || true
done

ssh-add --apple-use-keychain "$KEY" >/dev/null 2>&1 || ssh-add "$KEY" >/dev/null 2>&1 || true

echo "更新しました (バックアップ: $CONFIG.bak-isucon)"
i=1
for ip in "$@"; do echo "  s$i -> $ip"; i=$((i + 1)); done
echo
echo "確認: ssh s1 'hostname && sudo -n true && echo sudo-ok && ssh -T git@github.com'"
