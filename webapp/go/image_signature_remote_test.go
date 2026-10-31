package isureal

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
)

// fakeSigner は s2・s3 の署名 API。handler が nil なら本物の signImagesHandler で答える
func fakeSigner(t *testing.T, handler echo.HandlerFunc) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var calls atomic.Int64
	if handler == nil {
		handler = signImagesHandler
	}
	e := echo.New()
	e.POST("/api/internal/image-signatures", func(c *echo.Context) error {
		calls.Add(1)
		return handler(c)
	})
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return srv, &calls
}

func useSigners(t *testing.T, urls ...string) {
	t.Helper()
	imageSigners = urls
	imageSignerDownUntil = sync.Map{}
	inflightRequests.Store(0)
	t.Cleanup(func() {
		imageSigners = nil
		imageSignerDownUntil = sync.Map{}
		inflightRequests.Store(0)
	})
}

func testImagesToSign(key []byte, n int) []imageToSign {
	profile := "profile-" + strconv.Itoa(len(key))
	user := User{ImageKey: key, ProfileImageID: &profile}
	var toSign []imageToSign
	for i := range n {
		toSign = appendImagesToSign(toSign, user, "img-"+strconv.Itoa(i))
	}
	return toSign
}

func assertSignedFromCache(t *testing.T, key []byte, exp int64, ids ...string) {
	t.Helper()
	for _, id := range ids {
		sig, ok := cachedImageSignature(imageSignatureCacheKey(key, id, exp))
		if !ok || !bytes.Equal(sig, computeImageSignature(key, id, exp)) {
			t.Fatalf("signature for %s exp %d missing or wrong", id, exp)
		}
	}
}

func TestPrefetchImageSignaturesFromRemote(t *testing.T) {
	srv, calls := fakeSigner(t, nil)
	useSigners(t, srv.URL, srv.URL)

	key := bytes.Repeat([]byte{7}, 32)
	before := time.Now().Add(imageURLLifetime).Unix()
	prefetchImageSignatures(context.Background(), testImagesToSign(key, 16))
	after := time.Now().Add(imageURLLifetime).Unix()
	if calls.Load() != 2 {
		t.Fatalf("expected one request per signer, got %d", calls.Load())
	}
	// 取り寄せは exp から imageSignatureExpWindow 個なので、取り寄せの前後どちらの秒でも揃っている
	for exp := before; exp <= after; exp++ {
		assertSignedFromCache(t, key, exp, "img-0", "img-15", "profile-32")
	}
	// 署名の形は手元の計算と同じ（URL は変わらない）
	for exp := before; exp < before+imageSignatureExpWindow; exp++ {
		if !bytes.Equal(imageSignature(key, "img-3", exp), computeImageSignature(key, "img-3", exp)) {
			t.Fatal("remote signature differs from local")
		}
	}
}

func TestPrefetchSmallBatchOnlyWhenBusy(t *testing.T) {
	srv, calls := fakeSigner(t, nil)
	useSigners(t, srv.URL)

	key := bytes.Repeat([]byte{8}, 32)
	prefetchImageSignatures(context.Background(), testImagesToSign(key, 2))
	if calls.Load() != 0 {
		t.Fatal("a small batch must stay local while s1 is idle")
	}
	inflightRequests.Store(int64(imageSignBusyInflight))
	prefetchImageSignatures(context.Background(), testImagesToSign(key, 2))
	if calls.Load() != 1 {
		t.Fatalf("a small batch must go remote while s1 is busy, got %d calls", calls.Load())
	}
	assertSignedFromCache(t, key, time.Now().Add(imageURLLifetime).Unix(), "img-0", "img-1", "profile-32")
	// 2回目は全部メモにあるので頼まない
	prefetchImageSignatures(context.Background(), testImagesToSign(key, 2))
	if calls.Load() != 1 {
		t.Fatal("cached signatures must not be requested again")
	}
}

