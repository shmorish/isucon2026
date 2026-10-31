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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v5"
)

// s1 の CPU には SHA-NI がなく、署名（HMAC 1000回）が CPU の3割を使っていた（#48, #77）。
// SHA-NI があり CPU の空いている s2・s3 に、レスポンスで使う署名をまとめて計算させてメモに入れておく。
// exp の決め方は変えない: 取り寄せてから URL を組み立てるまでに秒が進んでもよいように、exp から
// imageSignatureExpWindow 個の exp を取る。失敗したら何もしない（今まで通り手元で計算する）。
// 例: ISUREAL_IMAGE_SIGNERS=http://192.168.2.10,http://192.168.3.10
var imageSigners = strings.FieldsFunc(os.Getenv("ISUREAL_IMAGE_SIGNERS"), func(r rune) bool { return r == ',' })

// 取り寄せから組み立てまでは往復 100ms に s1 のスケジューリング待ちが乗る。exp と exp+1 だけだと
// 混んでいるときに取り寄せた署名が使われずに手元で計算し直していた（#77）
const imageSignatureExpWindow = 3

// 空いているときは、往復 100ms より手元の計算（1枚 0.45ms）が速い小さなレスポンスは任せない。
// 混んでいるとき（処理中のリクエストが imageSignBusyInflight 以上）は CPU のほうが貴重なので 1枚から任せる
const (
	imageSignaturePrefetchMin     = 16
	imageSignaturePrefetchMinBusy = 1
)

var imageSignBusyInflight = envInt("ISUREAL_IMAGE_SIGN_BUSY_INFLIGHT", 16)

// 署名の取り寄せ 1 回に待つ上限。往復は 100ms（新しい接続で 200ms）、s2 側の計算は 300 枚でも 25ms。
// これを超えたら手元で計算する。s2・s3 が落ちているときに毎回待たないよう、失敗した署名先はしばらく使わない
const (
	imageSignerTimeout  = 500 * time.Millisecond
	imageSignerCooldown = 2 * time.Second
)

var imageSignerClient = &http.Client{
	Timeout:   time.Second,
	Transport: &http.Transport{MaxIdleConnsPerHost: 64},
}

func envInt(key string, defaultValue int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return defaultValue
}

// inflightRequests は処理中の /api リクエスト数（通知のロングポーリングと署名の内部 API は除く）。
// s1 の CPU が足りないときはこれが数百になる（14:26 のベンチでは約 1,000 goroutine がプール待ち）
var inflightRequests atomic.Int64

func InflightMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			switch c.Request().URL.Path {
			case "/api/notifications", "/api/internal/image-signatures":
				return next(c)
			}
			inflightRequests.Add(1)
			defer inflightRequests.Add(-1)
			return next(c)
		}
	}
}

func imageSigningBusy() bool {
	return inflightRequests.Load() >= int64(imageSignBusyInflight)
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

// imageID の画像と user のプロフィール画像を、user の鍵で署名する対象に加える。
// 対象は newPostResponse・newRealMojiResponse・newCommentResponse・newGetMeResponse が署名する画像と揃える
// （ずれても手元で計算するだけで URL は同じ）
func appendImagesToSign(dst []imageToSign, user User, imageIDs ...string) []imageToSign {
	for _, id := range imageIDs {
		dst = append(dst, imageToSign{user.ImageKey, id})
	}
	if user.ProfileImageID != nil {
		dst = append(dst, imageToSign{user.ImageKey, *user.ProfileImageID})
	}
	return dst
}

func imagesToSignForPosts(posts []PostWithAuthor) []imageToSign {
	toSign := make([]imageToSign, 0, 2*len(posts))
	for _, post := range posts {
		toSign = appendImagesToSign(toSign, post.Author, post.ImageID)
	}
	return toSign
}

func imagesToSignForUsers(users []User) []imageToSign {
	toSign := make([]imageToSign, 0, len(users))
	for _, user := range users {
		toSign = appendImagesToSign(toSign, user)
	}
	return toSign
}

// missingImageSignatureRequests は images のうちメモにない (image, exp) を、exp から imageSignatureExpWindow 個の
// exp について返す。同じ画像が何度も出てきても（フィードの同じ作者など）1回しか頼まない。
// images は足りない exp がひとつでもある画像の数
func missingImageSignatureRequests(images []imageToSign, exp int64) (reqs []imageSignatureRequest, missing int) {
	seen := make(map[string]bool, imageSignatureExpWindow*len(images))
	for _, img := range images {
		missingExps := 0
		for e := exp; e < exp+imageSignatureExpWindow; e++ {
			k := imageSignatureCacheKey(img.key, img.imageID, e)
			if seen[k] {
				continue
			}
			seen[k] = true
			if _, ok := cachedImageSignature(k); !ok {
				reqs = append(reqs, imageSignatureRequest{img.key, img.imageID, e})
				missingExps++
			}
		}
		if missingExps > 0 {
			missing++
		}
	}
	return reqs, missing
}

// prefetchImageSignatures は images の署名を s2・s3 から取り寄せてメモに入れる。戻ったあとに signedImageURL を
// 呼べばメモから取れる。取り寄せない・失敗したときは何もせず、signedImageURL が手元で計算する
func prefetchImageSignatures(ctx context.Context, images []imageToSign) {
	if len(imageSigners) == 0 || len(images) == 0 {
		return
	}
	reqs, missing := missingImageSignatureRequests(images, time.Now().Add(imageURLLifetime).Unix())
	minimum := imageSignaturePrefetchMin
	if imageSigningBusy() {
		minimum = imageSignaturePrefetchMinBusy
	}
	if missing < minimum {
		return
	}
	signers := availableImageSigners()
	if len(signers) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, imageSignerTimeout)
	defer cancel()
	var wg sync.WaitGroup
	for i, signer := range signers {
		chunk := reqs[i*len(reqs)/len(signers) : (i+1)*len(reqs)/len(signers)]
		if len(chunk) == 0 {
			continue
		}
		wg.Go(func() {
			sigs, err := requestImageSignatures(ctx, signer, chunk)
			if err != nil {
				markImageSignerDown(signer)
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

// startPrefetchImageSignatures は取り寄せを裏で始め、待つ関数を返す。DB の INSERT・COMMIT と並行して
// 取り寄せる投稿作成・RealMoji 用。返した関数は何度呼んでもよい
func startPrefetchImageSignatures(ctx context.Context, images []imageToSign) func() {
	if len(imageSigners) == 0 || len(images) == 0 {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		prefetchImageSignatures(ctx, images)
	}()
	return func() { <-done }
}

// 失敗した署名先は imageSignerCooldown のあいだ使わない（キーは署名先の URL、値は再開してよい時刻の UnixNano）
var imageSignerDownUntil sync.Map

func markImageSignerDown(signer string) {
	imageSignerDownUntil.Store(signer, time.Now().Add(imageSignerCooldown).UnixNano())
}

func availableImageSigners() []string {
	now := time.Now().UnixNano()
	signers := make([]string, 0, len(imageSigners))
	for _, signer := range imageSigners {
		if until, ok := imageSignerDownUntil.Load(signer); ok && until.(int64) > now {
			continue
		}
		signers = append(signers, signer)
	}
	return signers
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
	for _, sig := range sigs {
		if len(sig) != sha256.Size {
			return nil, fmt.Errorf("got a %d-byte signature", len(sig))
		}
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
