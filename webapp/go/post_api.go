package isureal

import (
	"bytes"
	"database/sql"
	"errors"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type createPostRequest struct {
	Image        *multipart.FileHeader `form:"image"`
	Caption      string                `form:"caption"`
	Visibility   string                `form:"visibility"`
	LocationKind string                `form:"location_kind"`
	Latitude     *float64              `form:"latitude"`
	Longitude    *float64              `form:"longitude"`
	Locality     *string               `form:"locality"`
}

type postAuthorResponse struct {
	ID              string  `json:"id"`
	Username        string  `json:"username"`
	DisplayName     string  `json:"displayName"`
	ProfileImageURL *string `json:"profileImageUrl"`
	Bio             string  `json:"bio"`
	AccountType     string  `json:"accountType"`
}

type createPostResponse struct {
	ID              string             `json:"id"`
	Author          postAuthorResponse `json:"author"`
	PostingWindowID string             `json:"postingWindowId"`
	ImageURL        string             `json:"imageUrl"`
	Caption         string             `json:"caption"`
	Visibility      string             `json:"visibility"`
	Location        PostLocation       `json:"location"`
	PostedAt        time.Time          `json:"postedAt"`
	OnTime          bool               `json:"onTime"`
	LateSeconds     int64              `json:"lateSeconds"`
}

func createPostHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input createPostRequest
		if err := c.Bind(&input); err != nil {
			return err
		}
		if input.Image == nil || !validUserText(input.Caption, 0, 1000) {
			return NewAPIError(http.StatusBadRequest, "invalid_post")
		}
		switch input.Visibility {
		case "friends", "friends_of_friends", "public":
		default:
			return NewAPIError(http.StatusBadRequest, "invalid_post")
		}
		location := PostLocation{
			Kind: input.LocationKind, Latitude: input.Latitude,
			Longitude: input.Longitude, Locality: input.Locality,
		}
		if !location.valid() {
			return NewAPIError(http.StatusBadRequest, "invalid_post")
		}
		imageFile, err := input.Image.Open()
		if err != nil {
			return err
		}
		defer imageFile.Close()
		imageData, err := io.ReadAll(imageFile)
		if err != nil {
			return err
		}
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(imageData))
		if err != nil {
			return NewAPIError(http.StatusBadRequest, "invalid_post")
		}
		if cfg.Width > 4096 || cfg.Height > 4096 {
			return NewAPIError(http.StatusBadRequest, "invalid_post")
		}

		ctx := c.Request().Context()
		userID := GetSession(c).UserID
		tx, err := h.DB.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		user, err := lockUser(ctx, tx, userID)
		if err != nil {
			return err
		}
		if err := user.requireActive(); err != nil {
			return err
		}
		var window struct {
			ID        string    `db:"id"`
			LocalDate time.Time `db:"local_date"`
			StartsAt  time.Time `db:"starts_at"`
			Active    int       `db:"active"`
			Deleted   int       `db:"deleted"`
		}
		err = tx.GetContext(ctx, &window, activeWindowWithPostCountsQuery, userID, user.Region)
		// 重複に見えるが消さない。FOR SHARE で待つ間に投稿期間が終わると、READ COMMITTED の再確認で
		// 行が外れて0行になる。次の文は新しいスナップショットで始まり、新しく開いた期間を見る。
		if errors.Is(err, sql.ErrNoRows) {
			err = tx.GetContext(ctx, &window, activeWindowWithPostCountsQuery, userID, user.Region)
		}
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return NewAPIError(http.StatusConflict, "post_slot_unavailable")
			}
			return err
		}
		postedAt := time.Now().UTC()
		if window.Active != 0 || window.Deleted >= 2 || postedAt.Before(window.StartsAt) {
			return NewAPIError(http.StatusConflict, "post_slot_unavailable")
		}
		imageID := uuid.NewString()
		var post Post
		if err := tx.GetContext(ctx, &post, `
   WITH image AS (
     INSERT INTO image_objects (id, owner_user_id, data, media_type, created_at) VALUES ($1, $2, $3, 'image/jpeg', $4)
   ), p AS (
     INSERT INTO posts (id, author_id, posting_window_id, image_id, caption, visibility,
                        location_kind, latitude, longitude, locality, posted_at)
     VALUES ($5, $2, $6, $1, $7, $8, $9, $10, $11, $12, $4)
     RETURNING *
   )
   SELECT `+postColumns+` FROM p JOIN posting_windows w ON w.id = p.posting_window_id
  `, imageID, userID, imageData, postedAt, uuid.NewString(), window.ID, input.Caption, input.Visibility,
			location.Kind, location.Latitude, location.Longitude, location.Locality); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		cacheUploadedImage(imageID, imageData)
		imageCreatedAtCache.put(imageID, post.PostedAt)
		return c.JSON(http.StatusCreated, newPostResponse(post, user))
	}
}

