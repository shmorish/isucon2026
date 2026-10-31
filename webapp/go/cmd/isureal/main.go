package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"

	"example.com/isureal-company/isureal"
)

// 2026-09-23

func main() {
	// 終盤は環境変数 NO_PPROF=1 で止める
	if os.Getenv("NO_PPROF") == "" {
		go http.ListenAndServe("localhost:6060", nil)
	}
	config := isureal.ConfigFromEnv()
	server := isureal.NewServer(config)
	if err := server.Start(":8080"); err != nil {
		panic("failed to start server: " + err.Error())
	}
}
