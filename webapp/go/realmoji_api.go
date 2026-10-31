package isureal

import (
	"bytes"
	"database/sql"
	"errors"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type replaceRealMojiRequest struct {
	PostID       string                `param:"postId" json:"-"`
	Image        *multipart.FileHeader `form:"image"`
	ReactionType string                `form:"reactionType"`
}

type replaceRealMojiResponse struct {
	ID           string                   `json:"id"`
	Reactor      relationshipUserResponse `json:"reactor"`
	ReactionType string                   `json:"reactionType"`
	ImageURL     string                   `json:"imageUrl"`
	CreatedAt    time.Time                `json:"createdAt"`
}

// 署名する画像を変えたら appendImagesToSign の呼び出し側も揃える
func newRealMojiResponse(reaction RealMoji, user User) replaceRealMojiResponse {
	return replaceRealMojiResponse{ID: reaction.ID, Reactor: newRelationshipUserResponse(user), ReactionType: reaction.ReactionType, ImageURL: signedImageURL(user, reaction.ImageID), CreatedAt: reaction.CreatedAt}
}

// 画像と RealMoji は同じ created_at（Go で1回取った時刻）で入れる。別々に clock_timestamp() を使うと数 ms ずれて、
// imageCreatedAtLookupQuery が返す realmojis.created_at では画像が見つからず、全パーティションをロックする
// id だけの検索に落ちる。レスポンスの createdAt は RETURNING で DB に入った値（マイクロ秒）を返す
const (
	replaceRealMojiImageQuery = `
   WITH image AS (INSERT INTO image_objects(id,owner_user_id,data,media_type,created_at) VALUES($3,$2,$4,'image/jpeg',$5))
   UPDATE realmojis SET replaced_at=clock_timestamp() WHERE post_id=$1 AND reactor_id=$2 AND replaced_at IS NULL AND deleted_at IS NULL`
	createRealMojiQuery = `INSERT INTO realmojis(id,post_id,reactor_id,image_id,reaction_type,created_at) VALUES($1,$2,$3,$4,$5,$6) RETURNING ` + realMojiColumns
)

// readReplaceRealMojiInput returns the 400s (and image read errors) that come before any DB access.
func readReplaceRealMojiInput(c *echo.Context) (replaceRealMojiRequest, []byte, error) {
	var input replaceRealMojiRequest
	// 画像はここで1回のバッファに読み、c.Bind は読み終えたフォームを束縛する（upload_read.go）
	data, err := readMultipartUpload(c.Request(), "image")
	if err != nil {
		return input, nil, echo.ErrBadRequest.Wrap(err)
	}
	if err := c.Bind(&input); err != nil {
		return input, nil, err
	}
	if input.Image == nil {
		return input, nil, NewAPIError(http.StatusBadRequest, "invalid_realmoji")
	}
	switch input.ReactionType {
	case "like", "smile", "surprise", "heart", "laugh", "instant":
	default:
		return input, nil, NewAPIError(http.StatusBadRequest, "invalid_realmoji")
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return input, nil, NewAPIError(http.StatusBadRequest, "invalid_realmoji")
	}
	if cfg.Width > 512 || cfg.Height > 512 {
		return input, nil, NewAPIError(http.StatusBadRequest, "invalid_realmoji")
	}
	return input, data, nil
}

func replaceRealMojiHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		input, data, err := readReplaceRealMojiInput(c)
		if err != nil {
			return err
		}

		ctx := c.Request().Context()
		actor := GetSession(c).UserID
		tx, post, err := beginTxWithPost(ctx, h.DB, input.PostID)
		if tx != nil {
			defer tx.Rollback()
		}
		if err != nil {
			return err
		}
		if actor == post.AuthorID {
			return NewAPIError(http.StatusForbidden, "interaction_forbidden")
		}
		if err := validateUserPair(actor, post.AuthorID); err != nil {
			return err
		}
		if err := lockUserPair(ctx, tx, actor, post.AuthorID); err != nil {
			return err
		}
		current, viewer, found, err := loadPostAndViewer(ctx, tx, actor, post.ID)
		if err != nil {
			return err
		}
		if !found || !viewer.canView(current.Post) {
			return NewAPIError(http.StatusForbidden, "interaction_forbidden")
		}
		imageID := uuid.NewString()
		createdAt := time.Now().UTC()
		// 鍵と画像 id はここで決まるので、レスポンスの署名は INSERT・COMMIT と並行して取り寄せる（#77）
		signed := startPrefetchImageSignatures(ctx, appendImagesToSign(nil, viewer.User, imageID))
		defer signed()
		// The new realmoji is inserted in a separate statement: in one statement the
		// replacement UPDATE could run after it and hit realmojis_one_active.
		if _, err := tx.ExecContext(ctx, replaceRealMojiImageQuery, binaryImage, post.ID, actor, imageID, data, createdAt); err != nil {
			return err
		}
		var reaction RealMoji
		if err := tx.GetContext(ctx, &reaction, createRealMojiQuery, uuid.NewString(), post.ID, actor, imageID, input.ReactionType, createdAt); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		cacheUploadedImage(imageID, data)
		rememberUploadedImageMeta(imageID, actor, reaction.CreatedAt)
		signed()
		return c.JSON(http.StatusCreated, newRealMojiResponse(reaction, viewer.User))
	}
}

