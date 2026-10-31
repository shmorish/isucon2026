# PostgreSQL の log_min_duration_statement のログをクエリ別に集計する (task slp から呼ぶ)。閾値未満のクエリは数えないので、件数の合計は task pgstat を見る
# slp pg は pgx のプリペアドステートメント (execute stmtcache_...) を読めないので自前で集計する

import re, sys, collections
head = re.compile(r'^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d\.\d+ UTC \[\d+\] ')
dur = re.compile(r'LOG:  duration: ([\d.]+) ms  (?:execute [^:]*|statement): (.*)', re.S)
agg = collections.defaultdict(lambda: [0, 0.0, 0.0])
def flush(buf):
    if not buf: return
    m = dur.search(buf)
    if not m: return
    q = re.sub(r'\s+', ' ', m.group(2)).strip()
    q = re.sub(r"'[^']*'", "'S'", q)
    q = re.sub(r'\$\d+(, \$\d+)+', '$N..', q)
    a = agg[q[:200]]; t = float(m.group(1)) / 1000
    a[0] += 1; a[1] += t; a[2] = max(a[2], t)
buf = ''
for line in open(sys.argv[1], errors='replace'):
    if head.match(line):
        flush(buf); buf = line
    elif line.startswith('\t') or not line.startswith('20'):
        buf += line
flush(buf)
print('count\tsum(s)\tavg(ms)\tmax(s)\tquery')
for q, (c, s, mx) in sorted(agg.items(), key=lambda kv: -kv[1][1])[:25]:
    print(f'{c}\t{s:.2f}\t{s/c*1000:.2f}\t{mx:.2f}\t{q}')
