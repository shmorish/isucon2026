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
const imageExpStep = 10 * time.Minute

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
	for range imageSignatureIterations {
		mac := hmac.New(sha256.New, key)
		mac.Write(value)
		value = mac.Sum(nil)
	}
	return value
}

func signedImageURL(user User, imageID string) string {
	// exp を10分単位で切り上げて、同じ画像の署名をメモ化から返せるようにする（#29）。
	// 期限は24時間より短くならず、最大10分延びるだけ
	exp := time.Now().Add(imageURLLifetime + imageExpStep - time.Second).Truncate(imageExpStep).Unix()
	query := url.Values{
		"exp": {strconv.FormatInt(exp, 10)},
		"sig": {base64.RawURLEncoding.EncodeToString(imageSignature(user.ImageKey, imageID, exp))},
	}
	return "/api/images/" + url.PathEscape(imageID) + "?" + query.Encode()
}
