package isureal

import "testing"

func TestBoundedCacheKeepsFirstValue(t *testing.T) {
	c := newBoundedCache(10, func(v []byte) int { return len(v) })
	c.put("a", []byte("x"))
	c.put("a", []byte("y"))
	if v, _ := c.get("a"); string(v) != "x" {
		t.Fatalf("got %q, want first value", v)
	}
}

func TestBoundedCacheDropsAllWhenOverMax(t *testing.T) {
	c := newBoundedCache(3, func(int) int { return 1 })
	for i, k := range []string{"a", "b", "c"} {
		c.put(k, i)
	}
	for _, k := range []string{"a", "b", "c"} {
		if _, ok := c.get(k); !ok {
			t.Fatalf("%s evicted before reaching max", k)
		}
	}
	c.put("d", 3)
	for _, k := range []string{"a", "b", "c"} {
		if _, ok := c.get(k); ok {
			t.Fatalf("%s survived overflow", k)
		}
	}
	if v, ok := c.get("d"); !ok || v != 3 {
		t.Fatalf("d = %v, %v after overflow", v, ok)
	}
}

func TestBoundedCacheCountsBytes(t *testing.T) {
	c := newBoundedCache(4, func(v []byte) int { return len(v) })
	c.put("a", []byte("abc"))
	c.put("b", []byte("d"))
	if _, ok := c.get("a"); !ok {
		t.Fatal("a evicted at exactly max bytes")
	}
	c.put("c", []byte("e"))
	if _, ok := c.get("a"); ok {
		t.Fatal("a survived exceeding max bytes")
	}
}
