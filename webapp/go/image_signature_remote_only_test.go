package isureal

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
)

func TestImageSignatureRemoteOnly(t *testing.T) {
	imageSignRemoteOnly = true
	t.Cleanup(func() { imageSignRemoteOnly = false })
	key := bytes.Repeat([]byte{7}, 32)
	exp := time.Now().Add(imageURLLifetime).Unix() + 1000 // 先取りの範囲外: メモにない
	broken, brokenCalls := fakeSigner(t, func(c *echo.Context) error { return c.NoContent(http.StatusInternalServerError) })
	good, goodCalls := fakeSigner(t, nil)

	// (a)(b) 1つ目が失敗しても 2つ目で取れ、手元では計算しない
	useSigners(t, good.URL, broken.URL) // 1件だけの依頼は最後の署名先に行く
	local, fallbacks := imageSignLocalComputes.Load(), imageSignLocalFallbacks.Load()
	if got := imageSignature(key, "remote-only-a", exp); !bytes.Equal(got, computeImageSignature(key, "remote-only-a", exp)) {
		t.Fatal("remote signature differs from local")
	}
	if imageSignLocalComputes.Load() != local || brokenCalls.Load() != 1 || goodCalls.Load() != 1 {
		t.Fatalf("local=%d broken=%d good=%d", imageSignLocalComputes.Load()-local, brokenCalls.Load(), goodCalls.Load())
	}
	// 先取りも 1枚から頼む（broken は休ませ中なので good だけ）
	prefetchImageSignatures(t.Context(), []imageToSign{{key, "remote-only-p"}})
	if goodCalls.Load() != 2 || imageSignLocalComputes.Load() != local {
		t.Fatalf("prefetch: good=%d local=%d", goodCalls.Load(), imageSignLocalComputes.Load()-local)
	}

	// (c) 全部失敗したら手元で計算して数える
	useSigners(t, broken.URL)
	if got := imageSignature(key, "remote-only-c", exp); !bytes.Equal(got, computeImageSignature(key, "remote-only-c", exp)) {
		t.Fatal("fallback signature differs")
	}
	if imageSignLocalFallbacks.Load() != fallbacks+1 || imageSignLocalComputes.Load() != local+1 {
		t.Fatalf("fallbacks=%d", imageSignLocalFallbacks.Load()-fallbacks)
	}
}
