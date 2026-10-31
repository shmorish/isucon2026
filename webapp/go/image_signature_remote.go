package isureal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
)

// s1 の CPU には SHA-NI がなく、署名（HMAC 1000回）が CPU の6割を使っていた（#48）。
// SHA-NI があり CPU の空いている s2・s3 に、レスポンスで使う署名をまとめて計算させてメモに入れておく。
// exp の決め方は変えない: 組み立て中に秒が変わってもよいように exp と exp+1 の両方を取る。
// 失敗したら何もしない（今まで通り手元で計算する）。
// 例: ISUREAL_IMAGE_SIGNERS=http://192.168.2.10,http://192.168.3.10
var imageSigners = strings.FieldsFunc(os.Getenv("ISUREAL_IMAGE_SIGNERS"), func(r rune) bool { return r == ',' })

// 往復 100ms より手元の計算（1枚 0.45ms）が速い小さなレスポンスは任せない
const imageSignaturePrefetchMin = 16

var imageSignerClient = &http.Client{
	Timeout:   time.Second,
	Transport: &http.Transport{MaxIdleConnsPerHost: 64},
}

type imageSignatureRequest struct {
	Key     []byte `json:"key"`
	ImageID string `json:"imageId"`
	Exp     int64  `json:"exp"`
}

type imageToSign struct {
	key     []byte
	imageID string
}

// imageID の画像と user のプロフィール画像を、user の鍵で署名する対象に加える
func appendImagesToSign(dst []imageToSign, user User, imageIDs ...string) []imageToSign {
	for _, id := range imageIDs {
		dst = append(dst, imageToSign{user.ImageKey, id})
	}
	if user.ProfileImageID != nil {
		dst = append(dst, imageToSign{user.ImageKey, *user.ProfileImageID})
	}
	return dst
}

func prefetchImageSignatures(ctx context.Context, images []imageToSign) {
	if len(imageSigners) == 0 {
		return
	}
	exp := time.Now().Add(imageURLLifetime).Unix()
	var reqs []imageSignatureRequest
	seen := map[string]bool{}
	for _, img := range images {
		for _, e := range []int64{exp, exp + 1} {
			k := imageSignatureCacheKey(img.key, img.imageID, e)
			if seen[k] {
				continue
			}
			seen[k] = true
			if _, ok := cachedImageSignature(k); !ok {
				reqs = append(reqs, imageSignatureRequest{img.key, img.imageID, e})
			}
		}
	}
	if len(reqs) < imageSignaturePrefetchMin {
		return
	}
	var wg sync.WaitGroup
	for i, signer := range imageSigners {
		chunk := reqs[i*len(reqs)/len(imageSigners) : (i+1)*len(reqs)/len(imageSigners)]
		wg.Go(func() {
			sigs, err := requestImageSignatures(ctx, signer, chunk)
			if err != nil {
				slog.Warn("remote image signing failed", "signer", signer, "error", err)
				return
			}
			for j, r := range chunk {
				storeImageSignature(imageSignatureCacheKey(r.Key, r.ImageID, r.Exp), sigs[j])
			}
		})
	}
	wg.Wait()
}

func requestImageSignatures(ctx context.Context, signer string, reqs []imageSignatureRequest) ([][]byte, error) {
	body, err := json.Marshal(reqs)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, signer+"/api/internal/image-signatures", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := imageSignerClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", res.StatusCode)
	}
	var sigs [][]byte
	if err := json.NewDecoder(res.Body).Decode(&sigs); err != nil {
		return nil, err
	}
	if len(sigs) != len(reqs) {
		return nil, fmt.Errorf("got %d signatures for %d requests", len(sigs), len(reqs))
	}
	return sigs, nil
}

func signImagesHandler(c *echo.Context) error {
	var reqs []imageSignatureRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&reqs); err != nil || len(reqs) > 10000 {
		return NewAPIError(http.StatusBadRequest, "invalid_request")
	}
	for _, r := range reqs {
		if len(r.Key) != sha256.Size {
			return NewAPIError(http.StatusBadRequest, "invalid_request")
		}
	}
	sigs := make([][]byte, len(reqs))
	workers := runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			for i := w; i < len(reqs); i += workers {
				sigs[i] = computeImageSignature(reqs[i].Key, reqs[i].ImageID, reqs[i].Exp)
			}
		})
	}
	wg.Wait()
	return c.JSON(http.StatusOK, sigs)
}
