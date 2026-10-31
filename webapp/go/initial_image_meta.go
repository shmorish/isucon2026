package isureal

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// 初期データの画像（約 183万枚）の変わらない列を s1 のメモリに持つ。画像メタの問い合わせが DB 時間の1位
// （約2割、ディスク読み 72 秒）で、#79 の imageMetaCache はベンチ中に初めて見る初期画像には当たらなかった。
// これに当たれば image_objects を引かず、今見せてよいかだけを imageOwnerQuery（users・posts・realmojis）で判定する。
//
// 起動時に裏で1回読む。/api/initialize では作り直さない: 中身は初期データの画像の id・所有者・形式・作成時刻で、
// init.sh が同じ値に戻す。ベンチ中に消えた画像（差し替え前のプロフィール画像）の値が残っても、imageOwnerQuery が
// 見せられないと答え、今までどおり image_objects を引く経路に落ちるので結果は変わらない。
//
// 183万件の map[string] は GC が毎回たどるので、キーを [16]byte にし、値もポインタを持たない形にする（約 90MB）
type initialImageMetaTable struct {
	byID   map[[16]byte]initialImageMetaEntry
	owners []string
	types  []string
}

type initialImageMetaEntry struct {
	createdAt int64 // UnixMicro。PostgreSQL の timestamptz と同じ精度
	owner     uint32
	mediaType uint8
}

var initialImageMeta atomic.Pointer[initialImageMetaTable]

func lookupInitialImageMeta(id string) (imageMeta, bool) {
	t := initialImageMeta.Load()
	if t == nil {
		return imageMeta{}, false
	}
	key, err := uuid.Parse(id)
	if err != nil {
		return imageMeta{}, false
	}
	e, ok := t.byID[key]
	if !ok {
		return imageMeta{}, false
	}
	return imageMeta{Owner: t.owners[e.owner], MediaType: t.types[e.mediaType], CreatedAt: time.UnixMicro(e.createdAt).UTC()}, true
}

func loadInitialImageMeta(ctx context.Context, db *sqlx.DB, logger *slog.Logger) error {
	start := time.Now()
	rows, err := db.QueryxContext(ctx, `
  SELECT id, owner_user_id, media_type, created_at FROM image_objects
  WHERE created_at <= (SELECT created_at FROM initial_data_boundary)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	t := &initialImageMetaTable{byID: make(map[[16]byte]initialImageMetaEntry, 1<<21)}
	ownerIdx := map[string]uint32{}
	typeIdx := map[string]uint8{}
	for rows.Next() {
		var id, owner, mediaType string
		var createdAt time.Time
		if err := rows.Scan(&id, &owner, &mediaType, &createdAt); err != nil {
			return err
		}
		key, err := uuid.Parse(id)
		if err != nil {
			continue
		}
		o, ok := ownerIdx[owner]
		if !ok {
			o = uint32(len(t.owners))
			ownerIdx[owner] = o
			t.owners = append(t.owners, owner)
		}
		m, ok := typeIdx[mediaType]
		if !ok {
			if len(t.types) == 255 {
				continue
			}
			m = uint8(len(t.types))
			typeIdx[mediaType] = m
			t.types = append(t.types, mediaType)
		}
		t.byID[key] = initialImageMetaEntry{createdAt: createdAt.UnixMicro(), owner: o, mediaType: m}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	initialImageMeta.Store(t)
	logger.Info("loaded initial image metadata", "images", len(t.byID), "owners", len(t.owners), "elapsed", time.Since(start))
	return nil
}
