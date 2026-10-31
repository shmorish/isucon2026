# ip-192-168-4-10 2026-10-31 01:17:04

## スペック

```
2
CPU(s):                                  2
Model name:                              Intel(R) Xeon(R) Platinum 8275CL CPU @ 3.00GHz
               total        used        free      shared  buff/cache   available
Mem:           3.7Gi       655Mi       130Mi       142Mi       3.3Gi       3.1Gi
Swap:             0B          0B          0B
Filesystem      Size  Used Avail Use% Mounted on
/dev/root        58G   24G   34G  42% /
7.0.0-1014-aws
PRETTY_NAME="Ubuntu 24.04.5 LTS"
```

## 動いているサービス

```
  ModemManager.service                           loaded active running Modem Manager
  postgresql@16-main.service                     loaded active running PostgreSQL Cluster 16-main
```

## isu 関連の unit 一覧

```
envcheck-baseline.service                      enabled         enabled
envcheck.service                               enabled         enabled
isucon-register.service                        enabled         enabled
isureal-go.service                             disabled        enabled
isureal-guard.service                          disabled        enabled
isureal-nodejs.service                         disabled        enabled
isureal-perl.service                           disabled        enabled
isureal-php.service                            disabled        enabled
isureal-python.service                         disabled        enabled
isureal-ruby.service                           disabled        enabled
isureal-rust.service                           disabled        enabled
nginx.service                                  disabled        enabled
postgresql.service                             enabled         enabled
postgresql@.service                            indirect        enabled
```

## isureal-go の unit 定義