func TestPrefetchFallsBackWhenSignerFails(t *testing.T) {
	defer func(d time.Duration) { imageSignerTimeout = d }(imageSignerTimeout)
	imageSignerTimeout = 300 * time.Millisecond
	failing, failCalls := fakeSigner(t, func(c *echo.Context) error { return c.NoContent(http.StatusInternalServerError) })
	refused := httptest.NewServer(http.NotFoundHandler())
	refused.Close()
	hanging, hangCalls := fakeSigner(t, func(c *echo.Context) error {
		time.Sleep(imageSignerTimeout + 200*time.Millisecond)
		return c.NoContent(http.StatusOK)
	})
	useSigners(t, failing.URL, refused.URL, hanging.URL)

	key := bytes.Repeat([]byte{9}, 32)
	start := time.Now()
	prefetchImageSignatures(context.Background(), testImagesToSign(key, 16))
	if elapsed := time.Since(start); elapsed > imageSignerTimeout+300*time.Millisecond {
		t.Fatalf("prefetch waited %v, longer than the timeout", elapsed)
	}
	// 失敗しても URL は手元の計算で同じになる
	exp := time.Now().Add(imageURLLifetime).Unix()
	if !bytes.Equal(imageSignature(key, "img-0", exp), computeImageSignature(key, "img-0", exp)) {
		t.Fatal("local fallback produced a different signature")
	}
	// 200 以外・つながらない署名先はしばらく使わないが、時間切れの署名先は休ませない（s1 が混んでいるだけ）
	if got := availableImageSigners(); len(got) != 1 || got[0] != hanging.URL {
		t.Fatalf("only the timed-out signer must stay available, got %v", got)
	}
	prefetchImageSignatures(context.Background(), testImagesToSign(key, 16))
	if failCalls.Load() != 1 {
		t.Fatalf("a signer in cooldown was called again (%d calls)", failCalls.Load())
	}
	if hangCalls.Load() != 2 {
		t.Fatalf("a timed-out signer must be tried again (%d calls)", hangCalls.Load())
	}
	imageSignerDownUntil.Store(failing.URL, time.Now().Add(-time.Second).UnixNano())
	if got := availableImageSigners(); len(got) != 2 || got[0] != failing.URL {
		t.Fatalf("signer must come back after the cooldown, got %v", got)
	}
}

func TestRemoteSignaturesMatchLocal(t *testing.T) {
	srv, _ := fakeSigner(t, nil)
	keys := [][]byte{bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{0xab}, 32)}
	var reqs []imageSignatureRequest
	for _, k := range keys {
		for _, exp := range []int64{100, 1761900000, 1761900001} {
			reqs = append(reqs, imageSignatureRequest{k, "0192f1c2-1111-7000-8000-000000000000", exp})
		}
	}
	sigs, err := requestImageSignatures(context.Background(), srv.URL, reqs)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range reqs {
		if !bytes.Equal(sigs[i], computeImageSignature(r.Key, r.ImageID, r.Exp)) {
			t.Fatalf("remote signature %d differs from local", i)
		}
	}
}

func TestStartPrefetchImageSignaturesWaits(t *testing.T) {
	srv, calls := fakeSigner(t, nil)
	useSigners(t, srv.URL)
	inflightRequests.Store(int64(imageSignBusyInflight))

	key := bytes.Repeat([]byte{10}, 32)
	wait := startPrefetchImageSignatures(context.Background(), testImagesToSign(key, 1))
	wait()
	wait() // 2回呼んでも戻る
	if calls.Load() != 1 {
		t.Fatalf("expected one remote call, got %d", calls.Load())
	}
	assertSignedFromCache(t, key, time.Now().Add(imageURLLifetime).Unix(), "img-0", "profile-32")
	if startPrefetchImageSignatures(context.Background(), nil) == nil {
		t.Fatal("nothing to sign must still return a wait function")
	}
}

func TestImageSignatureCacheKeepsPreviousGeneration(t *testing.T) {
	imageSignatureMu.Lock()
	imageSignatureCache, imageSignatureCacheOld = map[string][]byte{}, map[string][]byte{}
	imageSignatureMu.Unlock()
	storeImageSignature("first", []byte("1"))
	for i := range imageSignatureCacheMax {
		storeImageSignature("k"+strconv.Itoa(i), []byte("x"))
	}
	if _, ok := cachedImageSignature("first"); !ok {
		t.Fatal("an entry from the previous generation must still be found")
	}
	for i := range imageSignatureCacheMax {
		storeImageSignature("n"+strconv.Itoa(i), []byte("y"))
	}
	if _, ok := cachedImageSignature("first"); ok {
		t.Fatal("entries two generations back must be gone")
	}
	imageSignatureMu.Lock()
	imageSignatureCache, imageSignatureCacheOld = map[string][]byte{}, map[string][]byte{}
	imageSignatureMu.Unlock()
}

func TestInflightMiddlewareSkipsLongPolling(t *testing.T) {
	inflightRequests.Store(0)
	e := echo.New()
	e.Use(InflightMiddleware())
	var seen []int64
	e.GET("/api/notifications", func(c *echo.Context) error { seen = append(seen, inflightRequests.Load()); return c.NoContent(200) })
	e.GET("/api/me", func(c *echo.Context) error { seen = append(seen, inflightRequests.Load()); return c.NoContent(200) })
	for _, path := range []string{"/api/notifications", "/api/me"} {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	}
	if seen[0] != 0 || seen[1] != 1 || inflightRequests.Load() != 0 {
		t.Fatalf("inflight counts %v, after %d", seen, inflightRequests.Load())
	}
}
