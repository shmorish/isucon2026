package isureal

import (
	"bytes"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestSignedImageURLExpNotShorterThanLifetime(t *testing.T) {
	min := time.Now().Add(imageURLLifetime).Unix()
	u, err := url.Parse(signedImageURL(User{ImageKey: make([]byte, 32)}, "img"))
	if err != nil {
		t.Fatal(err)
	}
	exp, err := strconv.ParseInt(u.Query().Get("exp"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	if exp < min || exp > min+int64(imageExpStep/time.Second) || exp%int64(imageExpStep/time.Second) != 0 {
		t.Fatalf("exp=%d min=%d", exp, min)
	}
	if !strings.HasPrefix(u.Path, "/api/images/img") {
		t.Fatalf("path=%s", u.Path)
	}
}
