# isucon-template

ISUCON 当日用テンプレート。repo は各サーバーの `/home/isucon` に置き、`.gitignore` のホワイトリストで必要なものだけ追跡する。

```
CLAUDE.md             当日 Claude に守らせるルールと進め方
notes/manual.md       マニュアルの要点 (仕様担当が当日記入。Claude が最優先で守る)
Taskfile.yml          deploy / analyze / score などの当日コマンド (task --list で一覧)
scripts/setup.sh      各サーバーで1回だけ実行 (alp, slp, pt-query-digest, task 等の導入)
scripts/collect.sh    サーバーの状況 (スペック・サービス・nginx 設定・テーブル定義) を s1/collect.md にまとめる
scripts/monitor.sh    ベンチ中の CPU・メモリ (全体とプロセス別) を記録して要約する
scripts/deploy-all.sh 手元の Mac で実行。全台に同時に task deploy (未 push の変更があれば止まる)
scripts/reboot-all.sh 手元の Mac で実行。再起動試験 (全台を再起動してサービス状態と GET / を確認)
scripts/ssh-hosts.sh  手元の Mac で実行。当日のIPを渡すと ~/.ssh/config に s1〜s3 を書く
scripts/alp.yml       alp の設定。当日 matching_groups を書く
scripts/slp.yml       slp の設定 (合計時間順・上位30件・rows examined 付き)
s1/ s2/ s3/           サーバー別の /etc 設定 (etc/) と環境変数ファイル (home/)。task get-conf で取り込み、task deploy で書き戻す
snippets/             nginx / MySQL / sysctl / Go のコピペ部品
```

## コマンド一覧

| コマンド | いつ使うか |
|---|---|
| `./scripts/ssh-hosts.sh <IP>...` | 当日最初に手元で1回（`~/.ssh/config` を書く） |
| `./scripts/setup.sh s1` | 各サーバーで最初に1回（ツールのインストール） |
| `task get-conf` | 各サーバーで最初に1回（`/etc` の設定と環境変数ファイルを repo に取り込む） |
| `task collect` | 各サーバーで最初に1回（サーバーの状況を `s1/collect.md` に） |
| `./scripts/deploy-all.sh [s1 s2 s3]` | 変更を push した後、毎回（手元から全台に `task deploy`） |
| `task deploy` | 1台だけサーバー上で反映するとき |
| `./scripts/reboot-all.sh` | 大きな変更の後と終盤の再起動試験（手元から） |
| `task analyze` | ベンチの後、毎回（alp / slp / pt-query-digest を集計） |
| `task score S=点数 M="メモ"` | ポータルで結果を見た後、毎回（s1 の `~/scores.log` に記録） |
| `task pprof` | ベンチ開始の直前（手元のブラウザで localhost:1080） |
| `task monitor` | ベンチ開始の直前（CPU・メモリを記録して `/tmp/resource.txt` に要約）。3台同時に取るときは下の「計測付きベンチ」の手元コマンドを使う |

---

## 事前準備（前日まで）

1. このテンプレートから **private** repo を作る（「Use this template」）。競技中に問題の内容を公開するのは規約違反なので、public にしない
2. 作った repo を手元に clone する（`scripts/ssh-hosts.sh` を手元で使うため）
3. SSH の鍵がなければ作る（`ssh-keygen -t ed25519`、質問は全部 Enter）
4. 公開鍵を GitHub に登録する（`pbcopy < ~/.ssh/id_ed25519.pub` → https://github.com/settings/keys に貼る）。https://github.com/<ユーザー名>.keys に表示されれば登録済み
5. 手元で `ssh -T git@github.com` を実行し、`Hi <ユーザー名>!` と出ることを確認する
6. AWS アカウントを用意する（競技は各チームの AWS アカウントで EC2 を起動する）
7. private-isu や過去問で、下の「当日の手順」を通しで練習する

---

## 当日の手順（最初の計測まで: 10:00〜10:45）

ゴールは「計測付きのベンチを1回回し、重いエンドポイントとクエリの上位5つをチームで共有する」こと。ここまではアプリの改善に手を出さない。

前提: 3人チーム、サーバー3台（s1〜s3）。`bash` 付きのコードブロックは**手元の Mac**、それ以外は**競技サーバー上**で打つ。

### 1. 10:00〜10:05　ログインと分担

| 担当 | 最初の15分 |
|---|---|
| A（インフラ） | この手順の 2〜5 を進める |
| B（アプリ） | アプリのコード、テーブル定義（`webapp/sql/` など）、言語切り替え手順を読む |
| C（仕様） | マニュアルと規約を読み、スコア計算式・減点・反映の猶予・キャッシュの可否をメモして共有する |

マニュアルの手順で EC2 を起動し、ポータルに出た IP を渡して `~/.ssh/config` を書く。

