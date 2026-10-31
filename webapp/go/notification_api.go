package isureal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
)

type getNotificationsResponse struct {
	ID              string    `json:"id" db:"id"`
	PostingWindowID string    `json:"postingWindowId" db:"posting_window_id"`
	Region          string    `json:"region" db:"region"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
}

func getNotificationsHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ctx := c.Request().Context()
		session := GetSession(c)
		// 投稿期間の starts_at は s4 の clock_timestamp()。s1 と s4 は chrony で数µs以内に揃っているので、ヘッダを返す前に
		// DB を1往復しない（最終日にプールが埋まるとヘッダが返せず、ルーターが 503 を返していた）
		connectedAt := time.Now()
		response := c.Response()
		response.Header().Set("Content-Type", "text/event-stream")
		response.WriteHeader(http.StatusOK)
		controller := http.NewResponseController(response)
		if err := controller.Flush(); err != nil {
			return nil
		}
		sub := h.Notifications.subscribe(session.UserID, session.Region, connectedAt)
		defer h.Notifications.unsubscribe(sub)
		var notification *getNotificationsResponse
		select {
		case <-ctx.Done():
			return nil
		case notification = <-sub.result:
		}
		if notification == nil {
			return nil
		}
		data, err := json.Marshal(notification)
		if err != nil {
			c.Logger().Error("failed to encode notification", "error", err)
			return nil
		}
		if _, err := fmt.Fprintf(response, "data: %s\n\n", data); err != nil {
			return nil
		}
		_ = controller.Flush()
		return nil
	}
}
