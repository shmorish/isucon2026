package isureal

import (
	"database/sql"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
)

type createFriendRequestRequest struct {
	RecipientID string `json:"recipientId"`
}

type createFriendRequestResponse struct {
	RelationID string    `json:"relationId"`
	Status     string    `json:"status"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// createFriendRequestQuery runs after lockUserPair, so the checks see a snapshot taken after the lock: $1 actor,
// $2 recipient, $3/$4 the smaller/larger id. Blocked, already friends or the actor's own pending request -> no row.
// A pending request from the recipient is accepted (friendship $5); otherwise a new request ($6) is inserted.
const createFriendRequestQuery = `
  WITH st AS (
   SELECT
    EXISTS(SELECT 1 FROM block_relations WHERE (blocker_id=$1 AND blocked_id=$2) OR (blocker_id=$2 AND blocked_id=$1)) AS blocked,
    EXISTS(SELECT 1 FROM friendships WHERE user_low_id=$3 AND user_high_id=$4) AS friends,
    p.id AS pending_id, p.requester_id AS pending_requester_id
   FROM (SELECT 1) one
   LEFT JOIN LATERAL (
    SELECT id, requester_id FROM friend_requests WHERE status='pending' AND
     ((requester_id=$1 AND recipient_id=$2) OR (requester_id=$2 AND recipient_id=$1))
    LIMIT 1
   ) p ON true
  ), ok AS (
   SELECT pending_id FROM st WHERE NOT blocked AND NOT friends AND (pending_id IS NULL OR pending_requester_id=$2)
  ), r AS (
   UPDATE friend_requests SET status='accepted', updated_at=clock_timestamp() WHERE id=(SELECT pending_id FROM ok)
   RETURNING id, requester_id, recipient_id, status, created_at, updated_at
  ), f AS (
   INSERT INTO friendships (id, user_low_id, user_high_id, source_request_id) SELECT $5::varchar, $3, $4, id FROM r
  ), ins AS (
   INSERT INTO friend_requests (id, requester_id, recipient_id, status, created_at, updated_at)
   SELECT $6::varchar, $1, $2, 'pending', clock_timestamp(), clock_timestamp() FROM ok WHERE pending_id IS NULL
   RETURNING id, requester_id, recipient_id, status, created_at, updated_at
  )
  SELECT * FROM r UNION ALL SELECT * FROM ins
`

func createFriendRequestHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input createFriendRequestRequest
		if err := c.Bind(&input); err != nil {
			return err
		}
		ctx := c.Request().Context()
		actor := GetSession(c).UserID
		tx, err := beginRelationshipTx(ctx, h.DB, actor, input.RecipientID)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		// 状態の確認と書き込みを1往復で行う。0行なら衝突
		var request FriendRequest
		err = tx.GetContext(ctx, &request, createFriendRequestQuery, actor, input.RecipientID,
			min(actor, input.RecipientID), max(actor, input.RecipientID), uuid.NewString(), uuid.NewString())
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusConflict, "relationship_conflict")
		}
		if err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, createFriendRequestResponse{RelationID: request.ID, Status: request.Status, UpdatedAt: request.UpdatedAt})
	}
}

type transitionFriendRequestRequest struct {
	RequestID string `param:"requestId" json:"-"`
	Action    string `param:"action" json:"-"`
}

type transitionFriendRequestResponse createFriendRequestResponse

// transitionFriendRequestQuery runs after lockUserPair: $1 request, $2/$3 requester/recipient, $4 new status,
// $5 friendship id, $6/$7 the smaller/larger user id. The update only happens when the request is pending and,
// for accepted, the pair is not blocked; current_status and blocked tell the handler which error to return.
const transitionFriendRequestQuery = `
  WITH cur AS (
   SELECT status, EXISTS(SELECT 1 FROM block_relations WHERE (blocker_id=$2 AND blocked_id=$3) OR (blocker_id=$3 AND blocked_id=$2)) AS blocked
   FROM friend_requests WHERE id=$1
  ), r AS (
   UPDATE friend_requests SET status=$4::text, updated_at=clock_timestamp()
   WHERE id=$1 AND EXISTS(SELECT 1 FROM cur WHERE status='pending' AND ($4::text<>'accepted' OR NOT blocked))
   RETURNING status, updated_at
  ), f AS (
   INSERT INTO friendships (id, user_low_id, user_high_id, source_request_id)
   SELECT $5::varchar, $6::varchar, $7::varchar, $1 FROM r WHERE status='accepted'
  )
  SELECT cur.status AS current_status, cur.blocked, r.status, r.updated_at FROM cur LEFT JOIN r ON true
`

func transitionFriendRequestHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input transitionFriendRequestRequest
		if err := c.Bind(&input); err != nil {
			return err
		}
		var status string
		switch input.Action {
		case "accept":
			status = "accepted"
		case "reject":
			status = "rejected"
		case "cancel":
			status = "cancelled"
		default:
			return NewAPIError(http.StatusForbidden, "relationship_action_forbidden")
		}
		ctx := c.Request().Context()
		actor := GetSession(c).UserID
		// BEGIN は申請の取得と同時に送る。BEGIN だけではスナップショットもロックも取らない
		var tx *sqlx.Tx
		var txErr error
		var wg sync.WaitGroup
		wg.Go(func() { tx, txErr = h.DB.BeginTxx(ctx, nil) })
		request, err := getFriendRequest(ctx, h.DB, input.RequestID)
		wg.Wait()
		if txErr == nil {
			defer tx.Rollback()
		}
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusNotFound, "relationship_not_found")
		}
		if err != nil {
			return err
		}
		if (status == "cancelled" && actor != request.RequesterID) || (status != "cancelled" && actor != request.RecipientID) {
			return NewAPIError(http.StatusForbidden, "relationship_action_forbidden")
		}
		if err := validateUserPair(request.RequesterID, request.RecipientID); err != nil {
			return err
		}
		if txErr != nil {
			return txErr
		}
		if err := lockUserPair(ctx, tx, request.RequesterID, request.RecipientID); err != nil {
			return err
		}
		// ロックの後の読み直し・ブロックの確認・書き込みを1往復で行う
		var result struct {
			Current   string     `db:"current_status"`
			Blocked   bool       `db:"blocked"`
			Status    *string    `db:"status"`
			UpdatedAt *time.Time `db:"updated_at"`
		}
		if err := tx.GetContext(ctx, &result, transitionFriendRequestQuery, request.ID, request.RequesterID, request.RecipientID,
			status, uuid.NewString(), min(request.RequesterID, request.RecipientID), max(request.RequesterID, request.RecipientID)); err != nil {
			return err
		}
		if result.Current != "pending" {
			return NewAPIError(http.StatusForbidden, "relationship_action_forbidden")
		}
		if status == "accepted" && result.Blocked {
			return NewAPIError(http.StatusConflict, "relationship_conflict")
		}
		if result.Status == nil {
			return errors.New("friend request was not updated")
		}
		request.Status, request.UpdatedAt = *result.Status, *result.UpdatedAt
		if err := tx.Commit(); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, transitionFriendRequestResponse{RelationID: request.ID, Status: request.Status, UpdatedAt: request.UpdatedAt})
	}
}
