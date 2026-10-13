// Package isuutil は当日アプリにコピペする部品集。ファイル単位で持っていく。
package isuutil

import (
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
)

// OpenDB は ISUCON 定番の設定で MySQL に接続する。
// 参照実装の接続処理をこれに差し替える (sqlx なら sqlx.NewDb(db, "mysql"))。
func OpenDB(host, port, user, pass, dbname string) (*sql.DB, error) {
	cfg := mysql.NewConfig()
	cfg.Net = "tcp"
	cfg.Addr = host + ":" + port
	cfg.User = user
	cfg.Passwd = pass
	cfg.DBName = dbname
	cfg.ParseTime = true
	cfg.Loc = time.Local
	cfg.InterpolateParams = true // prepare の往復を消す。最初にやる
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(64) // 10→40 で 97k→112k (ISUCON11 fujiwara組)。ベンチで調整
	db.SetMaxIdleConns(64) // Open と揃えないと接続を張り直し続ける
	db.SetConnMaxLifetime(2 * time.Minute)
	return db, nil
}

// バルク INSERT (sqlx):
//   db.NamedExec("INSERT INTO t (a, b) VALUES (:a, :b)", rows) // rows は []struct
// IN 句 (sqlx):
//   q, args, _ := sqlx.In("SELECT * FROM t WHERE id IN (?)", ids) // len(ids)==0 なら呼ばない
//   db.Select(&out, db.Rebind(q), args...)
