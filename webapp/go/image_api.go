package isureal

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
		var meta imageMetaRow
		// 画像本体は作成後に変わらない（id は UUID で再利用されない）のでメモリに持つ。
		// 持っていなければ同じ問い合わせで data も読む。s1 の nginx 経由でディスクに完全なファイルがあれば、
		// 確認だけしてファイルの送信は nginx に任せる（#76）
		accelPath, accel := "", fromLocalNginx(c.Request())
		if accel {
			accelPath, accel = imageAccelPath(input.ImageID)
		}
		var data []byte
		cached := accel
		if !accel {
			data, cached = getImageData(input.ImageID)
		}
		// owner・media_type・created_at も作成後に変わらないので覚えておき、本体が手元にあるときは image_objects を
		// 引かずに「今見せてよいか」だけを users・posts・realmojis で判定する（#79）。
		// 見せられないときは今までどおり image_objects を引く。消えた画像（差し替え前のプロフィール画像）と
		// 削除された投稿の画像で、署名を確認する前に 403 にするか（abuse の記録の有無）が変わらないようにするため
		known, hit := imageMetaCache.get(input.ImageID)
		if !hit {
			known, hit = lookupInitialImageMeta(input.ImageID)
		}
		checked := false
		// 「見せてよい」と覚えている画像は DB を引かない（#89）。署名の確認は下で必ず行う。
		// それ以外（覚えていない・本体が手元にない）は今までの経路に落とす
		visibilityGen := imageVisibility.generation()
		if v, ok := imageVisibility.get(input.ImageID); ok && cached {
			meta = imageMetaRow{SecretKeyHash: v.SecretKeyHash, OwnerUserID: v.Owner, MediaType: v.MediaType, CreatedAt: v.CreatedAt, Visible: true}
			checked = true
		}
		if !checked && hit && cached {
			var owner imageOwnerRow
			err = h.DB.GetContext(ctx, &owner, imageOwnerQuery, input.ImageID, known.Owner)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if err == nil && owner.Visible {
				meta = imageMetaRow{SecretKeyHash: owner.SecretKeyHash, OwnerUserID: known.Owner, MediaType: known.MediaType, CreatedAt: known.CreatedAt, Visible: true}
				checked = true
				imageVisibility.put(input.ImageID, visibleImage{SecretKeyHash: owner.SecretKeyHash, Owner: known.Owner, MediaType: known.MediaType, CreatedAt: known.CreatedAt}, visibilityGen)
			}
		}
		if !checked {
			q := imageMetaQueryWithData
			if cached {
				q = imageMetaQueryMetaOnly
			}
			createdAt, ok := known.CreatedAt, hit
			if !ok {
				createdAt, ok = imageCreatedAtCache.get(input.ImageID)
			}
			if !ok {
				// 覚えていなければ、画像を参照している行から created_at を引いてくる。id だけの検索は全パーティションを
				// ロックする（1回あたり relation ロック 94 個、fast-path に収まるのは 16 個）ので、最後の手段にする
				var referenced sql.NullTime
				if err := h.DB.GetContext(ctx, &referenced, imageCreatedAtLookupQuery, input.ImageID); err != nil {
					return err
				}
				createdAt, ok = referenced.Time, referenced.Valid
			}
			if ok {
				err = h.DB.GetContext(ctx, &meta, q.byCreatedAt, binaryImage, input.ImageID, createdAt)
			}
			// 引いた created_at で見つからないとき（画像の created_at が参照元の時刻と違うなど）は id だけで引き直す
			if !ok || errors.Is(err, sql.ErrNoRows) {
				err = h.DB.GetContext(ctx, &meta, q.byID, binaryImage, input.ImageID)
			}
			if errors.Is(err, sql.ErrNoRows) {
				return NewAPIError(http.StatusForbidden, "image_forbidden")
			}
			if err != nil {
				return err
			}
			imageCreatedAtCache.put(input.ImageID, meta.CreatedAt)
			imageMetaCache.put(input.ImageID, imageMeta{Owner: meta.OwnerUserID, MediaType: meta.MediaType, CreatedAt: meta.CreatedAt})
			if meta.Visible {
				imageVisibility.put(input.ImageID, visibleImage{SecretKeyHash: meta.SecretKeyHash, Owner: meta.OwnerUserID, MediaType: meta.MediaType, CreatedAt: meta.CreatedAt}, visibilityGen)
			}
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
		if !cached {
			if data, err = imageDataFromDB(input.ImageID, meta.Data); err != nil {
				return err
			}
		}
		if exp <= time.Now().Unix() {
			return NewAPIError(http.StatusForbidden, "image_forbidden")
		}
		if accel {
			// Content-Type と Cache-Control は nginx がそのままクライアントに返す（s1 の isureal.conf の /_image_cache/）
			c.Response().Header().Set(echo.HeaderContentType, meta.MediaType)
			c.Response().Header().Set("X-Accel-Redirect", accelPath)
			return c.NoContent(http.StatusOK)
		}
		return c.Blob(http.StatusOK, meta.MediaType, data)
	}
}

