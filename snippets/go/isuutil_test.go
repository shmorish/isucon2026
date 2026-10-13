package isuutil

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheLoadsOnceUnderConcurrency(t *testing.T) {
	c := NewCache[int, string](time.Minute)
	var calls atomic.Int32
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := c.GetOrLoad(1, func() (string, error) {
				calls.Add(1)
				time.Sleep(10 * time.Millisecond)
				return "x", nil
			})
			if err != nil || v != "x" {
				t.Errorf("got %q %v", v, err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("load called %d times", calls.Load())
	}
}

func TestCacheExpires(t *testing.T) {
	c := NewCache[string, int](10 * time.Millisecond)
	c.Set("a", 1)
	if _, ok := c.Get("a"); !ok {
		t.Fatal("miss before ttl")
	}
	time.Sleep(20 * time.Millisecond)
	if _, ok := c.Get("a"); ok {
		t.Fatal("hit after ttl")
	}
}

func TestIDGenUniqueAcrossNodes(t *testing.T) {
	const nodes = 3
	seen := sync.Map{}
	var wg sync.WaitGroup
	for node := range int64(nodes) {
		g := NewIDGen(1000, node, nodes)
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 1000 {
					id := g.Next()
					if id <= 1000 {
						t.Errorf("id %d not above maxID", id)
					}
					if _, dup := seen.LoadOrStore(id, true); dup {
						t.Errorf("duplicate id %d", id)
					}
				}
			}()
		}
	}
	wg.Wait()
}
