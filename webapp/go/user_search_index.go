package isureal

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
)

// ユーザー検索の候補を s1 のメモリで絞る。検索（GET /api/users）は DB 時間の2位（約14%、1回 21.8ms）で、
// 中身は 5,000 行あまりの users に対する部分一致（pg_trgm）だった。
//
// ここは「一致しうるユーザーの id」を多めに返すだけで、状態・ブロック・並び順・ページング・部分一致の最終判定は
// 今までと同じ SQL が DB で行う（id = ANY を足すだけ）。そのため、退会や表示名の変更でここが古くても、
// 余分な候補が DB で落ちるだけで結果は変わらない。足りない候補だけは結果を変えるので、
// 一致しうる文字列（username と、これまでの display_name_search）は消さずに足していき、
// DB に書く前（コミット前）に足す。/api/initialize の後と起動時に DB から作り直す。
// ponytail: 作り直しは中身を丸ごと差し替えるので、initialize と同時に走った登録の候補は落ちうる。initialize はベンチの負荷の前に1回だけ
type userSearchIndex struct {
	mu     sync.RWMutex
	loaded bool
	texts  map[string][]string // user id → 一致しうる文字列
}

var userSearch = &userSearchIndex{texts: map[string][]string{}}

func (x *userSearchIndex) load(ctx context.Context, db *sqlx.DB) error {
	var rows []struct {
		ID       string  `db:"id"`
		Username string  `db:"username"`
		Search   *string `db:"display_name_search"`
	}
	if err := db.SelectContext(ctx, &rows, `SELECT id, username, display_name_search FROM users`); err != nil {
		return err
	}
	texts := make(map[string][]string, len(rows))
	for _, r := range rows {
		texts[r.ID] = []string{r.Username}
		if r.Search != nil {
			texts[r.ID] = append(texts[r.ID], *r.Search)
		}
	}
	x.mu.Lock()
	x.texts, x.loaded = texts, true
	x.mu.Unlock()
	return nil
}

// add は id のユーザーに一致しうる文字列を足す。消さない（上の説明）
func (x *userSearchIndex) add(id string, texts ...string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	for _, t := range texts {
		if !slices.Contains(x.texts[id], t) {
			x.texts[id] = append(x.texts[id], t)
		}
	}
}

// candidates は query（normalizeSearchText 済み）を部分文字列に含みうるユーザーの id を返す。
// LIKE '%query%'（likeEscaper で % と _ をエスケープ済み）と同じ一致。ok が false なら未構築なので、絞らずに検索する
func (x *userSearchIndex) candidates(query string) (ids []string, ok bool) {
	x.mu.RLock()
	defer x.mu.RUnlock()
	if !x.loaded {
		return nil, false
	}
	for id, texts := range x.texts {
		for _, t := range texts {
			if strings.Contains(t, query) {
				ids = append(ids, id)
				break
			}
		}
	}
	return ids, true
}