type imageMetaRow struct {
	SecretKeyHash string    `db:"secret_key_hash"`
	OwnerUserID   string    `db:"owner_user_id"`
	MediaType     string    `db:"media_type"`
	CreatedAt     time.Time `db:"created_at"`
	Visible       bool      `db:"visible"`
	Data          []byte    `db:"data"`
}

// imageVisibleExpr は所有者 u の画像 image を今見せてよいかで、imageMetaSelect と imageOwnerQuery で同じ式を使う
func imageVisibleExpr(image string) string {
	return `(
     u.profile_image_id = ` + image + `
     OR EXISTS (
       SELECT 1 FROM posts p
       WHERE p.image_id = ` + image + ` AND p.author_id = u.id AND p.deleted_at IS NULL
     )
     OR EXISTS (
       SELECT 1 FROM realmojis r
       JOIN posts p ON p.id = r.post_id AND p.deleted_at IS NULL
       JOIN users author ON author.id = p.author_id AND author.status = 'active'
       WHERE r.image_id = ` + image + ` AND r.reactor_id = u.id AND r.replaced_at IS NULL AND r.deleted_at IS NULL
     )
   ) IS TRUE AS visible`
}

func imageMetaSelect(cols string) string {
	return `
   SELECT ` + cols + `u.secret_key_hash, i.owner_user_id, i.media_type, i.created_at, ` + imageVisibleExpr("i.id")
}

// imageOwnerQuery は imageMetaQuery の image_objects 抜きの形。$1 の画像が $2 の所有で存在することは imageMetaCache が
// 保証する（owner は作成後に変わらず、id は使い回されない）ので、残りの判定は byID/byCreatedAt と同じになる
var imageOwnerQuery = `
   SELECT u.secret_key_hash, ` + imageVisibleExpr("$1") + `
   FROM users u WHERE u.id = $2 AND u.status = 'active'`

type imageOwnerRow struct {
	SecretKeyHash string `db:"secret_key_hash"`
	Visible       bool   `db:"visible"`
}

// imageMeta は image_objects の変わらない列。DB から読んだ値（RETURNING を含む）だけを入れる（imageCreatedAtCache と同じ理由）
type imageMeta struct {
	Owner     string
	MediaType string
	CreatedAt time.Time
}

var imageMetaCache = newBoundedCache(imageCreatedAtCacheMaxEntries, func(imageMeta) int { return 1 })

// rememberUploadedImageMeta は自分で INSERT した画像を覚える。アプリが入れる画像はすべて image/jpeg
func rememberUploadedImageMeta(id, owner string, createdAt time.Time) {
	imageCreatedAtCache.put(id, createdAt)
	imageMetaCache.put(id, imageMeta{Owner: owner, MediaType: "image/jpeg", CreatedAt: createdAt})
}

// resetImageMetaCaches は /api/initialize で消えた画像の値を捨てる。どちらも DB から作り直せる値だけを持つ
func resetImageMetaCaches() {
	imageCreatedAtCache.reset()
	imageMetaCache.reset()
	imageVisibility.revokeAll()
}

type imageMetaQuery struct{ byID, byCreatedAt string }

// image_objects は created_at で分割されていて id だけだと全パーティションの主キーを引く。
// created_at を引数で渡したときだけプランナが1パーティションに絞る（サブクエリで求めた値では、実行時に
// 枝は刈られてもプランに残った全パーティションをロックする）ので、先に参照元の行から created_at を取ってから引く。
var (
	imageMetaQueryMetaOnly = newImageMetaQuery("")
	imageMetaQueryWithData = newImageMetaQuery("i.data, ")
)

