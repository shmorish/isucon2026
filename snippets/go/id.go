package isuutil

import "sync/atomic"

// IDGen は DB の AUTO_INCREMENT を使わずにアプリ側で ID を採番する (ISUCON12 上位の定番)。
// 複数台で衝突しないよう id = n*nodes + node の形で配る。値が小さいので JS の 2^53 制限にも当たらない。
// 起動時に SELECT MAX(id) を渡せば再起動後も重複しない。
// ponytail: 対象テーブルへの INSERT が全て IDGen 経由であることが前提。混在するなら Snowflake 等に替える。
type IDGen struct {
	n     atomic.Int64
	node  int64
	nodes int64
}

// NewIDGen: maxID は現在の MAX(id)、node は 0..nodes-1 のサーバー番号。
func NewIDGen(maxID, node, nodes int64) *IDGen {
	g := &IDGen{node: node, nodes: nodes}
	g.n.Store(maxID / nodes)
	return g
}

func (g *IDGen) Next() int64 {
	return g.n.Add(1)*g.nodes + g.node
}