type getRealMojisRequest struct {
	PostID          string     `param:"postId"`
	BeforeCreatedAt *time.Time `query:"beforeCreatedAt"`
	BeforeID        string     `query:"beforeId"`
}

type getRealMojisResponse struct {
	Items   []replaceRealMojiResponse `json:"items"`
	HasMore bool                      `json:"hasMore"`
}

func getRealMojisHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input getRealMojisRequest
		if err := c.Bind(&input); err != nil {
			return err
		}
		if (input.BeforeCreatedAt == nil) != (input.BeforeID == "") {
			return NewAPIError(http.StatusBadRequest, "invalid_pagination")
		}
		ctx := c.Request().Context()
		// 一覧は権限の確認と同時に取り、権限がなければ捨てる
		var reactions []RealMojiWithReactor
		var listErr error
		var wg sync.WaitGroup
		wg.Go(func() {
			reactions, listErr = selectCurrentRealMojis(ctx, h.DB, `AND r.post_id=$1 AND ($2::timestamptz IS NULL OR (r.created_at,r.id)<($2,$3)) ORDER BY r.created_at DESC,r.id DESC LIMIT 51`, input.PostID, input.BeforeCreatedAt, input.BeforeID)
		})
		allowed, err := canViewPost(ctx, h.DB, GetSession(c).UserID, input.PostID)
		wg.Wait()
		if err != nil {
			return err
		}
		if !allowed {
			return NewAPIError(http.StatusForbidden, "interaction_forbidden")
		}
		if err := listErr; err != nil {
			return err
		}
		hasMore := len(reactions) > 50
		if hasMore {
			reactions = reactions[:50]
		}
		toSign := make([]imageToSign, 0, 2*len(reactions))
		for _, reaction := range reactions {
			toSign = appendImagesToSign(toSign, reaction.Reactor, reaction.ImageID)
		}
		prefetchImageSignatures(ctx, toSign)
		items := make([]replaceRealMojiResponse, 0, len(reactions))
		for _, reaction := range reactions {
			items = append(items, newRealMojiResponse(reaction.RealMoji, reaction.Reactor))
		}
		return c.JSON(http.StatusOK, getRealMojisResponse{Items: items, HasMore: hasMore})
	}
}

type deleteRealMojiRequest struct {
	RealMojiID string `param:"realMojiId" json:"-"`
}

func deleteRealMojiHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input deleteRealMojiRequest
		if err := c.Bind(&input); err != nil {
			return err
		}
		ctx := c.Request().Context()
		actor := GetSession(c).UserID
		tx, err := h.DB.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if err := lockUserRow(ctx, tx, actor); err != nil {
			return err
		}
		var record struct {
			ReactorID string `db:"reactor_id"`
			AuthorID  string `db:"author_id"`
		}
		err = tx.GetContext(ctx, &record, `SELECT r.reactor_id,p.author_id FROM realmojis r JOIN posts p ON p.id=r.post_id WHERE r.id=$1 AND r.deleted_at IS NULL FOR UPDATE OF r`, input.RealMojiID)
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusNotFound, "interaction_not_found")
		}
		if err != nil {
			return err
		}
		if actor != record.ReactorID && actor != record.AuthorID {
			return NewAPIError(http.StatusForbidden, "interaction_forbidden")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE realmojis SET deleted_at=clock_timestamp() WHERE id=$1`, input.RealMojiID); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return c.NoContent(http.StatusNoContent)
	}
}
