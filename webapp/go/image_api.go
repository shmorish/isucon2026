package isureal

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
)

type getImageRequest struct {
	ImageID string `param:"imageId" json:"-"`
	Exp     string `query:"exp" json:"-"`
	Sig     string `query:"sig" json:"-"`
}

func getImageHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		c.Response().Header().Set("Cache-Control", "no-store")
		var input getImageRequest
		if err := c.Bind(&input); err != nil {
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}
		query, err := url.ParseQuery(c.Request().URL.RawQuery)
		if err != nil || len(query["exp"]) != 1 || len(query["sig"]) != 1 {
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}
		exp, err := strconv.ParseInt(input.Exp, 10, 64)
		if err != nil || strconv.FormatInt(exp, 10) != input.Exp || exp <= time.Now().Unix() {
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}
		sig, err := base64.RawURLEncoding.Strict().DecodeString(input.Sig)
		if err != nil || len(sig) != sha256.Size || base64.RawURLEncoding.EncodeToString(sig) != input.Sig {
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}

		ctx := c.Request().Context()
		// 署名鍵と削除状態を1回で確認する（data は読まない）。即時反映も5秒以内の許容範囲に含まれる。
		var meta struct {
			SecretKeyHash string `db:"secret_key_hash"`
			MediaType     string `db:"media_type"`
			Visible       bool   `db:"visible"`
		}
		err = h.DB.GetContext(ctx, &meta, `
   SELECT u.secret_key_hash, i.media_type, (
     u.profile_image_id = i.id
     OR EXISTS (
       SELECT 1 FROM posts p
       WHERE p.image_id = i.id AND p.author_id = u.id AND p.deleted_at IS NULL
     )
     OR EXISTS (
       SELECT 1 FROM realmojis r
       JOIN posts p ON p.id = r.post_id AND p.deleted_at IS NULL
       JOIN users author ON author.id = p.author_id AND author.status = 'active'
       WHERE r.image_id = i.id AND r.reactor_id = u.id AND r.replaced_at IS NULL AND r.deleted_at IS NULL
     )
   ) IS TRUE AS visible
   FROM image_objects i JOIN users u ON u.id = i.owner_user_id
   WHERE i.id = $1 AND u.status = 'active'
  `, input.ImageID)
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}
		if err != nil {
			return err
		}
		key, err := imageSigningKey(meta.SecretKeyHash)
		if err != nil {
			return err
		}
		if !hmac.Equal(sig, imageSignature(key, input.ImageID, exp)) {
			c.Set("abuse", "invalid_signature")
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}
		if !meta.Visible {
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}

		// 画像本体は作成後に変わらない（id は UUID で再利用されない）のでメモリに持つ
		data, ok := imageDataCache.get(input.ImageID)
		if !ok {
			err = h.DB.GetContext(ctx, &data, `SELECT data FROM image_objects WHERE id = $1`, input.ImageID)
			if errors.Is(err, sql.ErrNoRows) {
				return NewAPIError(http.StatusForbidden, "image_forbidden")
			}
			if err != nil {
				return err
			}
			imageDataCache.put(input.ImageID, data)
		}
		if exp <= time.Now().Unix() {
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}
		return c.Blob(http.StatusOK, meta.MediaType, data)
	}
}

// ponytail: 上限を超えたら全消去する。ヒット率が落ちるようなら LRU にする
const imageDataCacheMaxBytes = 512 << 20

var imageDataCache = &bytesCache{m: map[string][]byte{}}

type bytesCache struct {
	mu   sync.RWMutex
	m    map[string][]byte
	size int
}

func (c *bytesCache) get(k string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[k]
	return v, ok
}

func (c *bytesCache) put(k string, v []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.m[k]; ok {
		return
	}
	if c.size+len(v) > imageDataCacheMaxBytes {
		c.m = map[string][]byte{}
		c.size = 0
	}
	c.m[k] = v
	c.size += len(v)
}