func newPostResponse(post Post, user User) createPostResponse {
	response := createPostResponse{
		ID: post.ID,
		Author: postAuthorResponse{
			ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
			ProfileImageURL: newGetMeResponse(user).ProfileImageURL,
			Bio:             user.Bio, AccountType: user.AccountType,
		},
		PostingWindowID: post.PostingWindowID, ImageURL: signedImageURL(user, post.ImageID),
		Caption: post.Caption, Visibility: post.Visibility, Location: post.PostLocation,
		PostedAt: post.PostedAt, OnTime: !post.PostedAt.After(post.OnTimeDeadline),
	}
	if !response.OnTime {
		response.LateSeconds = int64(post.PostedAt.Sub(post.OnTimeDeadline) / time.Second)
	}
	return response
}

type getMyCurrentPostResponse createPostResponse

func getMyCurrentPostHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ctx := c.Request().Context()
		var row struct {
			sessionColumns
			PostID *string `db:"post_id"`
		}
		// "Posted today" means a post in any window sharing the local_date, regardless of region. The same rule is spelled out separately (not shared, to keep each query plan) in viewerContextSQL, activeWindowWithPostCountsQuery, getMyCurrentPostHandler, getCurrentPostingWindowHandler and deletePostHandler; change them together.
		err := h.DB.GetContext(ctx, &row, `
   WITH `+sessionCTE+`
   SELECT `+sessionSelect+`, mine.id AS post_id FROM sess LEFT JOIN LATERAL (
     SELECT p.id FROM posts p
     JOIN posting_windows w ON w.id = p.posting_window_id
     JOIN users u ON u.id = p.author_id
     JOIN posting_windows current ON current.region = u.region AND current.ended_at IS NULL
                                AND current.local_date = w.local_date
     WHERE p.author_id = sess.user_id AND p.deleted_at IS NULL
   ) mine ON true
  `, foldedSessionTokenHash(c))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := acceptFoldedSession(c, h, err == nil, row.sessionColumns); err != nil {
			return err
		}
		if row.PostID == nil {
			return NewAPIError(http.StatusNotFound, "post_not_found")
		}
		userID := row.SessUserID
		post, err := getPost(ctx, h.DB, *row.PostID)
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusNotFound, "post_not_found")
		}
		if err != nil {
			return err
		}
		user, err := getUser(ctx, h.DB, userID)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, getMyCurrentPostResponse(newPostResponse(post, user)))
	}
}

type getPostRequest struct {
	PostID string `param:"postId"`
}

type getPostResponse createPostResponse

func getPostHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input getPostRequest
		if err := c.Bind(&input); err != nil {
			return err
		}
		ctx := c.Request().Context()
		post, viewer, found, err := loadPostAndViewer(ctx, h.DB, GetSession(c).UserID, input.PostID)
		if err != nil {
			return err
		}
		if !found || !viewer.canView(post.Post) {
			return NewAPIError(http.StatusNotFound, "post_not_found")
		}
		return c.JSON(http.StatusOK, getPostResponse(newPostResponse(post.Post, post.Author)))
	}
}

