package isureal

import (
	"database/sql"
	"expvar"
	"sync"
	"sync/atomic"

	"github.com/jmoiron/sqlx"
)

var (
	statsDB            atomic.Pointer[sqlx.DB]
	publishDBStatsOnce sync.Once
)

// echo（:8080）ではなく DefaultServeMux の /debug/vars に出す。DefaultServeMux は localhost:6060 の pprof だけが使う
func publishDBStats(db *sqlx.DB) {
	statsDB.Store(db)
	publishDBStatsOnce.Do(func() {
		expvar.Publish("db", expvar.Func(func() any {
			s := statsDB.Load().Stats()
			return struct {
				sql.DBStats
				WaitDurationMs int64
			}{s, s.WaitDuration.Milliseconds()}
		}))
	})
}
