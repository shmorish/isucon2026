package isureal

import (
	"bytes"
	"testing"
)

func TestImageSignatureCacheMatchesCompute(t *testing.T) {
	k1, k2 := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	for range 2 { // 2回目はキャッシュから
		if !bytes.Equal(imageSignature(k1, "img", 100), computeImageSignature(k1, "img", 100)) {
			t.Fatal("cached signature differs")
		}
	}
	if bytes.Equal(imageSignature(k1, "img", 100), imageSignature(k2, "img", 100)) {
		t.Fatal("different keys must not share a cache entry")
	}
	if bytes.Equal(imageSignature(k1, "img", 100), imageSignature(k1, "img", 101)) {
		t.Fatal("different exp must not share a cache entry")
	}
}
