package isureal

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type FriendRequest struct {
	ID          string    `db:"id"`
	RequesterID string    `db:"requester_id"`
	RecipientID string    `db:"recipient_id"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func getFriendRequest(ctx context.Context, db sqlx.QueryerContext, id string) (FriendRequest, error) {
	var request FriendRequest
	err := sqlx.GetContext(ctx, db, &request, `SELECT id, requester_id, recipient_id, status, created_at, updated_at FROM friend_requests WHERE id=$1`, id)
	return request, err
}

const (
	lockActiveUserExclusive = `SELECT id FROM users WHERE id=$1 AND status='active' FOR NO KEY UPDATE`
	lockActiveUserShared    = `SELECT id FROM users WHERE id=$1 AND status='active' FOR SHARE`
)

func beginRelationshipTx(ctx context.Context, db *sqlx.DB, actor, target string) (*sqlx.Tx, error) {
	return beginUserPairTx(ctx, db, actor, target, lockActiveUserExclusive)
}

// beginReactionTx locks the target FOR SHARE, which still conflicts with the
// target's deletion and relationship changes but not with other reactions.
func beginReactionTx(ctx context.Context, db *sqlx.DB, actor, target string) (*sqlx.Tx, error) {
	return beginUserPairTx(ctx, db, actor, target, lockActiveUserShared)
}

func beginUserPairTx(ctx context.Context, db *sqlx.DB, actor, target, targetLock string) (*sqlx.Tx, error) {
	if actor == target || target == "" {
		return nil, NewAPIError(http.StatusBadRequest, "invalid_relationship")
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if targetLock == lockActiveUserExclusive {
		// 1往復で2行ともロックする。LockRows は Sort の上で動くので、ID 順にロックされる
		var lockedIDs []string
		if err := tx.SelectContext(ctx, &lockedIDs, `SELECT id FROM users WHERE id IN ($1, $2) AND status='active' ORDER BY id FOR NO KEY UPDATE`, actor, target); err != nil {
			tx.Rollback()
			return nil, err
		}
		if len(lockedIDs) != 2 {
			tx.Rollback()
			return nil, NewAPIError(http.StatusNotFound, "relationship_not_found")
		}
		return tx, nil
	}
	for _, id := range []string{min(actor, target), max(actor, target)} {
		lock := lockActiveUserExclusive
		if id == target {
			lock = targetLock
		}
		var lockedID string
		if err := tx.GetContext(ctx, &lockedID, lock, id); err != nil {
			tx.Rollback()
			if errors.Is(err, sql.ErrNoRows) {
				return nil, NewAPIError(http.StatusNotFound, "relationship_not_found")
			}
			return nil, err
		}
	}
	return tx, nil
}

func isBlocked(ctx context.Context, db sqlx.QueryerContext, actor, target string) (bool, error) {
	var blocked bool
	err := sqlx.GetContext(ctx, db, &blocked, `SELECT EXISTS(SELECT 1 FROM block_relations WHERE (blocker_id=$1 AND blocked_id=$2) OR (blocker_id=$2 AND blocked_id=$1))`, actor, target)
	return blocked, err
}

// acceptFriendRequest は申請を承認済みにして友達関係を作り、更新後の申請を返す（1往復）
func acceptFriendRequest(ctx context.Context, tx *sqlx.Tx, requestID, userA, userB string) (FriendRequest, error) {
	var request FriendRequest
	err := tx.GetContext(ctx, &request, `
  WITH r AS (
   UPDATE friend_requests SET status='accepted', updated_at=clock_timestamp() WHERE id=$1
   RETURNING id, requester_id, recipient_id, status, created_at, updated_at
  ), f AS (
   INSERT INTO friendships (id, user_low_id, user_high_id, source_request_id) VALUES ($2, $3, $4, $1)
  )
  SELECT * FROM r
 `, requestID, uuid.NewString(), min(userA, userB), max(userA, userB))
	return request, err
}

func requireRelationshipChange(result sql.Result, err error, status int, code string) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return NewAPIError(status, code)
	}
	return nil
}
