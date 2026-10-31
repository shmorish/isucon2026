package isureal

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type PostingWindow struct {
	ID             string     `db:"id"`
	Region         string     `db:"region"`
	LocalDate      time.Time  `db:"local_date"`
	StartsAt       time.Time  `db:"starts_at"`
	OnTimeDeadline time.Time  `db:"on_time_deadline"`
	EndedAt        *time.Time `db:"ended_at"`
}

// Advisory lock classes: 1 = a region's set of active users, 2 = a username.
// Every statement taking one of them is built from the expressions below, so
// the window opener and the registration/deletion/region-change paths it must
// exclude cannot drift apart.
const (
	regionUsersLockClass = "1"
	usernameLockClass    = "2"
)

func regionUsersLockExpr(key string) string {
	return "pg_advisory_xact_lock(" + regionUsersLockClass + ", hashtext(" + key + "))"
}

func regionUsersSharedLockExpr(key string) string {
	return "pg_advisory_xact_lock_shared(" + regionUsersLockClass + ", hashtext(" + key + "))"
}

func usernameLockExpr(key string) string {
	return "pg_advisory_xact_lock(" + usernameLockClass + ", hashtext(" + key + "))"
}

var (
	lockRegionUsersSQL       = "SELECT " + regionUsersLockExpr("$1")
	lockRegionUsersSharedSQL = "SELECT " + regionUsersSharedLockExpr("$1")
	// Alphabetical so every transaction takes the three region locks in the same order (no deadlock); must list exactly the regions validRegion accepts.
	lockAllRegionUsersSharedSQL = "SELECT " + regionUsersSharedLockExpr("r") + " FROM unnest(ARRAY['america','asia','europe']) AS r"
	// $1 = region, $2 = username. The region's shared lock comes before the username lock; OFFSET 0 keeps the subquery from being flattened so that order is fixed.
	lockRegistrationSQL = "SELECT " + usernameLockExpr("$2") + " FROM (SELECT " + regionUsersSharedLockExpr("$1") + " OFFSET 0) AS region_lock"
)

// lockRegionUsers is taken by opening a posting window; the shared lock by
// anything that changes which users are active in a region (registration via
// lockRegistrationSQL, deletion and region change via lockAllRegionUsersShared),
// so a window's notifications cover exactly the region's active users when it
// opens. lockRegionUsersShared itself is only used by region_lock_test.go.
func lockRegionUsers(ctx context.Context, tx *sqlx.Tx, region string) error {
	_, err := tx.ExecContext(ctx, lockRegionUsersSQL, region)
	return err
}

func lockRegionUsersShared(ctx context.Context, tx *sqlx.Tx, region string) error {
	_, err := tx.ExecContext(ctx, lockRegionUsersSharedSQL, region)
	return err
}

func lockAllRegionUsersShared(ctx context.Context, tx *sqlx.Tx) error {
	_, err := tx.ExecContext(ctx, lockAllRegionUsersSharedSQL)
	return err
}

func getPostingWindow(ctx context.Context, db sqlx.QueryerContext, windowID string) (PostingWindow, error) {
	var window PostingWindow
	err := sqlx.GetContext(ctx, db, &window, `SELECT id, region, local_date, starts_at, on_time_deadline, ended_at FROM posting_windows WHERE id = $1`, windowID)
	return window, err
}
