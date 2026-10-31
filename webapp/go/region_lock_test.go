package isureal

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("ISUREAL_TEST_DSN")
	if dsn == "" {
		t.Skip("ISUREAL_TEST_DSN is not set")
	}
	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func beginTestTx(t *testing.T, db *sqlx.DB) *sqlx.Tx {
	t.Helper()
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

func lockAsync(lock func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- lock() }()
	return done
}

func requireBlocked(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		t.Fatalf("lock was granted while it should wait (err=%v)", err)
	case <-time.After(300 * time.Millisecond):
	}
}

func requireGranted(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("lock was not granted")
	}
}

func TestRegionUsersLock(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	opener := beginTestTx(t, db)
	if err := lockRegionUsers(ctx, opener, "asia"); err != nil {
		t.Fatal(err)
	}

	otherRegion := beginTestTx(t, db)
	requireGranted(t, lockAsync(func() error { return lockRegionUsersShared(ctx, otherRegion, "europe") }))
	otherOpener := beginTestTx(t, db)
	requireGranted(t, lockAsync(func() error { return lockRegionUsers(ctx, otherOpener, "america") }))
	otherOpener.Rollback()

	register := beginTestTx(t, db)
	registerDone := lockAsync(func() error { return lockRegionUsersShared(ctx, register, "asia") })
	deletion := beginTestTx(t, db)
	deletionDone := lockAsync(func() error { return lockAllRegionUsersShared(ctx, deletion) })
	sameOpener := beginTestTx(t, db)
	sameOpenerDone := lockAsync(func() error { return lockRegionUsers(ctx, sameOpener, "asia") })
	requireBlocked(t, registerDone)
	requireBlocked(t, deletionDone)
	requireBlocked(t, sameOpenerDone)

	if err := opener.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-sameOpenerDone:
		sameOpener.Rollback()
	case <-registerDone:
		requireGranted(t, deletionDone)
		register.Rollback()
		deletion.Rollback()
		requireGranted(t, sameOpenerDone)
		return
	case <-time.After(3 * time.Second):
		t.Fatal("no waiter was granted after commit")
	}
	requireGranted(t, registerDone)
	requireGranted(t, deletionDone)
}
