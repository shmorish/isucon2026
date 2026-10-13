package isuutil

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type entry[V any] struct {
	v   V
	exp time.Time
}

// Cache は TTL 付きオンメモリキャッシュ。同じキーの同時ミスは singleflight で1回のロードにまとめる。
// ttl=0 なら期限なし (マスタデータ向け)。更新時は Delete か Set で反映する。
type Cache[K comparable, V any] struct {
	m   sync.Map
	g   singleflight.Group
	ttl time.Duration
}

func NewCache[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	return &Cache[K, V]{ttl: ttl}
}

func (c *Cache[K, V]) Set(k K, v V) {
	e := entry[V]{v: v}
	if c.ttl > 0 {
		e.exp = time.Now().Add(c.ttl)
	}
	c.m.Store(k, e)
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	if x, ok := c.m.Load(k); ok {
		e := x.(entry[V])
		if e.exp.IsZero() || time.Now().Before(e.exp) {
			return e.v, true
		}
	}
	var zero V
	return zero, false
}

func (c *Cache[K, V]) Delete(k K) { c.m.Delete(k) }

// GetOrLoad はキャッシュになければ load を呼んで保存する。load がエラーならキャッシュしない。
func (c *Cache[K, V]) GetOrLoad(k K, load func() (V, error)) (V, error) {
	if v, ok := c.Get(k); ok {
		return v, nil
	}
	x, err, _ := c.g.Do(fmt.Sprint(k), func() (any, error) {
		v, err := load()
		if err == nil {
			c.Set(k, v)
		}
		return v, err
	})
	return x.(V), err
}
