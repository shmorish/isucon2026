# ip-192-168-3-10 2026-10-31 01:16:51

## スペック

```
2
CPU(s):                                  2
Model name:                              Intel(R) Xeon(R) Platinum 8488C
               total        used        free      shared  buff/cache   available
Mem:           3.7Gi       461Mi       3.2Gi       2.9Mi       281Mi       3.3Gi
Swap:             0B          0B          0B
Filesystem      Size  Used Avail Use% Mounted on
/dev/root        58G   24G   34G  42% /
7.0.0-1014-aws
PRETTY_NAME="Ubuntu 24.04.5 LTS"
```

## 動いているサービス

```
  isureal-go.service                             loaded active running isureal-go
  isureal-guard.service                          loaded active running isureal-guard
  ModemManager.service                           loaded active running Modem Manager
  nginx.service                                  loaded active running A high performance web server and a reverse proxy server
```

## isu 関連の unit 一覧

```
envcheck-baseline.service                      enabled         enabled
envcheck.service                               enabled         enabled
isucon-register.service                        enabled         enabled
isureal-go.service                             enabled         enabled
isureal-guard.service                          enabled         enabled
isureal-nodejs.service                         disabled        enabled
isureal-perl.service                           disabled        enabled
isureal-php.service                            disabled        enabled
isureal-python.service                         disabled        enabled
isureal-ruby.service                           disabled        enabled
isureal-rust.service                           disabled        enabled
nginx.service                                  enabled         enabled
postgresql.service                             disabled        enabled
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
LISTEN 0      4096   127.0.0.53%lo:53        0.0.0.0:*    users:(("systemd-resolve",pid=373,fd=15))                                      
LISTEN 0      4096      127.0.0.54:53        0.0.0.0:*    users:(("systemd-resolve",pid=373,fd=17))                                      
LISTEN 0      4096         0.0.0.0:22        0.0.0.0:*    users:(("sshd",pid=704,fd=3),("systemd",pid=1,fd=112))                         
LISTEN 0      511          0.0.0.0:80        0.0.0.0:*    users:(("nginx",pid=1227,fd=5),("nginx",pid=1226,fd=5),("nginx",pid=1225,fd=5))
LISTEN 0      4096               *:8080            *:*    users:(("isureal",pid=1498,fd=6))                                              
LISTEN 0      4096            [::]:22           [::]:*    users:(("sshd",pid=704,fd=4),("systemd",pid=1,fd=114))                         
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
drwxr-xr-x 20 isucon isucon  4096 Oct 31 01:10 .
drwxr-xr-x  5 root   root    4096 Oct 31 01:10 ..
drwxrwxr-x  3 isucon isucon  4096 Oct 30 11:52 .ansible
-rw-r--r--  1 isucon isucon   220 Mar 31  2024 .bash_logout
-rw-r--r--  1 isucon isucon  3771 Mar 31  2024 .bashrc
drwxrwxr-x  3 isucon isucon  4096 Oct 30 11:59 .bundle
drwxrwxr-x  5 isucon isucon  4096 Oct 31 01:16 .cache
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
-rw-r--r--  1 isucon isucon   149 Oct 31 01:10 env.sh
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
