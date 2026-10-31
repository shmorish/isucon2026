package isureal

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
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
		// ブロック・友達・申請中の確認を1往復で行う
		var state struct {
			Blocked            bool    `db:"blocked"`
			Friends            bool    `db:"friends"`
			PendingID          *string `db:"pending_id"`
			PendingRequesterID *string `db:"pending_requester_id"`
		}
		if err := tx.GetContext(ctx, &state, `
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
  `, actor, input.RecipientID, min(actor, input.RecipientID), max(actor, input.RecipientID)); err != nil {
			return err
		}
		if state.Blocked || state.Friends || (state.PendingID != nil && *state.PendingRequesterID == actor) {
			return NewAPIError(http.StatusConflict, "relationship_conflict")
		}
		var request FriendRequest
		if state.PendingID != nil {
			// 相手からの申請があれば承認して友達にする
			request, err = acceptFriendRequest(ctx, tx, *state.PendingID, actor, input.RecipientID)
		} else {
			err = tx.GetContext(ctx, &request, `INSERT INTO friend_requests (id, requester_id, recipient_id, status, created_at, updated_at) VALUES ($1,$2,$3,'pending',clock_timestamp(),clock_timestamp()) RETURNING id, requester_id, recipient_id, status, created_at, updated_at`, uuid.NewString(), actor, input.RecipientID)
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
		request, err := getFriendRequest(ctx, h.DB, input.RequestID)
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusNotFound, "relationship_not_found")
		}
		if err != nil {
			return err
		}
		if (status == "cancelled" && actor != request.RequesterID) || (status != "cancelled" && actor != request.RecipientID) {
			return NewAPIError(http.StatusForbidden, "relationship_action_forbidden")
		}
		tx, err := beginRelationshipTx(ctx, h.DB, request.RequesterID, request.RecipientID)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		// ロックの後の読み直しとブロックの確認を1往復で行う
		var current struct {
			Status  string `db:"status"`
			Blocked bool   `db:"blocked"`
		}
		if err := tx.GetContext(ctx, &current, `SELECT status, EXISTS(SELECT 1 FROM block_relations WHERE (blocker_id=$2 AND blocked_id=$3) OR (blocker_id=$3 AND blocked_id=$2)) AS blocked FROM friend_requests WHERE id=$1`, request.ID, request.RequesterID, request.RecipientID); err != nil {
			return err
		}
		if current.Status != "pending" {
			return NewAPIError(http.StatusForbidden, "relationship_action_forbidden")
		}
		if status == "accepted" {
			if current.Blocked {
				return NewAPIError(http.StatusConflict, "relationship_conflict")
			}
			request, err = acceptFriendRequest(ctx, tx, request.ID, request.RequesterID, request.RecipientID)
		} else {
			err = tx.GetContext(ctx, &request, `UPDATE friend_requests SET status=$2, updated_at=clock_timestamp() WHERE id=$1 RETURNING id, requester_id, recipient_id, status, created_at, updated_at`, request.ID, status)
		}
		if err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, transitionFriendRequestResponse{RelationID: request.ID, Status: request.Status, UpdatedAt: request.UpdatedAt})
	}
}
