#!/usr/bin/env bash
# 各サーバーで1回: ./scripts/setup.sh s1
set -euo pipefail

SERVER_ID=${1:?usage: ./scripts/setup.sh s1}
echo "$SERVER_ID" > ~/.server_id

sudo apt-get update -qq
sudo apt-get install -y -qq percona-toolkit htop dstat unzip git

ALP_VERSION=v1.0.21
TASK_VERSION=v3.53.1
SLP_VERSION=v0.2.3
case "$(uname -m)" in
  x86_64) ARCH=amd64 ;;
  aarch64) ARCH=arm64 ;;
  *) echo "unsupported arch"; exit 1 ;;
esac
curl -sSL "https://github.com/tkuchiki/alp/releases/download/${ALP_VERSION}/alp_linux_${ARCH}.tar.gz" | sudo tar xz -C /usr/local/bin alp
curl -sSL "https://github.com/go-task/task/releases/download/${TASK_VERSION}/task_linux_${ARCH}.tar.gz" | sudo tar xz -C /usr/local/bin task
curl -sSL "https://github.com/tkuchiki/slp/releases/download/${SLP_VERSION}/slp_linux_${ARCH}.tar.gz" | sudo tar xz -C /usr/local/bin slp

git config --global user.name  "isucon-${SERVER_ID}"
git config --global user.email "isucon@example.com"
git config --global pull.rebase false

echo "done: SERVER_ID=${SERVER_ID}"