```bash
./scripts/ssh-hosts.sh <s1のIP> <s2のIP> <s3のIP>
```

- `~/.ssh/config` の末尾に s1〜s3 の設定を書く。2回目以降は前回の分を入れ替え、既存の設定には触らない。実行のたびに `~/.ssh/config.bak-isucon` にバックアップを取る
- エージェント転送（サーバー上の `git push` で手元の鍵を使う）、s1 の pprof 用ポート転送、接続の使い回しを設定する
- ユーザー名が `isucon` 以外の回は `ISU_USER=ubuntu ./scripts/ssh-hosts.sh ...` のように指定する
- `~/.ssh/config` を TextEdit で開いて編集しない。中身が消えて保存された例がある

3台とも入れることを確認する。

```bash
for h in s1 s2 s3; do ssh $h 'hostname && sudo -n true && echo sudo-ok && ssh -T git@github.com'; done
```

`sudo-ok` と `Hi <ユーザー名>!` が3台分出れば OK。

### 2. 10:05〜10:10　サーバーの中身を確認する（A）

3台とも同じとは限らない。ISUCON10 では台ごとのスペック差に気付いたかどうかで差がついた。詳しい情報は手順3の `task collect` でまとめて取るので、ここでは次の項目だけ確認する。

```
nproc; free -h
systemctl list-units --type=service --state=running | grep -Ei 'isu|nginx|mysql|maria|redis|postgres|h2o|envoy'
systemctl cat isu-go.service
```

- [ ] CPU コア数とメモリ（台ごと）
- [ ] 動いているアプリのサービス名（例: `isu-ruby.service`）と、Go 版のサービス名
- [ ] DB の種類（MariaDB なら設定の置き場所が `mariadb.conf.d/` になる）
- [ ] リバースプロキシが nginx かどうか（h2o や envoy の回もある）

`systemctl cat` の結果から Taskfile.yml に書く値を拾う。

- `WorkingDirectory=` → `APP_DIR`
- `ExecStart=` のバイナリ名 → `APP_BIN`
- `EnvironmentFile=` → DB のパスワードが書いてあるファイル（手順3のバックアップで使う）

### 3. 10:10〜10:20　s1 をリポジトリ管理にする（A）

まずホームに巨大なファイルがないか確認する。初期データの SQL や画像が数百MBあることがある。

```
du -sh ~/* ~/webapp/* 2>/dev/null | sort -h | tail -15
```

テンプレートを取り込む。

```
cd ~ && git init -b main && git remote add origin git@github.com:<you>/<repo>.git && git pull origin main
./scripts/setup.sh s1
task get-conf
task collect
```

- `./scripts/setup.sh s1`: alp・slp・pt-query-digest・task を入れ、`~/.server_id` に `s1` を書く
- `task get-conf`: `/etc/nginx` と `/etc/mysql` を `s1/etc/` に、アプリの環境変数ファイル（例: `~/env.sh`）を `s1/home/` にコピーする
- `task collect`: スペック・サービス・ポート・nginx の実効設定・MySQL の主要変数・テーブルの件数と定義を `s1/collect.md` にまとめる。Claude にはこのファイルを読ませる。パスワードらしき値は伏せ字になる

Taskfile.yml の先頭3つの変数を手順2で調べた値に書き換え、`.gitignore` の末尾に Go のバイナリ名と巨大なファイルを足す。

```
vi Taskfile.yml
vi .gitignore
git status --short | head -50
```

`git status` に巨大なファイルや不要なファイルが出ていないことを確認してからコミットし、ビルドできることも確かめる。

```
git add -A && git commit -m init && git push -u origin main
task build
```

最後に DB のバックアップを取る。パスワードは `EnvironmentFile` に書いてあるものを使う。

```
mysqldump -uisucon -p --single-transaction --all-databases | gzip > ~/backup.sql.gz
```

**つまずきやすい所**

- `git pull origin main` が「untracked working tree files would be overwritten」で止まる: ホームにテンプレートと同じ名前のファイル（`README.md` など）がある。別名に退避してからやり直す
- `git push` が Permission denied: エージェント転送が効いていない。手元で `ssh-add -l` に鍵が出るか確認する

### 4. 10:15〜10:20　s2・s3 も同じ状態にする（A）

```
cd ~ && git init -b main && git remote add origin git@github.com:<you>/<repo>.git && git fetch && git reset --hard origin/main && git branch -u origin/main
./scripts/setup.sh s2 && task get-conf && task collect && git add s2 && git commit -m "s2 conf" && git push
```

s3 でも `s2` を `s3` に変えて同じことをする。push が重なって失敗したときは、`git pull` してからもう一度 `git push` する。

### 5. 10:20〜10:25　初回ベンチ（A + B）

