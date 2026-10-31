package isureal

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// 画像を今見せてよいか（#89）は閲覧者によらず、署名の鍵（所有者の secret_key_hash）・所有者が active・画像がまだ
// 参照されていること、だけで決まる。/api の書き込みはすべて s1 のアプリが処理する（s2・s3 の nginx は画像を含む
// /api を s1 に送る）ので、見えなくなる変更もすべて s1 で起きる。そこで「見せてよい」と DB で確かめた画像を覚えておき、
// 見えなくなる書き込み（投稿削除・プロフィール画像の差し替えと削除・RealMoji の差し替えと削除・退会・初期化）で
// コミットの後、応答する前に消す。覚えていない画像は今までどおり DB で確かめる。
//
// 書き込みを処理するアプリが1台だけのときにしか正しくないので、署名を s2・s3 に頼む s1（ISUREAL_IMAGE_SIGNERS あり）か
// ISUREAL_IMAGE_VISIBILITY_CACHE=1 のときだけ使う。
var imageVisibilityCacheEnabled = len(imageSigners) > 0 || os.Getenv("ISUREAL_IMAGE_VISIBILITY_CACHE") == "1"

// 覚えるのは DB から読んだ、作成後に変わらない値だけ（secret_key_hash は登録後に変わらない）
type visibleImage struct {
	SecretKeyHash string
	Owner         string
	MediaType     string
	CreatedAt     time.Time
}

// visibleImageCache の gen は取り消しのたびに進む。DB で確かめ始めたときの gen と入れるときの gen が違えば入れない。
// 取り消しの前に始まった問い合わせが（コミット前のスナップショットで）「見せてよい」を読んで、取り消しの後に
// 入れてしまうのを防ぐため
type visibleImageCache struct {
	mu  sync.RWMutex
	gen uint64
	m   map[string]visibleImage
}

var imageVisibility = &visibleImageCache{m: map[string]visibleImage{}}

func (c *visibleImageCache) get(id string) (visibleImage, bool) {
	if !imageVisibilityCacheEnabled {
		return visibleImage{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[id]
	return v, ok
}

func (c *visibleImageCache) generation() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gen
}

// put は gen が start から進んでいないときだけ入れる
func (c *visibleImageCache) put(id string, v visibleImage, start uint64) {
	if !imageVisibilityCacheEnabled {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gen != start {
		return
	}
	// ponytail: 上限を超えたら全消去する（imageMetaCache と同じ）
	if len(c.m) >= imageCreatedAtCacheMaxEntries {
		c.m = map[string]visibleImage{}
	}
	c.m[id] = v
}

func (c *visibleImageCache) revoke(ids ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gen++
	for _, id := range ids {
		delete(c.m, id)
	}
}

func (c *visibleImageCache) revokeAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gen++
	c.m = map[string]visibleImage{}
}

// revokePostImages は削除した投稿の画像と、その投稿に付いた RealMoji の画像を忘れる（RealMoji の画像は投稿が
// 消えると見えなくなる）。引けなければ全部忘れる
func revokePostImages(ctx context.Context, db *sqlx.DB, postID string) {
	var ids []string
	err := db.SelectContext(context.WithoutCancel(ctx), &ids, `
   SELECT image_id FROM posts WHERE id = $1
   UNION ALL
   SELECT image_id FROM realmojis WHERE post_id = $1`, postID)
	if err != nil {
		slog.Warn("revoke post images: falling back to revoking all", "post", postID, "error", err)
		imageVisibility.revokeAll()
		return
	}
	imageVisibility.revoke(ids...)
}
