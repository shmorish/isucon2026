package isureal

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"sync"
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
// ponytail: 上限を超えたら全消去。LRU が必要なら差し替える
const imageSignatureCacheMax = 200000

var (
	imageSignatureMu    sync.Mutex
	imageSignatureCache = map[string][]byte{}
)

func imageSignature(key []byte, imageID string, exp int64) []byte {
	cacheKey := string(key) + strconv.FormatInt(exp, 10) + ":" + imageID
	imageSignatureMu.Lock()
	sig, ok := imageSignatureCache[cacheKey]
	imageSignatureMu.Unlock()
	if ok {
		return sig
	}
	sig = computeImageSignature(key, imageID, exp)
	imageSignatureMu.Lock()
	if len(imageSignatureCache) >= imageSignatureCacheMax {
		clear(imageSignatureCache)
	}
	imageSignatureCache[cacheKey] = sig
	imageSignatureMu.Unlock()
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