```
# /etc/systemd/system/isureal-go.service
[Unit]
Description=isureal-go
After=network-online.target
Wants=network-online.target

[Service]
User=isucon
Group=isucon
WorkingDirectory=/home/isucon/webapp/go
EnvironmentFile=/home/isucon/env.sh
ExecStart=/home/isucon/webapp/go/isureal
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## listen しているポート

```
State  Recv-Q Send-Q Local Address:Port Peer Address:PortProcess                                                
LISTEN 0      4096         0.0.0.0:22        0.0.0.0:*    users:(("sshd",pid=682,fd=3),("systemd",pid=1,fd=177))
LISTEN 0      4096      127.0.0.54:53        0.0.0.0:*    users:(("systemd-resolve",pid=340,fd=17))             
LISTEN 0      4096   127.0.0.53%lo:53        0.0.0.0:*    users:(("systemd-resolve",pid=340,fd=15))             
LISTEN 0      200          0.0.0.0:5432      0.0.0.0:*    users:(("postgres",pid=1186,fd=6))                    
LISTEN 0      4096            [::]:22           [::]:*    users:(("sshd",pid=682,fd=4),("systemd",pid=1,fd=183))
LISTEN 0      200             [::]:5432         [::]:*    users:(("postgres",pid=1186,fd=7))                    
```

## バージョン

```
nginx version: nginx/1.24.0 (Ubuntu)
perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
psql (PostgreSQL) 16.15 (Ubuntu 16.15-0ubuntu0.24.04.1)
go
node
perl
php
ruby
uv
bash: line 1: go: command not found
```

## ホーム / webapp の構成

```
total 108
drwxr-xr-x 20 isucon isucon  4096 Oct 30 12:21 .
drwxr-xr-x  5 root   root    4096 Oct 31 01:10 ..
drwxrwxr-x  3 isucon isucon  4096 Oct 30 11:52 .ansible
-rw-r--r--  1 isucon isucon   220 Mar 31  2024 .bash_logout
-rw-r--r--  1 isucon isucon  3771 Mar 31  2024 .bashrc
drwxrwxr-x  3 isucon isucon  4096 Oct 30 11:59 .bundle
drwxrwxr-x  5 isucon isucon  4096 Oct 31 01:13 .cache
drwxrwxr-x  4 isucon isucon  4096 Oct 30 11:59 .cargo
drwxr-xr-x  2 isucon isucon  4096 Oct 30 12:21 .claude
drwxr-xr-x  2 isucon isucon  4096 Oct 30 12:21 .codex
drwxrwxr-x  5 isucon isucon  4096 Oct 30 12:21 .config
drwxr-xr-x  2 isucon isucon  4096 Oct 30 12:21 .copilot
drwxr-xr-x  2 isucon isucon  4096 Oct 30 12:21 .gemini
drwxrwxr-x  4 isucon isucon  4096 Oct 30 11:54 .local
drwxrwxr-x  4 isucon isucon  4096 Oct 30 12:01 .npm
drwxrwxr-x  5 isucon isucon  4096 Oct 30 12:00 .perl-cpm
drwxr-xr-x  3 isucon isucon  4096 Oct 30 12:21 .pi
-rw-r--r--  1 isucon isucon  1101 Oct 30 11:59 .profile
drwxrwxr-x  6 isucon isucon  4096 Oct 30 11:54 .rustup
drwx------  2 isucon isucon  4096 Oct 31 01:10 .ssh
-rw-r--r--  1 isucon isucon 10940 Oct 30 12:21 AGENTS.md
-rw-r--r--  1 isucon isucon   143 Oct 30 11:51 env.sh
drwxrwxr-x  3 isucon isucon  4096 Oct 30 11:59 go
drwxr-xr-x  8 isucon isucon  4096 Oct 30 11:58 local
drwxr-xr-x 11 isucon isucon  4096 Oct 30 12:21 webapp
/home/isucon/webapp
/home/isucon/webapp/._go
/home/isucon/webapp/._nodejs
/home/isucon/webapp/._perl
/home/isucon/webapp/._php
/home/isucon/webapp/._python
/home/isucon/webapp/._ruby
/home/isucon/webapp/._rust
/home/isucon/webapp/._sql
/home/isucon/webapp/AGENTS.md
/home/isucon/webapp/go
/home/isucon/webapp/go/._Makefile
/home/isucon/webapp/go/._account_deletion_api.go
/home/isucon/webapp/go/._action_log.go
/home/isucon/webapp/go/._admin_api.go
/home/isucon/webapp/go/._admin_posting_window_api.go
/home/isucon/webapp/go/._admin_session.go
/home/isucon/webapp/go/._admin_user_api.go
/home/isucon/webapp/go/._api_error.go
/home/isucon/webapp/go/._cmd
/home/isucon/webapp/go/._comment.go
/home/isucon/webapp/go/._comment_api.go
/home/isucon/webapp/go/._config.go
/home/isucon/webapp/go/._export_api.go
/home/isucon/webapp/go/._feed_api.go
/home/isucon/webapp/go/._friend_request_api.go
/home/isucon/webapp/go/._go.mod
/home/isucon/webapp/go/._go.sum
/home/isucon/webapp/go/._image_api.go
/home/isucon/webapp/go/._image_signature.go
/home/isucon/webapp/go/._initialize.go
/home/isucon/webapp/go/._ip_ban.go
/home/isucon/webapp/go/._memory_api.go
/home/isucon/webapp/go/._notification_api.go
/home/isucon/webapp/go/._post.go
/home/isucon/webapp/go/._post_api.go
/home/isucon/webapp/go/._posting_window.go
/home/isucon/webapp/go/._posting_window_api.go
/home/isucon/webapp/go/._profile_image_api.go
/home/isucon/webapp/go/._realmoji.go
/home/isucon/webapp/go/._realmoji_api.go
/home/isucon/webapp/go/._region.go
/home/isucon/webapp/go/._relationship.go
/home/isucon/webapp/go/._relationship_api.go
/home/isucon/webapp/go/._relationship_list_api.go
/home/isucon/webapp/go/._server.go
/home/isucon/webapp/go/._session.go
/home/isucon/webapp/go/._user.go
/home/isucon/webapp/go/._user_api.go
/home/isucon/webapp/go/._user_read_api.go
/home/isucon/webapp/go/AGENTS.md
/home/isucon/webapp/go/Makefile
/home/isucon/webapp/go/account_deletion_api.go
/home/isucon/webapp/go/action_log.go
/home/isucon/webapp/go/admin_api.go
/home/isucon/webapp/go/admin_posting_window_api.go
/home/isucon/webapp/go/admin_session.go
/home/isucon/webapp/go/admin_user_api.go
/home/isucon/webapp/go/api_error.go
/home/isucon/webapp/go/cmd
/home/isucon/webapp/go/comment.go
/home/isucon/webapp/go/comment_api.go
/home/isucon/webapp/go/config.go
/home/isucon/webapp/go/export_api.go
/home/isucon/webapp/go/feed_api.go
/home/isucon/webapp/go/friend_request_api.go
/home/isucon/webapp/go/go.mod
/home/isucon/webapp/go/go.sum
/home/isucon/webapp/go/image_api.go
/home/isucon/webapp/go/image_signature.go
/home/isucon/webapp/go/initialize.go
/home/isucon/webapp/go/ip_ban.go
/home/isucon/webapp/go/isureal
/home/isucon/webapp/go/memory_api.go
/home/isucon/webapp/go/notification_api.go
/home/isucon/webapp/go/post.go
/home/isucon/webapp/go/post_api.go
/home/isucon/webapp/go/posting_window.go
/home/isucon/webapp/go/posting_window_api.go
/home/isucon/webapp/go/profile_image_api.go
/home/isucon/webapp/go/realmoji.go
/home/isucon/webapp/go/realmoji_api.go
/home/isucon/webapp/go/region.go
/home/isucon/webapp/go/relationship.go
/home/isucon/webapp/go/relationship_api.go
/home/isucon/webapp/go/relationship_list_api.go
/home/isucon/webapp/go/server.go
/home/isucon/webapp/go/session.go
/home/isucon/webapp/go/user.go
/home/isucon/webapp/go/user_api.go
/home/isucon/webapp/go/user_read_api.go
/home/isucon/webapp/nodejs
/home/isucon/webapp/nodejs/._.gitignore
/home/isucon/webapp/nodejs/._README.md
/home/isucon/webapp/nodejs/._mise.toml
/home/isucon/webapp/nodejs/._package-lock.json
/home/isucon/webapp/nodejs/._package.json
/home/isucon/webapp/nodejs/._src
/home/isucon/webapp/nodejs/._tsconfig.json
/home/isucon/webapp/nodejs/AGENTS.md
/home/isucon/webapp/nodejs/README.md
/home/isucon/webapp/nodejs/mise.toml
/home/isucon/webapp/nodejs/package-lock.json
/home/isucon/webapp/nodejs/package.json
/home/isucon/webapp/nodejs/src
/home/isucon/webapp/nodejs/tsconfig.json
/home/isucon/webapp/perl
/home/isucon/webapp/perl/._.gitignore
/home/isucon/webapp/perl/._Makefile
/home/isucon/webapp/perl/._README.md
/home/isucon/webapp/perl/._cpanfile
/home/isucon/webapp/perl/._cpanfile.snapshot
/home/isucon/webapp/perl/._lib
/home/isucon/webapp/perl/._script
/home/isucon/webapp/perl/AGENTS.md
/home/isucon/webapp/perl/Makefile
/home/isucon/webapp/perl/README.md
/home/isucon/webapp/perl/cpanfile
/home/isucon/webapp/perl/cpanfile.snapshot
/home/isucon/webapp/perl/lib
/home/isucon/webapp/perl/local
/home/isucon/webapp/perl/script
/home/isucon/webapp/php
/home/isucon/webapp/php/._.gitignore
/home/isucon/webapp/php/._Makefile
/home/isucon/webapp/php/._README.md
/home/isucon/webapp/php/._app
/home/isucon/webapp/php/._composer.json
/home/isucon/webapp/php/._composer.lock
/home/isucon/webapp/php/._php-fpm.conf
/home/isucon/webapp/php/._phpcs.xml
/home/isucon/webapp/php/._phpstan.neon.dist
/home/isucon/webapp/php/._phpunit.xml.dist
/home/isucon/webapp/php/._public
/home/isucon/webapp/php/._src
/home/isucon/webapp/php/AGENTS.md
/home/isucon/webapp/php/Makefile
/home/isucon/webapp/php/README.md
/home/isucon/webapp/php/app
/home/isucon/webapp/php/composer.json
/home/isucon/webapp/php/composer.lock
/home/isucon/webapp/php/php-fpm.conf
/home/isucon/webapp/php/phpcs.xml
/home/isucon/webapp/php/phpstan.neon.dist
/home/isucon/webapp/php/phpunit.xml.dist
/home/isucon/webapp/php/public
/home/isucon/webapp/php/src
/home/isucon/webapp/php/vendor
/home/isucon/webapp/public
/home/isucon/webapp/public/._.
/home/isucon/webapp/public/._assets
/home/isucon/webapp/public/._index.html
/home/isucon/webapp/public/._isureal-logo.svg
/home/isucon/webapp/public/assets
/home/isucon/webapp/public/index.html
/home/isucon/webapp/public/isureal-logo.svg
/home/isucon/webapp/python
/home/isucon/webapp/python/._.gitignore
/home/isucon/webapp/python/._.python-version
/home/isucon/webapp/python/._Makefile
/home/isucon/webapp/python/._README.md
/home/isucon/webapp/python/._gunicorn.conf.py
/home/isucon/webapp/python/._isureal
/home/isucon/webapp/python/._pyproject.toml
/home/isucon/webapp/python/._uv.lock
/home/isucon/webapp/python/.python-version
/home/isucon/webapp/python/.venv
/home/isucon/webapp/python/AGENTS.md
/home/isucon/webapp/python/Makefile
/home/isucon/webapp/python/README.md
/home/isucon/webapp/python/gunicorn.conf.py
/home/isucon/webapp/python/isureal
/home/isucon/webapp/python/pyproject.toml
/home/isucon/webapp/python/uv.lock
/home/isucon/webapp/ruby
/home/isucon/webapp/ruby/._.gitignore
/home/isucon/webapp/ruby/._.ruby-version
/home/isucon/webapp/ruby/._Gemfile
/home/isucon/webapp/ruby/._Gemfile.lock
/home/isucon/webapp/ruby/._Makefile
/home/isucon/webapp/ruby/._README.md
/home/isucon/webapp/ruby/._config
/home/isucon/webapp/ruby/._config.ru
/home/isucon/webapp/ruby/._lib
/home/isucon/webapp/ruby/.bundle
/home/isucon/webapp/ruby/.ruby-version
/home/isucon/webapp/ruby/AGENTS.md
/home/isucon/webapp/ruby/Gemfile
/home/isucon/webapp/ruby/Gemfile.lock
/home/isucon/webapp/ruby/Makefile
/home/isucon/webapp/ruby/README.md
/home/isucon/webapp/ruby/config
/home/isucon/webapp/ruby/config.ru
/home/isucon/webapp/ruby/lib
/home/isucon/webapp/ruby/vendor
/home/isucon/webapp/rust
/home/isucon/webapp/rust/._.gitignore
/home/isucon/webapp/rust/._Cargo.lock
/home/isucon/webapp/rust/._Cargo.toml
/home/isucon/webapp/rust/._Makefile
/home/isucon/webapp/rust/._README.md
/home/isucon/webapp/rust/._rust-toolchain.toml
/home/isucon/webapp/rust/._src
/home/isucon/webapp/rust/AGENTS.md
/home/isucon/webapp/rust/Cargo.lock
/home/isucon/webapp/rust/Cargo.toml
/home/isucon/webapp/rust/Makefile
/home/isucon/webapp/rust/README.md
/home/isucon/webapp/rust/isureal
/home/isucon/webapp/rust/rust-toolchain.toml
/home/isucon/webapp/rust/src
/home/isucon/webapp/rust/target
/home/isucon/webapp/sql
/home/isucon/webapp/sql/._001-schema.sql
/home/isucon/webapp/sql/._003-initial-snapshot.sql
/home/isucon/webapp/sql/._init.sh
/home/isucon/webapp/sql/._reset.sql
/home/isucon/webapp/sql/001-schema.sql
/home/isucon/webapp/sql/003-initial-snapshot.sql
/home/isucon/webapp/sql/init.sh
/home/isucon/webapp/sql/reset.sql
```

## 大きいファイル

```
3.3G	/home/isucon
1.2G	/home/isucon/local
836M	/home/isucon/webapp
660M	/home/isucon/webapp/rust
649M	/home/isucon/webapp/rust/target/release
649M	/home/isucon/webapp/rust/target
628M	/home/isucon/webapp/rust/target/release/deps
617M	/home/isucon/.rustup/toolchains/1.99.1-x86_64-unknown-linux-gnu
617M	/home/isucon/.rustup/toolchains
617M	/home/isucon/.rustup
529M	/home/isucon/.rustup/toolchains/1.99.1-x86_64-unknown-linux-gnu/lib
297M	/home/isucon/local/ruby
274M	/home/isucon/local/php
269M	/home/isucon/local/go
241M	/home/isucon/local/ruby/lib
230M	/home/isucon/local/node
199M	/home/isucon/local/php/bin
199M	/home/isucon/.rustup/toolchains/1.99.1-x86_64-unknown-linux-gnu/lib/libLLVM.so.23.1-rust-1.99.1-stable
186M	/home/isucon/.cargo
182M	/home/isucon/.perl-cpm
```

## nginx の実効設定 (nginx -T)

```
user www-data;
worker_processes auto;
pid /run/nginx.pid;
error_log /var/log/nginx/error.log;
include /etc/nginx/modules-enabled/*.conf;
events {
	worker_connections 768;
}
http {
	sendfile on;
	tcp_nopush on;
	types_hash_max_size 2048;
	include /etc/nginx/mime.types;
	default_type application/octet-stream;
	ssl_protocols TLSv1 TLSv1.1 TLSv1.2 TLSv1.3; # Dropping SSLv3, ref: POODLE
	ssl_prefer_server_ciphers on;
	access_log /var/log/nginx/access.log;
	gzip on;
	include /etc/nginx/conf.d/*.conf;
	include /etc/nginx/sites-enabled/*;
}
types {
    text/html                                        html htm shtml;
    text/css                                         css;
    text/xml                                         xml;
    image/gif                                        gif;
    image/jpeg                                       jpeg jpg;
    application/javascript                           js;
    application/atom+xml                             atom;
    application/rss+xml                              rss;
    text/mathml                                      mml;
    text/plain                                       txt;
    text/vnd.sun.j2me.app-descriptor                 jad;
    text/vnd.wap.wml                                 wml;
    text/x-component                                 htc;
    image/avif                                       avif;
    image/png                                        png;
    image/svg+xml                                    svg svgz;
    image/tiff                                       tif tiff;
    image/vnd.wap.wbmp                               wbmp;
    image/webp                                       webp;
    image/x-icon                                     ico;
    image/x-jng                                      jng;
    image/x-ms-bmp                                   bmp;
    font/woff                                        woff;
    font/woff2                                       woff2;
    application/java-archive                         jar war ear;
    application/json                                 json;
    application/mac-binhex40                         hqx;
    application/msword                               doc;
    application/pdf                                  pdf;
    application/postscript                           ps eps ai;
    application/rtf                                  rtf;
    application/vnd.apple.mpegurl                    m3u8;
    application/vnd.google-earth.kml+xml             kml;
    application/vnd.google-earth.kmz                 kmz;
    application/vnd.ms-excel                         xls;
    application/vnd.ms-fontobject                    eot;
    application/vnd.ms-powerpoint                    ppt;
    application/vnd.oasis.opendocument.graphics      odg;
    application/vnd.oasis.opendocument.presentation  odp;
    application/vnd.oasis.opendocument.spreadsheet   ods;
    application/vnd.oasis.opendocument.text          odt;
    application/vnd.openxmlformats-officedocument.presentationml.presentation
                                                     pptx;
    application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
                                                     xlsx;
    application/vnd.openxmlformats-officedocument.wordprocessingml.document
                                                     docx;
    application/vnd.wap.wmlc                         wmlc;
    application/wasm                                 wasm;
    application/x-7z-compressed                      7z;
    application/x-cocoa                              cco;
    application/x-java-archive-diff                  jardiff;
    application/x-java-jnlp-file                     jnlp;
    application/x-makeself                           run;
    application/x-perl                               pl pm;
    application/x-pilot                              prc pdb;
    application/x-rar-compressed                     rar;
    application/x-redhat-package-manager             rpm;
    application/x-sea                                sea;
    application/x-shockwave-flash                    swf;
    application/x-stuffit                            sit;
    application/x-tcl                                tcl tk;
    application/x-x509-ca-cert                       der pem crt;
    application/x-xpinstall                          xpi;
    application/xhtml+xml                            xhtml;
    application/xspf+xml                             xspf;
    application/zip                                  zip;
    application/octet-stream                         bin exe dll;
    application/octet-stream                         deb;
    application/octet-stream                         dmg;
    application/octet-stream                         iso img;
    application/octet-stream                         msi msp msm;
    audio/midi                                       mid midi kar;
    audio/mpeg                                       mp3;
    audio/ogg                                        ogg;
    audio/x-m4a                                      m4a;
    audio/x-realaudio                                ra;
    video/3gpp                                       3gpp 3gp;
    video/mp2t                                       ts;
    video/mp4                                        mp4;
    video/mpeg                                       mpeg mpg;
    video/ogg                                        ogv;
    video/quicktime                                  mov;
    video/webm                                       webm;
    video/x-flv                                      flv;
    video/x-m4v                                      m4v;
    video/x-matroska                                 mkv;
    video/x-mng                                      mng;
    video/x-ms-asf                                   asx asf;
    video/x-ms-wmv                                   wmv;
    video/x-msvideo                                  avi;
}
server {
  listen 80 default_server;
  server_name _;
  client_max_body_size 32m;
  root /home/isucon/webapp/public;
  location /api/ {
    proxy_set_header Host $host;
    proxy_pass http://127.0.0.1:8080;
  }
  location / {
    try_files $uri /index.html;
  }
}
```

## PostgreSQL の主要設定

```
perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
            name            |                setting                | unit 
----------------------------+---------------------------------------+------
 checkpoint_timeout         | 300                                   | s
 effective_cache_size       | 524288                                | 8kB
 fsync                      | on                                    | 
 listen_addresses           | *                                     | 
 log_min_duration_statement | -1                                    | ms
 max_connections            | 100                                   | 
 max_wal_size               | 1024                                  | MB
 random_page_cost           | 4                                     | 
 server_version             | 16.15 (Ubuntu 16.15-0ubuntu0.24.04.1) | 
 shared_buffers             | 16384                                 | 8kB
 shared_preload_libraries   |                                       | 
 synchronous_commit         | on                                    | 
 wal_level                  | replica                               | 
 work_mem                   | 4096                                  | kB
(14 rows)

```

## pg_hba.conf (コメント除く)

```
perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
local   all             postgres                                peer
local   all             all                                     peer
host    all             all             127.0.0.1/32            scram-sha-256
host    all             all             ::1/128                 scram-sha-256
local   replication     all                                     peer
host    replication     all             127.0.0.1/32            scram-sha-256
host    replication     all             ::1/128                 scram-sha-256
host isureal isureal 0.0.0.0/0 scram-sha-256
```

## テーブルの件数とサイズ

```
perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
         relname         | n_live_tup |   total    
-------------------------+------------+------------
 image_objects_202605    |     155262 | 1251 MB
 image_objects_202603    |     154867 | 1251 MB
 image_objects_202601    |     154984 | 1249 MB
 image_objects_202608    |     154966 | 1248 MB
 image_objects_202512    |     153901 | 1242 MB
 image_objects_202607    |     153703 | 1242 MB
 image_objects_202609    |     150783 | 1215 MB
 image_objects_202511    |     150273 | 1212 MB
 image_objects_202604    |     150384 | 1211 MB
 image_objects_202606    |     149322 | 1206 MB
 image_objects_202602    |     141545 | 1141 MB
 image_objects_202610    |     135605 | 1090 MB
 notifications           |    1854450 | 645 MB
 realmojis               |    1199348 | 506 MB
 posts                   |     625199 | 211 MB
 comments                |     546011 | 161 MB
 image_objects_202510    |      19054 | 156 MB
 image_objects_20261028  |         89 | 46 MB
 friendships             |      25041 | 10 MB
 friend_requests         |      29093 | 7056 kB
 initial_friendships     |      24985 | 5152 kB
 initial_friend_requests |      28970 | 4776 kB
 follows                 |      10037 | 3720 kB
 users                   |       5043 | 2312 kB
 initial_follows         |      10020 | 1576 kB
 initial_users           |       5000 | 1272 kB
 image_objects_202501    |        214 | 752 kB
 image_objects_202505    |        217 | 712 kB
 image_objects_202503    |        214 | 712 kB
 image_objects_202412    |        218 | 712 kB
 image_objects_202506    |        215 | 704 kB
 image_objects_202508    |        213 | 696 kB
 image_objects_202507    |        209 | 680 kB
 image_objects_202411    |        200 | 680 kB
 image_objects_202509    |        199 | 672 kB
 image_objects_202504    |        198 | 656 kB
 image_objects_202502    |        174 | 608 kB
 posting_windows         |       1113 | 344 kB
 image_objects_202410    |         30 | 168 kB
 block_relations         |        254 | 160 kB
 login_sessions          |        253 | 136 kB
 initial_block_relations |        250 | 64 kB
 administrators          |          1 | 32 kB
 ip_bans                 |         43 | 32 kB
 administrator_sessions  |          1 | 32 kB
 image_objects_202702    |          0 | 16 kB
 image_objects_default   |          0 | 16 kB
 image_objects_202703    |          0 | 16 kB
 image_objects_202612    |          0 | 16 kB
 image_objects_202611    |          0 | 16 kB
 image_objects_202701    |          0 | 16 kB
 image_objects_202704    |          0 | 16 kB
 image_objects_202409    |          0 | 16 kB
 image_objects_202706    |          0 | 16 kB
 image_objects_202705    |          0 | 16 kB
 initial_data_boundary   |          1 | 8192 bytes
 image_objects           |          0 | 0 bytes
(57 rows)

```

## テーブル定義 (\d)

```
perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.administrator_sessions"
      Column      |           Type           | Collation | Nullable |      Default      
------------------+--------------------------+-----------+----------+-------------------
 id               | character varying(36)    |           | not null | 
 administrator_id | character varying(36)    |           | not null | 
 token_hash       | text                     |           | not null | 
 active           | boolean                  |           | not null | true
 created_at       | timestamp with time zone |           | not null | clock_timestamp()
 invalidated_at   | timestamp with time zone |           |          | 
Indexes:
    "administrator_sessions_pkey" PRIMARY KEY, btree (id)
Foreign-key constraints:
    "administrator_sessions_administrator_id_fkey" FOREIGN KEY (administrator_id) REFERENCES administrators(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                            Table "public.administrators"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 username      | character varying(64)    |           | not null | 
 password_hash | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Indexes:
    "administrators_pkey" PRIMARY KEY, btree (id)
Referenced by:
    TABLE "administrator_sessions" CONSTRAINT "administrator_sessions_administrator_id_fkey" FOREIGN KEY (administrator_id) REFERENCES administrators(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                          Table "public.block_relations"
   Column   |           Type           | Collation | Nullable |      Default      
------------+--------------------------+-----------+----------+-------------------
 id         | character varying(36)    |           | not null | 
 blocker_id | character varying(36)    |           | not null | 
 blocked_id | character varying(36)    |           | not null | 
 created_at | timestamp with time zone |           | not null | clock_timestamp()
Indexes:
    "block_relations_pkey" PRIMARY KEY, btree (id)
    "block_relations_blocker_id_blocked_id_key" UNIQUE CONSTRAINT, btree (blocker_id, blocked_id)
Foreign-key constraints:
    "block_relations_blocked_id_fkey" FOREIGN KEY (blocked_id) REFERENCES users(id) ON DELETE RESTRICT
    "block_relations_blocker_id_fkey" FOREIGN KEY (blocker_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                        Table "public.comments"
   Column   |           Type           | Collation | Nullable | Default 
------------+--------------------------+-----------+----------+---------
 id         | character varying(36)    |           | not null | 
 post_id    | character varying(36)    |           | not null | 
 author_id  | character varying(36)    |           | not null | 
 body       | text                     |           | not null | 
 created_at | timestamp with time zone |           | not null | 
 deleted_at | timestamp with time zone |           |          | 
Indexes:
    "comments_pkey" PRIMARY KEY, btree (id)
    "comments_post_id" btree (post_id)
Foreign-key constraints:
    "comments_author_id_fkey" FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT
    "comments_post_id_fkey" FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                              Table "public.follows"
   Column    |           Type           | Collation | Nullable |      Default      
-------------+--------------------------+-----------+----------+-------------------
 id          | character varying(36)    |           | not null | 
 follower_id | character varying(36)    |           | not null | 
 followee_id | character varying(36)    |           | not null | 
 created_at  | timestamp with time zone |           | not null | clock_timestamp()
 updated_at  | timestamp with time zone |           | not null | clock_timestamp()
Indexes:
    "follows_pkey" PRIMARY KEY, btree (id)
    "follows_follower_id_followee_id_key" UNIQUE CONSTRAINT, btree (follower_id, followee_id)
Foreign-key constraints:
    "follows_followee_id_fkey" FOREIGN KEY (followee_id) REFERENCES users(id) ON DELETE RESTRICT
    "follows_follower_id_fkey" FOREIGN KEY (follower_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                      Table "public.friend_requests"
    Column    |           Type           | Collation | Nullable | Default 
--------------+--------------------------+-----------+----------+---------
 id           | character varying(36)    |           | not null | 
 requester_id | character varying(36)    |           | not null | 
 recipient_id | character varying(36)    |           | not null | 
 status       | text                     |           | not null | 
 created_at   | timestamp with time zone |           | not null | 
 updated_at   | timestamp with time zone |           | not null | 
Indexes:
    "friend_requests_pkey" PRIMARY KEY, btree (id)
Foreign-key constraints:
    "friend_requests_recipient_id_fkey" FOREIGN KEY (recipient_id) REFERENCES users(id) ON DELETE RESTRICT
    "friend_requests_requester_id_fkey" FOREIGN KEY (requester_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                               Table "public.friendships"
      Column       |           Type           | Collation | Nullable |      Default      
-------------------+--------------------------+-----------+----------+-------------------
 id                | character varying(36)    |           | not null | 
 user_low_id       | character varying(36)    |           | not null | 
 user_high_id      | character varying(36)    |           | not null | 
 source_request_id | character varying(36)    |           |          | 
 created_at        | timestamp with time zone |           | not null | clock_timestamp()
 updated_at        | timestamp with time zone |           | not null | clock_timestamp()
Indexes:
    "friendships_pkey" PRIMARY KEY, btree (id)
    "friendships_user_low_id_user_high_id_key" UNIQUE CONSTRAINT, btree (user_low_id, user_high_id)
Foreign-key constraints:
    "friendships_user_high_id_fkey" FOREIGN KEY (user_high_id) REFERENCES users(id) ON DELETE RESTRICT
    "friendships_user_low_id_fkey" FOREIGN KEY (user_low_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                      Partitioned table "public.image_objects"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition key: ****
Indexes:
    "image_objects_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT
Number of partitions: 36 (Use \d+ to list them.)

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202409"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2024-09-01 00:00:00+00') TO ('2024-10-01 00:00:00+00')
Indexes:
    "image_objects_202409_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202410"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2024-10-01 00:00:00+00') TO ('2024-11-01 00:00:00+00')
Indexes:
    "image_objects_202410_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202411"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2024-11-01 00:00:00+00') TO ('2024-12-01 00:00:00+00')
Indexes:
    "image_objects_202411_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202412"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2024-12-01 00:00:00+00') TO ('2025-01-01 00:00:00+00')
Indexes:
    "image_objects_202412_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202501"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-01-01 00:00:00+00') TO ('2025-02-01 00:00:00+00')
Indexes:
    "image_objects_202501_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202502"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-02-01 00:00:00+00') TO ('2025-03-01 00:00:00+00')
Indexes:
    "image_objects_202502_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202503"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-03-01 00:00:00+00') TO ('2025-04-01 00:00:00+00')
Indexes:
    "image_objects_202503_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202504"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-04-01 00:00:00+00') TO ('2025-05-01 00:00:00+00')
Indexes:
    "image_objects_202504_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202505"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-05-01 00:00:00+00') TO ('2025-06-01 00:00:00+00')
Indexes:
    "image_objects_202505_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202506"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-06-01 00:00:00+00') TO ('2025-07-01 00:00:00+00')
Indexes:
    "image_objects_202506_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202507"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-07-01 00:00:00+00') TO ('2025-08-01 00:00:00+00')
Indexes:
    "image_objects_202507_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202508"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-08-01 00:00:00+00') TO ('2025-09-01 00:00:00+00')
Indexes:
    "image_objects_202508_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202509"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-09-01 00:00:00+00') TO ('2025-10-01 00:00:00+00')
Indexes:
    "image_objects_202509_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202510"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-10-01 00:00:00+00') TO ('2025-11-01 00:00:00+00')
Indexes:
    "image_objects_202510_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202511"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-11-01 00:00:00+00') TO ('2025-12-01 00:00:00+00')
Indexes:
    "image_objects_202511_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202512"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2025-12-01 00:00:00+00') TO ('2026-01-01 00:00:00+00')
Indexes:
    "image_objects_202512_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202601"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-01-01 00:00:00+00') TO ('2026-02-01 00:00:00+00')
Indexes:
    "image_objects_202601_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202602"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-02-01 00:00:00+00') TO ('2026-03-01 00:00:00+00')
Indexes:
    "image_objects_202602_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202603"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-03-01 00:00:00+00') TO ('2026-04-01 00:00:00+00')
Indexes:
    "image_objects_202603_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202604"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-04-01 00:00:00+00') TO ('2026-05-01 00:00:00+00')
Indexes:
    "image_objects_202604_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202605"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-05-01 00:00:00+00') TO ('2026-06-01 00:00:00+00')
Indexes:
    "image_objects_202605_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202606"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-06-01 00:00:00+00') TO ('2026-07-01 00:00:00+00')
Indexes:
    "image_objects_202606_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202607"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-07-01 00:00:00+00') TO ('2026-08-01 00:00:00+00')
Indexes:
    "image_objects_202607_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202608"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-08-01 00:00:00+00') TO ('2026-09-01 00:00:00+00')
Indexes:
    "image_objects_202608_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202609"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-09-01 00:00:00+00') TO ('2026-10-01 00:00:00+00')
Indexes:
    "image_objects_202609_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202610"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-10-01 00:00:00+00') TO ('2026-10-28 06:00:00+00')
Indexes:
    "image_objects_202610_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                        Table "public.image_objects_20261028"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-10-28 06:00:00+00') TO ('2026-11-01 00:00:00+00')
Indexes:
    "image_objects_20261028_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202611"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-11-01 00:00:00+00') TO ('2026-12-01 00:00:00+00')
Indexes:
    "image_objects_202611_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202612"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2026-12-01 00:00:00+00') TO ('2027-01-01 00:00:00+00')
Indexes:
    "image_objects_202612_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202701"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2027-01-01 00:00:00+00') TO ('2027-02-01 00:00:00+00')
Indexes:
    "image_objects_202701_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202702"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2027-02-01 00:00:00+00') TO ('2027-03-01 00:00:00+00')
Indexes:
    "image_objects_202702_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202703"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2027-03-01 00:00:00+00') TO ('2027-04-01 00:00:00+00')
Indexes:
    "image_objects_202703_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202704"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2027-04-01 00:00:00+00') TO ('2027-05-01 00:00:00+00')
Indexes:
    "image_objects_202704_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202705"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2027-05-01 00:00:00+00') TO ('2027-06-01 00:00:00+00')
Indexes:
    "image_objects_202705_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.image_objects_202706"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects FOR VALUES FROM ('2027-06-01 00:00:00+00') TO ('2027-07-01 00:00:00+00')
Indexes:
    "image_objects_202706_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                        Table "public.image_objects_default"
    Column     |           Type           | Collation | Nullable |      Default      
---------------+--------------------------+-----------+----------+-------------------
 id            | character varying(36)    |           | not null | 
 owner_user_id | character varying(36)    |           | not null | 
 data          | bytea                    |           | not null | 
 media_type    | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | clock_timestamp()
Partition of: image_objects DEFAULT
Indexes:
    "image_objects_default_pkey" PRIMARY KEY, btree (id, created_at)
Foreign-key constraints:
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                 Table "public.initial_block_relations"
   Column   |           Type           | Collation | Nullable | Default 
------------+--------------------------+-----------+----------+---------
 id         | character varying(36)    |           |          | 
 blocker_id | character varying(36)    |           |          | 
 blocked_id | character varying(36)    |           |          | 
 created_at | timestamp with time zone |           |          | 

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                  Table "public.initial_data_boundary"
   Column   |           Type           | Collation | Nullable | Default 
------------+--------------------------+-----------+----------+---------
 created_at | timestamp with time zone |           |          | 

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                     Table "public.initial_follows"
   Column    |           Type           | Collation | Nullable | Default 
-------------+--------------------------+-----------+----------+---------
 id          | character varying(36)    |           |          | 
 follower_id | character varying(36)    |           |          | 
 followee_id | character varying(36)    |           |          | 
 created_at  | timestamp with time zone |           |          | 
 updated_at  | timestamp with time zone |           |          | 

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                  Table "public.initial_friend_requests"
    Column    |           Type           | Collation | Nullable | Default 
--------------+--------------------------+-----------+----------+---------
 id           | character varying(36)    |           |          | 
 requester_id | character varying(36)    |           |          | 
 recipient_id | character varying(36)    |           |          | 
 status       | text                     |           |          | 
 created_at   | timestamp with time zone |           |          | 
 updated_at   | timestamp with time zone |           |          | 

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                      Table "public.initial_friendships"
      Column       |           Type           | Collation | Nullable | Default 
-------------------+--------------------------+-----------+----------+---------
 id                | character varying(36)    |           |          | 
 user_low_id       | character varying(36)    |           |          | 
 user_high_id      | character varying(36)    |           |          | 
 source_request_id | character varying(36)    |           |          | 
 created_at        | timestamp with time zone |           |          | 
 updated_at        | timestamp with time zone |           |          | 

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.initial_users"
      Column      |           Type           | Collation | Nullable | Default 
------------------+--------------------------+-----------+----------+---------
 id               | character varying(36)    |           |          | 
 username         | character varying(64)    |           |          | 
 secret_key_hash  | text                     |           |          | 
 display_name     | text                     |           |          | 
 bio              | text                     |           |          | 
 profile_image_id | character varying(36)    |           |          | 
 region           | text                     |           |          | 
 account_type     | text                     |           |          | 
 status           | text                     |           |          | 
 deleted_at       | timestamp with time zone |           |          | 
 created_at       | timestamp with time zone |           |          | 
 updated_at       | timestamp with time zone |           |          | 

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                               Table "public.ip_bans"
    Column    |           Type           | Collation | Nullable |      Default      
--------------+--------------------------+-----------+----------+-------------------
 ip           | text                     |           | not null | 
 banned_until | timestamp with time zone |           | not null | 
 created_at   | timestamp with time zone |           | not null | clock_timestamp()
Indexes:
    "ip_bans_pkey" PRIMARY KEY, btree (ip)

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                            Table "public.login_sessions"
     Column     |           Type           | Collation | Nullable |      Default      
----------------+--------------------------+-----------+----------+-------------------
 id             | character varying(36)    |           | not null | 
 user_id        | character varying(36)    |           | not null | 
 token_hash     | text                     |           | not null | 
 active         | boolean                  |           | not null | true
 created_at     | timestamp with time zone |           | not null | clock_timestamp()
 invalidated_at | timestamp with time zone |           |          | 
 reason         | text                     |           |          | 
Indexes:
    "login_sessions_pkey" PRIMARY KEY, btree (id)
Foreign-key constraints:
    "login_sessions_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.notifications"
      Column       |           Type           | Collation | Nullable | Default 
-------------------+--------------------------+-----------+----------+---------
 id                | character varying(36)    |           | not null | 
 user_id           | character varying(36)    |           | not null | 
 posting_window_id | character varying(36)    |           | not null | 
 created_at        | timestamp with time zone |           | not null | 
Indexes:
    "notifications_pkey" PRIMARY KEY, btree (id)
    "notifications_user_id_posting_window_id_key" UNIQUE CONSTRAINT, btree (user_id, posting_window_id)
Foreign-key constraints:
    "notifications_posting_window_id_fkey" FOREIGN KEY (posting_window_id) REFERENCES posting_windows(id) ON DELETE RESTRICT
    "notifications_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                             Table "public.posting_windows"
      Column      |           Type           | Collation | Nullable |      Default      
------------------+--------------------------+-----------+----------+-------------------
 id               | character varying(36)    |           | not null | 
 region           | text                     |           | not null | 
 local_date       | date                     |           | not null | 
 starts_at        | timestamp with time zone |           | not null | 
 on_time_deadline | timestamp with time zone |           | not null | 
 ended_at         | timestamp with time zone |           |          | 
 created_at       | timestamp with time zone |           | not null | clock_timestamp()
Indexes:
    "posting_windows_pkey" PRIMARY KEY, btree (id)
    "posting_windows_one_active_region" UNIQUE, btree (region) WHERE ended_at IS NULL
    "posting_windows_region_local_date_key" UNIQUE CONSTRAINT, btree (region, local_date)
Referenced by:
    TABLE "notifications" CONSTRAINT "notifications_posting_window_id_fkey" FOREIGN KEY (posting_window_id) REFERENCES posting_windows(id) ON DELETE RESTRICT
    TABLE "posts" CONSTRAINT "posts_posting_window_id_fkey" FOREIGN KEY (posting_window_id) REFERENCES posting_windows(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                                  Table "public.posts"
      Column       |           Type           | Collation | Nullable |      Default      
-------------------+--------------------------+-----------+----------+-------------------
 id                | character varying(36)    |           | not null | 
 author_id         | character varying(36)    |           | not null | 
 posting_window_id | character varying(36)    |           | not null | 
 image_id          | character varying(36)    |           | not null | 
 caption           | text                     |           | not null | ''::text
 visibility        | text                     |           | not null | 
 location_kind     | text                     |           | not null | 
 latitude          | numeric                  |           |          | 
 longitude         | numeric                  |           |          | 
 locality          | text                     |           |          | 
 posted_at         | timestamp with time zone |           | not null | 
 created_at        | timestamp with time zone |           | not null | clock_timestamp()
 deleted_at        | timestamp with time zone |           |          | 
Indexes:
    "posts_pkey" PRIMARY KEY, btree (id)
    "posts_author_id" btree (author_id)
    "posts_posting_window_id" btree (posting_window_id)
Foreign-key constraints:
    "posts_author_id_fkey" FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT
    "posts_posting_window_id_fkey" FOREIGN KEY (posting_window_id) REFERENCES posting_windows(id) ON DELETE RESTRICT
Referenced by:
    TABLE "comments" CONSTRAINT "comments_post_id_fkey" FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE RESTRICT
    TABLE "realmojis" CONSTRAINT "realmojis_post_id_fkey" FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                         Table "public.realmojis"
    Column     |           Type           | Collation | Nullable | Default 
---------------+--------------------------+-----------+----------+---------
 id            | character varying(36)    |           | not null | 
 post_id       | character varying(36)    |           | not null | 
 reactor_id    | character varying(36)    |           | not null | 
 image_id      | character varying(36)    |           | not null | 
 reaction_type | text                     |           | not null | 
 created_at    | timestamp with time zone |           | not null | 
 replaced_at   | timestamp with time zone |           |          | 
 deleted_at    | timestamp with time zone |           |          | 
Indexes:
    "realmojis_pkey" PRIMARY KEY, btree (id)
    "realmojis_one_active" UNIQUE, btree (post_id, reactor_id) WHERE replaced_at IS NULL AND deleted_at IS NULL
    "realmojis_post_id" btree (post_id)
Foreign-key constraints:
    "realmojis_post_id_fkey" FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE RESTRICT
    "realmojis_reactor_id_fkey" FOREIGN KEY (reactor_id) REFERENCES users(id) ON DELETE RESTRICT

perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "UTF-8",
	LANG = "C.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("C.UTF-8").
                                  Table "public.users"
      Column      |           Type           | Collation | Nullable |      Default      
------------------+--------------------------+-----------+----------+-------------------
 id               | character varying(36)    |           | not null | 
 username         | character varying(64)    |           | not null | 
 secret_key_hash  | text                     |           | not null | 
 display_name     | text                     |           | not null | 
 bio              | text                     |           | not null | ''::text
 profile_image_id | character varying(36)    |           |          | 
 region           | text                     |           | not null | 
 account_type     | text                     |           | not null | 'regular'::text
 status           | text                     |           | not null | 'active'::text
 deleted_at       | timestamp with time zone |           |          | 
 created_at       | timestamp with time zone |           | not null | clock_timestamp()
 updated_at       | timestamp with time zone |           | not null | clock_timestamp()
Indexes:
    "users_pkey" PRIMARY KEY, btree (id)
Referenced by:
    TABLE "block_relations" CONSTRAINT "block_relations_blocked_id_fkey" FOREIGN KEY (blocked_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "block_relations" CONSTRAINT "block_relations_blocker_id_fkey" FOREIGN KEY (blocker_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "comments" CONSTRAINT "comments_author_id_fkey" FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "follows" CONSTRAINT "follows_followee_id_fkey" FOREIGN KEY (followee_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "follows" CONSTRAINT "follows_follower_id_fkey" FOREIGN KEY (follower_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "friend_requests" CONSTRAINT "friend_requests_recipient_id_fkey" FOREIGN KEY (recipient_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "friend_requests" CONSTRAINT "friend_requests_requester_id_fkey" FOREIGN KEY (requester_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "friendships" CONSTRAINT "friendships_user_high_id_fkey" FOREIGN KEY (user_high_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "friendships" CONSTRAINT "friendships_user_low_id_fkey" FOREIGN KEY (user_low_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "image_objects" CONSTRAINT "image_objects_owner_fk" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "login_sessions" CONSTRAINT "login_sessions_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "notifications" CONSTRAINT "notifications_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "posts" CONSTRAINT "posts_author_id_fkey" FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "realmojis" CONSTRAINT "realmojis_reactor_id_fkey" FOREIGN KEY (reactor_id) REFERENCES users(id) ON DELETE RESTRICT

```
