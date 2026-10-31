package isureal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

func TestImageDiskCacheRoundTripAndRejectsNonUUID(t *testing.T) {
	imageDiskCacheDir = t.TempDir()
	id := "0123abcd-0000-4000-8000-000000000001"
	writeImageFile(id, []byte("jpeg"))
	imageDataCache = newBoundedCache(imageDataCacheMaxBytes, func(v []byte) int { return len(v) })
	if data, ok := getImageData(id); !ok || !bytes.Equal(data, []byte("jpeg")) {
		t.Fatalf("got %q, %v from disk", data, ok)
	}
	for _, bad := range []string{"../../etc/passwd", "0123ABCD-0000-4000-8000-000000000001", "x"} {
		writeImageFile(bad, []byte("x"))
		if _, ok := getImageData(bad); ok {
			t.Fatalf("%q was cached", bad)
		}
	}
	entries, _ := filepath.Glob(filepath.Join(imageDiskCacheDir, "*", "*"))
	if len(entries) != 1 {
		t.Fatalf("files on disk: %v", entries)
	}
	// putImageData writes in the background
	id2 := "0123abcd-0000-4000-8000-000000000002"
	putImageData(id2, []byte("jpeg2"))
	for i := 0; i < 100; i++ {
		if b, err := os.ReadFile(filepath.Join(imageDiskCacheDir, "01", id2)); err == nil && string(b) == "jpeg2" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("putImageData did not write the file")
}
