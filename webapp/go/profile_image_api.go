package isureal

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"image/jpeg"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
)

func putProfileImageHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if c.Request().Header.Get("Content-Type") != "image/jpeg" {
			return NewAPIError(http.StatusUnsupportedMediaType, "unsupported_media_type")
		}
		data, err := readBodyExact(c.Request())
		if err != nil {
			return err
		}
		img, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			c.Set("abuse", "invalid_image")
			return NewAPIError(http.StatusBadRequest, "invalid_profile_image")
		}
		if img.Bounds().Dx() > 512 || img.Bounds().Dy() > 512 {
			return NewAPIError(http.StatusBadRequest, "invalid_profile_image")
		}

		imageID := uuid.NewString()
		if err := storeUploadedImage(imageID, data); err != nil {
			return err
		}
		ctx := c.Request().Context()
		userID := GetSession(c).UserID
		tx, err := h.DB.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		var old profileImageRef
		if err := tx.GetContext(ctx, &old, lockProfileImageQuery, userID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return NewAPIError(http.StatusNotFound, "account_not_found")
			}
			return err
		}
		var createdAt time.Time
		if err := tx.GetContext(ctx, &createdAt, putProfileImageQuery, binaryImage, userID, imageID, fileBackedImageData, time.Now().UTC()); err != nil {
			return err
		}
		if err := deleteImageObject(ctx, tx, old); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		rememberUploadedImageMeta(imageID, userID, createdAt)
		return c.NoContent(http.StatusNoContent)
	}
}

// profileImageRef は users の行が今指しているプロフィール画像。updated_at は画像を入れたときの時刻なので、
// putProfileImageQuery で入れた画像なら created_at と同じ
type profileImageRef struct {
	ImageID   *string   `db:"profile_image_id"`
	UpdatedAt time.Time `db:"updated_at"`
}

const lockProfileImageQuery = `SELECT profile_image_id, updated_at FROM users WHERE id=$1 FOR NO KEY UPDATE`

// 画像の created_at と users.updated_at に同じ時刻（$4）を入れる。別々に clock_timestamp() を使うと数 ms ずれて、
// imageCreatedAtLookupQuery が返す updated_at では画像が見つからず、全パーティションをロックする id だけの検索に落ちる。
// RETURNING は DB に入った値（マイクロ秒）で、imageCreatedAtCache に入れるのはこちら
const putProfileImageQuery = `
   WITH image AS (INSERT INTO image_objects(id,owner_user_id,data,media_type,created_at) VALUES($2,$1,$3,'image/jpeg',$4))
   UPDATE users SET profile_image_id=$2,updated_at=$4 WHERE id=$1 RETURNING updated_at`

// deleteImageObject は外したプロフィール画像を消す。id だけの DELETE は全パーティションをロックするので、
// まず updated_at を created_at として1パーティションだけ当たり、消えなかったとき（初期データ以外で updated_at が
// 別の更新で動いた行など）だけ id だけで消す。消した後は users が指していないので、残っていても配信されない
func deleteImageObject(ctx context.Context, tx *sqlx.Tx, ref profileImageRef) error {
	if ref.ImageID == nil {
		return nil
	}
	// 消した画像を覚えたままでも visible が偽になって image_objects を引き直すだけだが、念のため忘れる（#79）
	imageMetaCache.delete(*ref.ImageID)
	result, err := tx.ExecContext(ctx, `DELETE FROM image_objects WHERE id=$1 AND created_at=$2`, *ref.ImageID, ref.UpdatedAt)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil || n > 0 {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM image_objects WHERE id=$1`, *ref.ImageID)
	return err
}

func deleteProfileImageHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ctx := c.Request().Context()
		userID := GetSession(c).UserID
		tx, err := h.DB.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		var current profileImageRef
		err = tx.GetContext(ctx, &current, lockProfileImageQuery, userID)
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusNotFound, "account_not_found")
		}
		if err != nil {
			return err
		}
		if current.ImageID == nil {
			return NewAPIError(http.StatusNotFound, "account_not_found")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET profile_image_id=NULL,updated_at=clock_timestamp() WHERE id=$1`, userID); err != nil {
			return err
		}
		if err := deleteImageObject(ctx, tx, current); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return c.NoContent(http.StatusNoContent)
	}
}
