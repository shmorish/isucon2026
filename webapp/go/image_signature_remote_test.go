package isureal

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
)

func TestPrefetchImageSignaturesFromRemote(t *testing.T) {
	e := echo.New()
	e.POST("/api/internal/image-signatures", signImagesHandler)
	srv := httptest.NewServer(e)
	defer srv.Close()
	imageSigners = []string{srv.URL, srv.URL}
	defer func() { imageSigners = nil }()

	key := bytes.Repeat([]byte{7}, 32)
	profile := "profile"
	user := User{ImageKey: key, ProfileImageID: &profile}
	var toSign []imageToSign
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		toSign = appendImagesToSign(toSign, user, id)
	}
	prefetchImageSignatures(context.Background(), toSign)
	exp := time.Now().Add(imageURLLifetime).Unix() // 取り寄せは exp と exp+1 の両方なので、後で測れば必ず含まれる
	for _, id := range []string{"a", "h", "profile"} {
		sig, ok := cachedImageSignature(imageSignatureCacheKey(key, id, exp))
		if !ok || !bytes.Equal(sig, computeImageSignature(key, id, exp)) {
			t.Fatalf("remote signature for %s missing or wrong", id)
		}
	}
}
