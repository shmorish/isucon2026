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

// lockRegionUsers is taken by opening a posting window; lockRegionUsersShared
// by anything that changes which users are active in a region, so a window's
// notifications cover exactly the region's active users when it opens.
func lockRegionUsers(ctx context.Context, tx *sqlx.Tx, region string) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(1, hashtext($1))`, region)
	return err
}

func lockRegionUsersShared(ctx context.Context, tx *sqlx.Tx, region string) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock_shared(1, hashtext($1))`, region)
	return err
}

func lockAllRegionUsersShared(ctx context.Context, tx *sqlx.Tx) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock_shared(1, hashtext(r)) FROM unnest(ARRAY['america','asia','europe']) AS r`)
	return err
}

func getPostingWindow(ctx context.Context, db sqlx.QueryerContext, windowID string) (PostingWindow, error) {
	var window PostingWindow
	err := sqlx.GetContext(ctx, db, &window, `SELECT id, region, local_date, starts_at, on_time_deadline, ended_at FROM posting_windows WHERE id = $1`, windowID)
	return window, err
}
