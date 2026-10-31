package isureal

import (
	"bytes"
	"database/sql"
	"errors"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"strings"
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

// readCreatePostInput returns the 400s (and image read errors) that come before any DB access.
func readCreatePostInput(c *echo.Context) (createPostRequest, PostLocation, []byte, error) {
	var input createPostRequest
	// 画像はここで1回のバッファに読み、c.Bind は読み終えたフォームを束縛する（upload_read.go）
	imageData, err := readMultipartUpload(c.Request(), "image")
	if err != nil {
		return input, PostLocation{}, nil, echo.ErrBadRequest.Wrap(err)
	}
	if err := c.Bind(&input); err != nil {
		return input, PostLocation{}, nil, err
	}
	if input.Image == nil || !validUserText(input.Caption, 0, 1000) {
		return input, PostLocation{}, nil, NewAPIError(http.StatusBadRequest, "invalid_post")
	}
	switch input.Visibility {
	case "friends", "friends_of_friends", "public":
	default:
		return input, PostLocation{}, nil, NewAPIError(http.StatusBadRequest, "invalid_post")
	}
	location := PostLocation{
		Kind: input.LocationKind, Latitude: input.Latitude,
		Longitude: input.Longitude, Locality: input.Locality,
	}
	if !location.valid() {
		return input, location, nil, NewAPIError(http.StatusBadRequest, "invalid_post")
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return input, location, nil, NewAPIError(http.StatusBadRequest, "invalid_post")
	}
	if cfg.Width > 4096 || cfg.Height > 4096 {
		return input, location, nil, NewAPIError(http.StatusBadRequest, "invalid_post")
	}
	return input, location, imageData, nil
}

// createPostQuery inserts the image and the post only when the user's active window has a free slot (the checks of
// activeWindowWithPostCountsQuery, whose $1/$2 are the user id and region). It runs after the user row is locked, so it
// sees posts committed while waiting for the lock. No row back means no free slot.
const createPostQuery = `
  WITH win AS (` + activeWindowWithPostCountsQuery + `),
  ok AS (SELECT id FROM win WHERE active = 0 AND deleted < 2 AND starts_at <= $5::timestamptz),
  image AS (
    INSERT INTO image_objects (id, owner_user_id, data, media_type, created_at)
    SELECT $3::varchar, $1::varchar, $4::bytea, 'image/jpeg', $5::timestamptz FROM ok
  ), p AS (
    INSERT INTO posts (id, author_id, posting_window_id, image_id, caption, visibility,
                       location_kind, latitude, longitude, locality, posted_at)
    SELECT $6::varchar, $1::varchar, ok.id, $3::varchar, $7::text, $8::text,
           $9::text, $10::numeric, $11::numeric, $12::text, $5::timestamptz FROM ok
    RETURNING *
  )
  SELECT ` + postColumns + ` FROM p JOIN posting_windows w ON w.id = p.posting_window_id
`

// createPostLockSQL looks the session up, locks the user's row (FOR NO KEY UPDATE, like lockUser) and then takes the
// region's shared lockRegionPosts in one statement. The advisory lock is in the select list so it is taken only after
// the row lock, keyed by the locked row's region (the one createPostQuery gets), and before createPostQuery's
// snapshot, so that query sees a window the opener ended while this waited.
var createPostLockSQL = `
  WITH ` + sessionCTE + `
  SELECT ` + sessionSelect + `, u.*, ` + regionPostsSharedLockExpr("u.region") + `::text AS region_posts_lock
  FROM sess CROSS JOIN LATERAL (
    SELECT ` + strings.Join(userColumns, ", ") + ` FROM users WHERE id = sess.user_id FOR NO KEY UPDATE
  ) u
`

// createPostHandler sends BEGIN, the session lookup with the user lock, the guarded INSERT and COMMIT: 4 round trips
// (was 6 with the separate session lookup and window query; ~100ms each on s2/s3).
func createPostHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		input, location, imageData, err := readCreatePostInput(c)
		if err := foldedSessionBefore(c, h, err); err != nil {
			return err
		}

		ctx := c.Request().Context()
		tx, err := h.DB.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		var locked struct {
			sessionColumns
			User
			RegionPostsLock string `db:"region_posts_lock"`
		}
		err = tx.GetContext(ctx, &locked, createPostLockSQL, foldedSessionTokenHash(c))
		if err := foldedSessionResult(c, h, err, locked.sessionColumns); err != nil {
			return err
		}
		user := locked.User
		if err := user.requireActive(); err != nil {
			return err
		}
		postedAt := time.Now().UTC()
		imageID := uuid.NewString()
		// 鍵と画像 id はここで決まるので、レスポンスの署名は INSERT・COMMIT と並行して取り寄せる（#77）。
		// 取り寄せに失敗しても newPostResponse が手元で計算するだけ
		signed := startPrefetchImageSignatures(ctx, appendImagesToSign(nil, user, imageID))
		defer signed()
		args := []any{binaryImage, user.ID, user.Region, imageID, imageData, postedAt, uuid.NewString(), input.Caption, input.Visibility,
			location.Kind, location.Latitude, location.Longitude, location.Locality}
		var post Post
		err = tx.GetContext(ctx, &post, createPostQuery, args...)
		// 重複に見えるが消さない。FOR SHARE で待つ間に投稿期間が終わると、READ COMMITTED の再確認で
		// 行が外れて0行になる。次の文は新しいスナップショットで始まり、新しく開いた期間を見る。
		if errors.Is(err, sql.ErrNoRows) {
			err = tx.GetContext(ctx, &post, createPostQuery, args...)
		}
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusConflict, "post_slot_unavailable")
		}
		if err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		cacheUploadedImage(imageID, imageData)
		rememberUploadedImageMeta(imageID, user.ID, post.PostedAt)
		signed()
		return c.JSON(http.StatusCreated, newPostResponse(post, user))
	}
}

func newPostResponse(post Post, user User) createPostResponse {
	// 投稿の画像の created_at は posted_at と同じ値で入れている。DB から読んだ posted_at を覚えておくと、
	// 続く GET /api/images が created_at で1パーティションだけを引ける（#63）
	imageCreatedAtCache.put(post.ImageID, post.PostedAt)
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
		if err := foldedSessionResult(c, h, err, row.sessionColumns); err != nil {
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
		prefetchImageSignatures(ctx, appendImagesToSign(nil, user, post.ImageID))
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
		prefetchImageSignatures(ctx, appendImagesToSign(nil, post.Author, post.ImageID))
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
		prefetchImageSignatures(ctx, appendImagesToSign(nil, user, post.ImageID))
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
		if err := lockUserRow(ctx, tx, userID); err != nil {
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