type updatePostRequest struct {
	PostID     string  `param:"postId" json:"-"`
	Caption    *string `json:"caption"`
	Visibility *string `json:"visibility"`
}

type updatePostResponse createPostResponse

func updatePostHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input updatePostRequest
		if err := c.Bind(&input); err != nil {
			return err
		}
		if input.Caption == nil && input.Visibility == nil {
			return NewAPIError(http.StatusBadRequest, "invalid_post")
		}
		if input.Caption != nil && !validUserText(*input.Caption, 0, 1000) {
			return NewAPIError(http.StatusBadRequest, "invalid_post")
		}
		if input.Visibility != nil {
			switch *input.Visibility {
			case "friends", "friends_of_friends", "public":
			default:
				return NewAPIError(http.StatusBadRequest, "invalid_post")
			}
		}
		ctx := c.Request().Context()
		userID := GetSession(c).UserID
		tx, err := h.DB.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		user, err := lockUser(ctx, tx, userID)
		if err != nil {
			return err
		}
		post, err := getPost(ctx, tx, input.PostID)
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusNotFound, "post_not_found")
		}
		if err != nil {
			return err
		}
		if post.AuthorID != userID {
			return NewAPIError(http.StatusForbidden, "post_not_editable")
		}
		if post.WindowEndedAt != nil {
			return NewAPIError(http.StatusForbidden, "post_not_editable")
		}
		if err := tx.GetContext(ctx, &post, `
   UPDATE posts p SET caption = COALESCE($2, p.caption), visibility = COALESCE($3, p.visibility)
   FROM posting_windows w
   WHERE p.id = $1 AND w.id = p.posting_window_id
   RETURNING `+postColumns, post.ID, input.Caption, input.Visibility); err != nil {
			return err
		}
		if err := user.requireActive(); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, updatePostResponse(newPostResponse(post, user)))
	}
}

type deletePostRequest struct {
	PostID string `param:"postId" json:"-"`
}

func deletePostHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input deletePostRequest
		if err := c.Bind(&input); err != nil {
			return err
		}
		ctx := c.Request().Context()
		userID := GetSession(c).UserID
		tx, err := h.DB.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		var lockedUserID string
		if err := tx.GetContext(ctx, &lockedUserID, `SELECT id FROM users WHERE id = $1 FOR NO KEY UPDATE`, userID); err != nil {
			return err
		}
		var target struct {
			AuthorID  string `db:"author_id"`
			Deletions int    `db:"deletions"`
		}
		// "Posted today" means a post in any window sharing the local_date, regardless of region. The same rule is spelled out separately (not shared, to keep each query plan) in viewerContextSQL, activeWindowWithPostCountsQuery, getMyCurrentPostHandler, getCurrentPostingWindowHandler and deletePostHandler; change them together.
		err = tx.GetContext(ctx, &target, `
   WITH target AS (
     SELECT p.id, p.author_id, (
       SELECT count(*) FROM posts dp JOIN posting_windows dw ON dw.id = dp.posting_window_id
       WHERE dp.author_id = $2 AND dw.local_date = w.local_date AND dp.deleted_at IS NOT NULL
     ) AS deletions
     FROM posts p JOIN posting_windows w ON w.id = p.posting_window_id
     WHERE p.id = $1 AND p.deleted_at IS NULL
   ), deleted AS (
     UPDATE posts SET deleted_at = clock_timestamp()
     WHERE id = (SELECT id FROM target WHERE author_id = $2 AND deletions < 2)
   )
   SELECT author_id, deletions FROM target
  `, input.PostID, userID)
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusNotFound, "post_not_found")
		}
		if err != nil {
			return err
		}
		if target.AuthorID != userID {
			return NewAPIError(http.StatusForbidden, "post_not_editable")
		}
		if target.Deletions >= 2 {
			return NewAPIError(http.StatusConflict, "post_slot_unavailable")
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return c.NoContent(http.StatusNoContent)
	}
}