マニュアルの手順に従って参照実装を Go に切り替える（サービス名は手順2のメモを使う）。

```
sudo systemctl disable --now isu-ruby.service
sudo systemctl enable --now isu-go.service
```

ブラウザでトップページが開けることを確認してから、ポータルでベンチを回す。結果が出るまで数分かかる。出たら s1 で記録する。

```
task score S=<点数> M="初期 Go"
```

`task score` はいつも s1 で実行する（`~/scores.log` はサーバーごとに別）。

### 6. 10:25〜10:35　計測を入れる（A）

ここで入れるのは計測だけ。MySQL の高速化設定などはまだ入れない（変化の原因が分からなくなるため）。

**6-1. nginx のアクセスログを JSON にする**

`s1/etc/nginx/nginx.conf` の `http {` の中に、`snippets/nginx.conf` の先頭にある `log_format json ...` と `access_log ... json;` をコピーする。元から残っている `access_log` があるとログが二重に出る（ISUCON13 の NaruseJun は、二重に出ていたログを終盤に止めて 277k から 468k に伸びた）。

```
grep -rn access_log s1/etc/nginx/
```

`json` 以外の `access_log` 行があればコメントアウトする。

**6-2. MySQL のスロークエリログを全件出す**

`snippets/mysql-isu.cnf` のうち「計測中」の3行だけを使う。MariaDB なら置き場所を `mariadb.conf.d/` に変える。

```
mkdir -p s1/etc/mysql/mysql.conf.d
printf '[mysqld]\nslow_query_log = 1\nslow_query_log_file = /var/log/mysql/mysql-slow.log\nlong_query_time = 0\n' > s1/etc/mysql/mysql.conf.d/zz-isu.cnf
```

**6-3. アプリに pprof を入れる**

`snippets/go/pprof.go` をアプリにコピーし、パッケージ名を `main` に変えて、`main()` の最初の行に `StartPprof()` を足す。

```
cp snippets/go/pprof.go webapp/go/pprof.go
sed -i 's/^package isuutil/package main/' webapp/go/pprof.go
vi webapp/go/main.go
```

**6-4. 反映して、計測が動いているか確かめる**

```
git add -A && git commit -m "計測を入れる" && git push
task deploy
```

`task deploy` は、設定のコピー → ビルド → 再起動（nginx は先に `nginx -t` で文法チェック）→ ログを空にする、までを行う。計測が動いているかを3つとも確認する。

```
curl -s -o /dev/null localhost/ ; sudo tail -1 /var/log/nginx/access.log
sudo mysql -e "SHOW VARIABLES LIKE 'slow_query_log%'; SHOW VARIABLES LIKE 'long_query_time';"
curl -s localhost:6060/debug/pprof/ | head -3
```

- アクセスログの最後の行が `{"time":...}` の JSON になっている
- `slow_query_log` が `ON`、`long_query_time` が `0.000000`
- pprof のページの HTML が返ってくる

**つまずきやすい所**

- `task deploy` が `nginx -t` で止まる: `log_format` を `http {` の外に書いている
- MySQL が起動しない: `sudo journalctl -u mysql -n 30` で原因を見る。`/var/log/mysql` が無いか、権限が無いことが多い
- pprof が返らない: `StartPprof()` より前に `main()` がブロックしている（`e.Start(...)` より前に書く）

### 7. 10:35〜10:45　計測付きベンチと共有（A + 全員）

**ベンチの直前に**、別のターミナルで `ssh s1` して pprof の取得を始める（60秒間 CPU プロファイルを取る）。

```
task pprof
```

同じくベンチの直前に、手元から3台の CPU・メモリの記録を同時に始める（75秒間。ベンチより少し長めに取る）。

```bash
for h in s1 s2 s3; do ssh $h 'cd ~ && task monitor' > /tmp/resource-$h.txt & done; wait; cat /tmp/resource-s*.txt
```

ポータルでベンチを回し、結果が出たら集計して記録する。

```
task analyze
task score S=<点数> M="計測あり"
```

`task analyze` は alp・slp・pt-query-digest の結果をターミナルに出し、`/tmp/alp.txt`・`/tmp/slp.txt`・`/tmp/slow.txt` にも保存する（Claude にはこのファイルを読ませる）。pprof の結果は手元のブラウザで http://localhost:1080 を開くと見られる。

ID を含む URL（`/api/user/123` など）が別々の行に分かれていたら、`scripts/alp.yml` の `matching_groups` に正規表現を書いて集計し直す。ログは次の `task deploy` まで残っているので、ベンチを回し直す必要はない。

```
vi scripts/alp.yml
task alp
```

**結果の読み方**

