package isuutil

import (
	"net/http"
	_ "net/http/pprof"
	"os"
)

// StartPprof を main の先頭で呼ぶ。終盤は環境変数 NO_PPROF=1 で止める。
// 取得: task pprof (ベンチ開始直前に実行)
func StartPprof() {
	if os.Getenv("NO_PPROF") != "" {
		return
	}
	go http.ListenAndServe("localhost:6060", nil)
}
