package isureal

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
)

type currentPostingWindowResponse struct {
	ID             string    `json:"id"`
	Region         string    `json:"region"`
	LocalDate      string    `json:"localDate"`
	StartsAt       time.Time `json:"startsAt"`
	OnTimeDeadline time.Time `json:"onTimeDeadline"`
}

type getCurrentPostingWindowResponse struct {
	Window        *currentPostingWindowResponse `json:"window"`
	HasPosted     bool                          `json:"hasPosted"`
	CanPost       bool                          `json:"canPost"`
	DeletionCount int                           `json:"deletionCount"`
}

func getCurrentPostingWindowHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var state struct {
			sessionColumns
			ID             *string    `db:"id"`
			Region         *string    `db:"region"`
			LocalDate      *time.Time `db:"local_date"`
			StartsAt       *time.Time `db:"starts_at"`
			OnTimeDeadline *time.Time `db:"on_time_deadline"`
			Active         *int       `db:"active"`
			Deleted        *int       `db:"deleted"`
		}
		// "Posted today" means a post in any window sharing the local_date, regardless of region. The same rule is spelled out separately (not shared, to keep each query plan) in viewerContextSQL, activeWindowWithPostCountsQuery, getMyCurrentPostHandler, getCurrentPostingWindowHandler and deletePostHandler; change them together.
		err := h.DB.GetContext(c.Request().Context(), &state, `
   WITH `+sessionCTE+`
   SELECT `+sessionSelect+`, st.* FROM sess LEFT JOIN LATERAL (
     SELECT w.id, w.region, w.local_date, w.starts_at, w.on_time_deadline,
            count(p.id) FILTER (WHERE p.deleted_at IS NULL) AS active,
            count(p.id) FILTER (WHERE p.deleted_at IS NOT NULL) AS deleted
     FROM users u JOIN posting_windows w ON w.region = u.region AND w.ended_at IS NULL
     LEFT JOIN posting_windows posted_window ON posted_window.local_date = w.local_date
     LEFT JOIN posts p ON p.author_id = u.id AND p.posting_window_id = posted_window.id
     WHERE u.id = sess.user_id
     GROUP BY w.id
   ) st ON true
  `, foldedSessionTokenHash(c))
		if err := foldedSessionResult(c, h, err, state.sessionColumns); err != nil {
			return err
		}
		if state.ID == nil {
			return c.JSON(http.StatusOK, getCurrentPostingWindowResponse{})
		}
		active, deleted := *state.Active, *state.Deleted
		return c.JSON(http.StatusOK, getCurrentPostingWindowResponse{
			Window: &currentPostingWindowResponse{
				ID: *state.ID, Region: *state.Region, LocalDate: state.LocalDate.Format(time.DateOnly),
				StartsAt: *state.StartsAt, OnTimeDeadline: *state.OnTimeDeadline,
			},
			HasPosted:     active > 0,
			CanPost:       active == 0 && deleted < 2 && !time.Now().Before(*state.StartsAt),
			DeletionCount: deleted,
		})
	}
}
