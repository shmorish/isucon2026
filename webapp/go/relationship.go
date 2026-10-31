package isureal

import (
	"context"
	"database/sql"
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

// beginRelationshipTx は actor を FOR NO KEY UPDATE、target を FOR SHARE でロックする。
// 関係の変更は2人のペアの中で閉じているので、同じペアへの操作は actor 側の排他と相手側の共有がぶつかって直列になる。
// 相手の退会や認証済みへの変更（本人の行の FOR NO KEY UPDATE・UPDATE）とも競合する。
// 人気ユーザーへのフォローや申請は、相手側が共有ロックなので同時に進む。
func beginRelationshipTx(ctx context.Context, db *sqlx.DB, actor, target string) (*sqlx.Tx, error) {
	if err := validateUserPair(actor, target); err != nil {
		return nil, err
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if err := lockUserPair(ctx, tx, actor, target); err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func validateUserPair(actor, target string) error {
	if actor == target || target == "" {
		return NewAPIError(http.StatusBadRequest, "invalid_relationship")
	}
	return nil
}

// lockUserPair takes beginRelationshipTx's locks on a transaction the caller began.
func lockUserPair(ctx context.Context, tx *sqlx.Tx, actor, target string) error {
	// 1往復で2行をロックする。対象リストは左から評価されるので、ID の小さい方から先にロックされる
	lock := func(id string) string {
		if id == actor {
			return "FOR NO KEY UPDATE"
		}
		return "FOR SHARE"
	}
	low, high := min(actor, target), max(actor, target)
	var locked struct {
		Low  *string `db:"low"`
		High *string `db:"high"`
	}
	if err := tx.GetContext(ctx, &locked, `SELECT
   (SELECT id FROM users WHERE id=$1 AND status='active' `+lock(low)+`) AS low,
   (SELECT id FROM users WHERE id=$2 AND status='active' `+lock(high)+`) AS high`, low, high); err != nil {
		return err
	}
	if locked.Low == nil || locked.High == nil {
		return NewAPIError(http.StatusNotFound, "relationship_not_found")
	}
	return nil
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
