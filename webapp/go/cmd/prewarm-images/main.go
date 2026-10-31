package main

import (
	"context"

	"example.com/isureal-company/isureal"
)

// 初期データの画像をディスクに書いておく（#65）。s1・s2・s3 でベンチの外に一度だけ（s2・s3 は europe・america の画像を自分のディスクから返す）:
//
//	cd ~/webapp/go && go build -o /tmp/prewarm-images ./cmd/prewarm-images && (set -a; . ~/env.sh; /tmp/prewarm-images)
func main() {
	if err := isureal.PrewarmImageDiskCache(context.Background(), isureal.ConfigFromEnv(), 3); err != nil {
		panic(err)
	}
}
