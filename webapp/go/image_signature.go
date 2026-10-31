package isureal

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const imageURLLifetime = 24 * time.Hour
const imageSignatureIterations = 1000

func imageSigningKey(secretKeyHash string) ([]byte, error) {
	key, err := hex.DecodeString(secretKeyHash)
	if err != nil || len(key) != sha256.Size {
		return nil, fmt.Errorf("invalid image signing key")
	}
	return key, nil
}

// 署名は (key, imageID, exp) で決まるのでメモ化する。再起動で消えても計算し直すだけ。
// 2世代で持つ: 今の世代が上限に達したら前の世代を捨てて入れ替える。全消去だと、s2・s3 から取り寄せたばかりの署名
// （#48, #77）や直後の GET /api/images が検証に使う署名まで消えて、手元で計算し直していた。
// 1件およそ 150 バイトなので 2世代で 60MB 程度
// ponytail: LRU が必要なら差し替える
const imageSignatureCacheMax = 200000

var (
	imageSignatureMu       sync.Mutex
	imageSignatureCache    = map[string][]byte{}
	imageSignatureCacheOld = map[string][]byte{}
)

// s1 では HMAC を計算せず、メモにない署名は s2・s3 にその場で頼む（ISUREAL_IMAGE_SIGN_REMOTE_ONLY=1）。
// 全署名先に断られたときだけ手元で計算する。ISUREAL_IMAGE_SIGN_LOCAL_FALLBACK=0 ならそれもせず 500 にする
var (
	imageSignRemoteOnly    = os.Getenv("ISUREAL_IMAGE_SIGN_REMOTE_ONLY") == "1"
	imageSignLocalFallback = os.Getenv("ISUREAL_IMAGE_SIGN_LOCAL_FALLBACK") != "0"
	// imageSignLocalFallbacks は remote-only なのに手元で計算した回数、imageSignLocalComputes は imageSignature が手元で計算した回数
	imageSignLocalFallbacks    atomic.Int64
	imageSignLocalComputes     atomic.Int64
	errImageSigningUnavailable = errors.New("image signing unavailable: all signers failed")
)

func imageSignatureCacheKey(key []byte, imageID string, exp int64) string {
	return string(key) + strconv.FormatInt(exp, 10) + ":" + imageID
}

func cachedImageSignature(cacheKey string) ([]byte, bool) {
	imageSignatureMu.Lock()
	defer imageSignatureMu.Unlock()
	sig, ok := imageSignatureCache[cacheKey]
	if !ok {
		sig, ok = imageSignatureCacheOld[cacheKey]
	}
	return sig, ok
}

func storeImageSignature(cacheKey string, sig []byte) {
	imageSignatureMu.Lock()
	defer imageSignatureMu.Unlock()
	if len(imageSignatureCache) >= imageSignatureCacheMax {
		imageSignatureCacheOld = imageSignatureCache
		imageSignatureCache = make(map[string][]byte, imageSignatureCacheMax)
	}
	imageSignatureCache[cacheKey] = sig
}

func imageSignature(key []byte, imageID string, exp int64) []byte {
	cacheKey := imageSignatureCacheKey(key, imageID, exp)
	if sig, ok := cachedImageSignature(cacheKey); ok {
		return sig
	}
	if imageSignRemoteOnly && len(imageSigners) > 0 {
		sigs := signImagesRemotely(context.Background(), []imageSignatureRequest{{key, imageID, exp}})
		if sigs[0] != nil {
			storeImageSignature(cacheKey, sigs[0])
			return sigs[0]
		}
		if !imageSignLocalFallback {
			// Recover ミドルウェアが拾って、ほかの内部エラーと同じ 500 になる
			panic(errImageSigningUnavailable)
		}
		slog.Warn("image signing fell back to local", "count", imageSignLocalFallbacks.Add(1))
	}
	imageSignLocalComputes.Add(1)
	sig := computeImageSignature(key, imageID, exp)
	storeImageSignature(cacheKey, sig)
	return sig
}

func computeImageSignature(key []byte, imageID string, exp int64) []byte {
	value := []byte(strconv.FormatInt(exp, 10) + ":" + imageID)
	// Reset reuses the precomputed ipad/opad states: half the SHA-256 blocks per round and no per-round allocs (#41)
	mac := hmac.New(sha256.New, key)
	var buf [sha256.Size]byte
	for range imageSignatureIterations {
		mac.Reset()
		mac.Write(value)
		value = mac.Sum(buf[:0])
	}
	return append([]byte(nil), value...)
}

func signedImageURL(user User, imageID string) string {
	exp := time.Now().Add(imageURLLifetime).Unix()
	query := url.Values{
		"exp": {strconv.FormatInt(exp, 10)},
		"sig": {base64.RawURLEncoding.EncodeToString(imageSignature(user.ImageKey, imageID, exp))},
	}
	return "/api/images/" + url.PathEscape(imageID) + "?" + query.Encode()
}
