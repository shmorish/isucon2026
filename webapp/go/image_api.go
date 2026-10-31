package isureal

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
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
		// 署名鍵と削除状態を1回で確認する。即時反映も5秒以内の許容範囲に含まれる。
		var meta struct {
			SecretKeyHash string    `db:"secret_key_hash"`
			MediaType     string    `db:"media_type"`
			CreatedAt     time.Time `db:"created_at"`
			Visible       bool      `db:"visible"`
			Data          []byte    `db:"data"`
		}
		// 画像本体は作成後に変わらない（id は UUID で再利用されない）のでメモリに持つ。
		// 持っていなければ同じ問い合わせで data も読む
		data, cached := imageDataCache.get(input.ImageID)
		q := imageMetaQueries[!cached]
		if createdAt, ok := imageCreatedAtCache.get(input.ImageID); ok {
			err = h.DB.GetContext(ctx, &meta, q.byCreatedAt, input.ImageID, createdAt)
		} else {
			err = h.DB.GetContext(ctx, &meta, q.byID, input.ImageID)
		}
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}
		if err != nil {
			return err
		}
		imageCreatedAtCache.put(input.ImageID, meta.CreatedAt)
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
		if !cached {
			data = meta.Data
			imageDataCache.put(input.ImageID, data)
		}
		if exp <= time.Now().Unix() {
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}
		return c.Blob(http.StatusOK, meta.MediaType, data)
	}
}

func imageMetaSelect(cols string) string {
	return `
   SELECT ` + cols + `u.secret_key_hash, i.media_type, i.created_at, (
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
   ) IS TRUE AS visible`
}

type imageMetaQuery struct{ byID, byCreatedAt string }

// image_objects は created_at で分割されていて id だけだと全パーティションの主キーを引く。
// 投稿の画像は created_at = posts.posted_at なので先にそれで1パーティションだけ引き、
// 見つからないとき（RealMoji・プロフィール画像）だけ全パーティションを引く。
// true のほうは data も読む
var imageMetaQueries = map[bool]imageMetaQuery{
	false: newImageMetaQuery(""),
	true:  newImageMetaQuery("i.data, "),
}

func newImageMetaQuery(cols string) imageMetaQuery {
	imgCols := cols + "i.id, i.owner_user_id, i.media_type, i.created_at"
	return imageMetaQuery{
		byID: `
   WITH by_post AS (
     SELECT ` + imgCols + ` FROM image_objects i
     WHERE i.id = $1 AND i.created_at = (SELECT p.posted_at FROM posts p WHERE p.image_id = $1 LIMIT 1)
   ), img AS (
     SELECT * FROM by_post
     UNION ALL
     SELECT ` + imgCols + ` FROM image_objects i
     WHERE i.id = $1 AND NOT EXISTS (SELECT 1 FROM by_post)
   )` + imageMetaSelect(cols) + `
   FROM img i JOIN users u ON u.id = i.owner_user_id
   WHERE u.status = 'active'`,
		byCreatedAt: imageMetaSelect(cols) + `
   FROM image_objects i JOIN users u ON u.id = i.owner_user_id
   WHERE i.id = $1 AND i.created_at = $2 AND u.status = 'active'`,
	}
}

// INSERT に渡した値は入れない。DB 側でマイクロ秒に丸められて一致しなくなるため、DB から読んだ値（RETURNING を含む）だけを覚える。
// id は UUID で再利用されず created_at も変わらないので、初期化で消えた画像の値が残っても該当行なしになるだけ。
const imageCreatedAtCacheMaxEntries = 1 << 20

var imageCreatedAtCache = newBoundedCache(imageCreatedAtCacheMaxEntries, func(time.Time) int { return 1 })

// ponytail: 上限を超えたら全消去する。ヒット率が落ちるようなら LRU にする
// Go のヒープは画像の長さの約1.08倍（実際の投稿・RealMoji のサイズ分布で計測）。
// 全消去直後は GOGC=100 だと最大で生存ヒープの2倍まで膨らむので、メモリ 3.8GB・swap なしでも
// 収まるよう GC の目標を imageHeapMemoryLimit で頭打ちにする
const imageDataCacheMaxBytes = 1 << 30

const imageHeapMemoryLimit = 2560 << 20

func init() {
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(imageHeapMemoryLimit)
	}
}

// アップロードを受けたサーバーでは、コミットが成功してから DB に書いたのと同じバイト列を入れる。
// io.ReadAll の余った容量まで抱えないようにコピーする
func cacheUploadedImage(id string, data []byte) {
	imageDataCache.put(id, bytes.Clone(data))
}

var imageDataCache = newBoundedCache(imageDataCacheMaxBytes, func(v []byte) int { return len(v) })

// 既にあるキーは上書きしない。sizeOf の合計が max を超えるときは新しいマップに替えて全部捨てる
type boundedCache[V any] struct {
	mu     sync.RWMutex
	m      map[string]V
	size   int
	max    int
	sizeOf func(V) int
}

func newBoundedCache[V any](max int, sizeOf func(V) int) *boundedCache[V] {
	return &boundedCache[V]{m: map[string]V{}, max: max, sizeOf: sizeOf}
}

func (c *boundedCache[V]) get(k string) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[k]
	return v, ok
}

func (c *boundedCache[V]) put(k string, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.m[k]; ok {
		return
	}
	n := c.sizeOf(v)
	if c.size+n > c.max {
		c.m = map[string]V{}
		c.size = 0
	}
	c.m[k] = v
	c.size += n
}
