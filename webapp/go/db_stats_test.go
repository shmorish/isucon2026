package isureal

import (
	"encoding/json"
	"expvar"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestPublishDBStatsExposesPoolCounters(t *testing.T) {
	db, err := sqlx.Open("pgx", "postgres://localhost/unused")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(7)
	publishDBStats(db)

	v := expvar.Get("db")
	if v == nil {
		t.Fatal("db expvar is not published")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(v.String()), &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"MaxOpenConnections", "InUse", "Idle", "WaitCount", "WaitDuration", "WaitDurationMs"} {
		if _, ok := got[key]; !ok {
			t.Errorf("missing %s in %s", key, v.String())
		}
	}
	if got["MaxOpenConnections"] != float64(7) {
		t.Errorf("MaxOpenConnections = %v, want 7", got["MaxOpenConnections"])
	}
}