// imageCreatedAtLookupQuery は画像を参照している行の時刻を返す。投稿の画像は createPostQuery が posted_at で
// INSERT する。初期データでは RealMoji の画像は realmojis.created_at、プロフィール画像は users.updated_at と
// 一致する（s4 で確認。ベンチ中に作られる RealMoji・プロフィール画像は数 ms ずれるので byID に落ちる）。
// 違っていても byCreatedAt が空になって byID で引き直すだけなので、結果は変わらない
const imageCreatedAtLookupQuery = `
   SELECT COALESCE(
     (SELECT p.posted_at FROM posts p WHERE p.image_id = $1 LIMIT 1),
     (SELECT r.created_at FROM realmojis r WHERE r.image_id = $1 LIMIT 1),
     (SELECT u.updated_at FROM users u WHERE u.profile_image_id = $1 LIMIT 1))`

func newImageMetaQuery(cols string) imageMetaQuery {
	return imageMetaQuery{
		byID: imageMetaSelect(cols) + `
   FROM image_objects i JOIN users u ON u.id = i.owner_user_id
   WHERE i.id = $1 AND u.status = 'active'`,
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
// 画像本体はディスクにもある（#65）ので、メモリは 256MiB に抑えて OS のページキャッシュに回す（#71）。
// 1GiB のときは s1 のページキャッシュが 1GB 前後しかなく、画像ファイルを読み直して iowait 7.8%・kswapd が出た
const imageDataCacheMaxBytes = 256 << 20

const imageHeapMemoryLimit = 2560 << 20

func init() {
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(imageHeapMemoryLimit)
	}
}

// アップロードされた画像の本体は DB に入れず、s1 のディスクにだけ置く（#81）。DB の image_objects には空の data で行を入れる。
// bytea の INSERT が投稿作成の大半（平均 180〜250ms、WAL と TOAST の拡張ロック）を占めていたため。
// API の処理は s1 だけ（s2・s3 の nginx は /api/ を s1 に送る）なので、ファイルは s1 にあればよい。
// 再起動後も読めるよう、DB に行を入れる前に fsync してから rename し、ディレクトリも fsync する。
// DB の書き込みが失敗したときのファイルは残るが、DB に行がないので 403 になるだけ
var fileBackedImageData = []byte{}

func storeUploadedImage(id string, data []byte) error {
	path, ok := imageDiskPath(id)
	if !ok {
		return errors.New("invalid image id")
	}
	if err := writeImageFileDurable(path, data); err != nil {
		return err
	}
	// 見てよいかは毎回 DB で判定するので、コミット前に入れてよい。読み込みは Content-Length で1回だけ確保するので
	// （upload_read.go）余りは数 KB。長さが分からず伸ばしながら読んだときだけ、余った容量まで抱えないようにコピーする
	if cap(data)-len(data) > uploadSlackBytes {
		data = bytes.Clone(data)
	}
	imageDataCache.put(id, data)
	return nil
}

func writeImageFileDurable(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	err = tmp.Chmod(0o644)
	if err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// imageDataFromDB は DB から読んだ data を返す。空ならアップロードされた画像なので、ディスクから読む
func imageDataFromDB(id string, data []byte) ([]byte, error) {
	if len(data) > 0 {
		putImageData(id, data)
		return data, nil
	}
	if data, ok := getImageData(id); ok {
		return data, nil
	}
	return nil, errors.New("image file missing: " + id)
}

// 画像本体はディスクにも持つ（#65）。本体は作成後に変わらず id も使い回されないので古くならない。
// 見てよいか・削除済みかは毎回 DB で判定するので、/initialize で消えた画像のファイルが残っても 403 になる。
// ponytail: ファイルは消さない（ベンチ1回で約 700MB 増える。ディスクの空きは 34GB）。足りなくなったら initialize で掃除する
var imageDiskCacheDir = envOrDefault("ISUREAL_IMAGE_CACHE_DIR", "/home/isucon/image-cache")

// id は URL から来るので、UUID の形のものだけファイルにする（パスの細工を防ぐ）
var imageIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func imageDiskPath(id string) (string, bool) {
	if !imageIDPattern.MatchString(id) {
		return "", false
	}
	return filepath.Join(imageDiskCacheDir, id[:2], id), true
}

// getImageData はメモリ、なければディスクから画像本体を返す
func getImageData(id string) ([]byte, bool) {
	if data, ok := imageDataCache.get(id); ok {
		return data, true
	}
	path, ok := imageDiskPath(id)
	if !ok {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil || !completeImage(data) {
		return nil, false
	}
	imageDataCache.put(id, data)
	return data, true
}

// X-Accel-Redirect は受け取った nginx が解釈する。ファイルがあるのは s1 だけで、s2・s3 の nginx は s1 の
// アプリ (:8080) に直接転送してくるので、s1 の nginx が付ける X-Isureal-Image-Accel と、相手が同じマシンで
// あることの両方を見る。s2・s3 経由ではクライアントが付けたヘッダーがそのまま届くが、送り元が s1 ではないので使われない
func fromLocalNginx(r *http.Request) bool {
	if r.Header.Get("X-Isureal-Image-Accel") != "1" {
		return false
	}
	addr, err := netip.ParseAddrPort(r.RemoteAddr)
	return err == nil && addr.Addr().IsLoopback()
}

// imageAccelPath は nginx に渡してよい画像ファイルの内部 URI を返す（s1 の isureal.conf の /_image_cache/）。
// 本体を読まずに、先頭と末尾の 8 バイトだけで completeImage と同じ判定をする。
// nginx (www-data) が読めない 0600 のファイル（#76 より前に書いたもの）はアプリが今までどおり返す
func imageAccelPath(id string) (string, bool) {
	f, st, ok := openCompleteImageFile(id)
	if !ok {
		return "", false
	}
	f.Close()
	if st.Mode().Perm()&0o004 == 0 {
		return "", false
	}
	return "/_image_cache/" + id[:2] + "/" + id, true
}

// openCompleteImageFile は画像ファイルを開き、本体を読まずに先頭と末尾の 8 バイトだけで completeImage と同じ判定をする
func openCompleteImageFile(id string) (*os.File, os.FileInfo, bool) {
	path, ok := imageDiskPath(id)
	if !ok {
		return nil, nil, false
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, false
	}
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		f.Close()
		return nil, nil, false
	}
	// PNG の先頭・末尾の印がどちらも 8 バイトなので、16 バイトあれば足りる
	b := make([]byte, min(st.Size(), 16))
	if len(b) < 16 {
		_, err = f.ReadAt(b, 0)
	} else if _, err = f.ReadAt(b[:8], 0); err == nil {
		_, err = f.ReadAt(b[8:], st.Size()-8)
	}
	if err != nil || !completeImage(b) {
		f.Close()
		return nil, nil, false
	}
	return f, st, true
}

var (
	jpegStart, jpegEnd = []byte{0xff, 0xd8}, []byte{0xff, 0xd9}
	pngStart, pngEnd   = []byte("\x89PNG\r\n\x1a\n"), []byte("IEND\xaeB`\x82")
)

// completeImage は画像が最後まで書けているかを見る。ファイルは fsync しないので、電源断のあとに
// 長さ0や途中までのファイルが残りうる。そういうファイルは使わずに DB から読み直す
// （s4 で抜き出した 914枚はすべて JPEG で、すべて FFD8 で始まり FFD9 で終わっていた）
func completeImage(b []byte) bool {
	switch {
	case bytes.HasPrefix(b, jpegStart):
		return bytes.HasSuffix(b, jpegEnd)
	case bytes.HasPrefix(b, pngStart):
		return bytes.HasSuffix(b, pngEnd)
	}
	return false
}

// putImageData は DB から読んだ（またはコミット済みの）画像本体をメモリに入れ、裏でディスクに書く
func putImageData(id string, data []byte) {
	imageDataCache.put(id, data)
	go writeImageFile(id, data)
}

// 一時ファイルに書いてから rename する。fsync はしない（1枚ごとの fsync で s1 の iowait が 0.2% → 11% に上がった）。
// 電源断で中身が欠けたファイルは、読むときに completeImage で弾く
func writeImageFile(id string, data []byte) {
	path, ok := imageDiskPath(id)
	if !ok {
		return
	}
	if _, err := os.Stat(path); err == nil {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return
	}
	// CreateTemp は 0600 で作る。nginx (www-data) が X-Accel-Redirect で読めるよう、rename の前に 0644 にする
	err = tmp.Chmod(0o644)
	if err == nil {
		_, err = tmp.Write(data)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
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

func (c *boundedCache[V]) delete(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.m[k]; ok {
		c.size -= c.sizeOf(v)
		delete(c.m, k)
	}
}

func (c *boundedCache[V]) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = map[string]V{}
	c.size = 0
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