- **alp**: `SUM`（合計時間）の大きい順。上から5つが改善の候補。`COUNT` が多いのに `AVG` が小さいものは、キャッシュや静的配信の候補
- **slp**: クエリを合計時間の大きい順に1行ずつ並べた表。まずここで上位5つを見る。`AVG(ROWSEXAMINED)` が `AVG(ROWSSENT)` より桁違いに大きいクエリはインデックスが効いていない可能性が高い。`COUNT` が異常に多いものは N+1 の可能性が高い
- **pt-query-digest**: slp で気になったクエリの詳細を見るときに使う。`Profile` 表の `Rank` から該当クエリを探すと、実際のクエリ例（そのまま `EXPLAIN` に使える）と対象テーブル名が載っている
- **pprof**: `Top` 表示で、アプリの CPU を使っている関数を確認する。DB 待ちが長い場合は、ここではあまり目立たない
- **CPU・メモリ（`task monitor`）**: サーバーごとに、全体の CPU 使用率・iowait・メモリと、プロセス別の CPU（100% = 1コア分）が出る
  - 全体の CPU が張り付いている（80% 以上）: いちばん使っているプロセスがボトルネック。mysqld ならクエリ・インデックスの改善か DB の別サーバー化、アプリなら pprof で重い関数を探すか台数を増やす、nginx なら静的配信・gzip・keepalive を見直す
  - CPU に余裕があるのにスコアが伸びない: 待ちが原因。ロック待ち、外部 API、アプリ内の sleep やリトライ間隔、コネクション数の上限（`MaxOpenConns`）、スコア計算式側の制約を疑う
  - iowait が高い: ディスク待ち。`innodb_buffer_pool_size` の不足や、binlog・flush の書き込みを疑う
  - swap を使っている、メモリが9割を超える: buffer pool やアプリのキャッシュを持ちすぎている
  - サーバー間で差が大きい: 空いているサーバーに処理を移す（DB の分離、重いエンドポイントの振り分け）

**チームで共有する（10:45）**

3人で次の3点を確認し、担当を割り振る。

1. alp の上位5エンドポイント
2. slp の上位5クエリと、それを呼んでいるエンドポイントの対応
3. C がマニュアルから見つけたスコア計算式と、使えそうな猶予・抜け道

**10:45 時点のチェックリスト**

- [ ] 3台ともリポジトリ管理になっていて、`task deploy` が通る
- [ ] DB のバックアップ（`~/backup.sql.gz`）がある
- [ ] Go 版でベンチが通り、初期スコアを `task score` で記録した
- [ ] アクセスログが JSON で1本だけ出ている
- [ ] スロークエリログが全件出ている
- [ ] pprof が取れる
- [ ] `task analyze` の結果をチームで見て、担当を決めた

---

## 改善のループ（10:45〜17:00）

```
変更を1つ → push → (手元で) ./scripts/deploy-all.sh → ベンチ → task analyze → task score S=<点数> M="何をしたか"
```

- 1回のベンチで変えるのは1つだけ
- インデックスやカラムの追加は、`/initialize` で毎回流れるスキーマ用 SQL に書く（手で `ALTER TABLE` しただけだと次の `/initialize` で消える）
- DB の分離など大きな変更の後は、その場で `./scripts/reboot-all.sh` → ベンチで再起動試験をする
- スコアが下がったら原因を探さず `git revert`
- DB を s2 に分けるときは、s2 の `zz-isu.cnf` に `bind-address = 0.0.0.0` を書き、DB ユーザーに他ホストからの接続権限を付け、`s1/home/` などの環境変数ファイルでアプリの接続先を s2 のプライベート IP に向ける。s2 のアプリは `sudo systemctl disable --now isu-go.service` で止める（`task restart` は enabled なサービスだけ再起動する）

## 終盤（17:00〜18:00）

- `access_log off;`、`slow_query_log = 0`、アプリの systemd に `Environment=NO_PPROF=1`、アプリのロガー停止
- `snippets/mysql-isu.cnf` の耐久性設定が入っていることを確認
- `./scripts/reboot-all.sh` で全台再起動 → ベンチが通るか確認 → **ブラウザで画面を確認**
- 残り30分でコード凍結

---

## Go スニペット (`snippets/go/`)

アプリに**ファイルごとコピー**して package 名を合わせる。

| ファイル | 中身 |
|---|---|
| db.go | `interpolateParams`・コネクションプール設定済みの接続、bulk insert / IN 句の書き方 |
| cache.go | TTL 付きジェネリックキャッシュ + singleflight (同時ミスを1回のロードに) |
| id.go | 複数台で衝突しないアプリ側 ID 採番 (`SELECT MAX(id)` から再開) |
| pprof.go | pprof を :6060 で起動、`NO_PPROF=1` で無効 |

`cd snippets/go && go test -race ./...` でテストが通る (Go 1.22 以上の機能を使用)。
