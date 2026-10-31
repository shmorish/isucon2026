package isureal

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
)

type exportPostResponse struct {
	createPostResponse
	ImagePath string `json:"imagePath"`
}

func exportPostsHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ctx := c.Request().Context()
		userID := GetSession(c).UserID
		author, err := getUser(ctx, h.DB, userID)
		if err != nil {
			return err
		}
		var posts []Post
		if err := h.DB.SelectContext(ctx, &posts, `
   SELECT `+postColumns+`
   FROM posts p
   JOIN posting_windows w ON w.id = p.posting_window_id
   WHERE p.author_id = $1 AND p.deleted_at IS NULL
   ORDER BY p.posted_at, p.id
  `, userID); err != nil {
			return err
		}

		// 署名の取り寄せ（往復 100ms）は画像の読み込みと並行して進める（#48）
		toSign := make([]imageToSign, 0, len(posts))
		for _, post := range posts {
			toSign = appendImagesToSign(toSign, author, post.ImageID)
		}
		var prefetched sync.WaitGroup
		prefetched.Go(func() { prefetchImageSignatures(ctx, toSign) })
		defer prefetched.Wait()

		// 画像本体は、メモリにあればそれを、ディスクにあればファイルから zip に直接流し、どちらにもないものだけ DB から取る。
		// ディスクの分をメモリのキャッシュに入れない: 1回で数百枚入れると、上限で全部捨てるキャッシュからフィードや
		// 画像配信の分まで追い出していた
		images := make(map[string][]byte, len(posts))
		onDisk := make(map[string]bool, len(posts))
		requested := make(map[string]struct{}, len(posts))
		var missing []Post
		for _, post := range posts {
			if data, ok := imageDataCache.get(post.ImageID); ok {
				images[post.ImageID] = data
			} else if f, _, ok := openCompleteImageFile(post.ImageID); ok {
				f.Close()
				onDisk[post.ImageID] = true
			} else if _, ok := requested[post.ImageID]; !ok {
				requested[post.ImageID] = struct{}{}
				missing = append(missing, post)
			}
		}
		fetches := startExportImageFetches(ctx, h.DB, missing)
		defer fetches.cancel()
		// 投稿順に並べた塊を前から順に待つ。後ろの塊を読んでいる間に、前の塊を zip に書いて送る
		next := 0
		advance := func() error {
			f := fetches.chunks[next]
			<-f.done
			if f.err != nil {
				return f.err
			}
			for k, v := range f.images {
				images[k] = v
			}
			next++
			return nil
		}
		// 先頭の塊が取れるまではヘッダを送らない（DB エラーなら今まで通り 500 にする）
		if len(fetches.chunks) > 0 {
			if err := advance(); err != nil {
				return err
			}
		}

		prefetched.Wait()
		// 取り寄せた署名が使えるうちに、全件のレスポンスを先に作る。zip を書きながら作ると、遅いエクスポート（平均1.35秒、
		// 最大8秒）では取り寄せた範囲を過ぎて、1件ごとに HMAC 1000回を手元で計算していた（s1 の CPU の約8%）
		responses := make([]createPostResponse, len(posts))
		for i, post := range posts {
			responses[i] = newPostResponse(post, author)
		}

		c.Response().Header().Set(echo.HeaderContentType, "application/zip")
		c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="isureal-posts.zip"`)
		c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
		c.Response().WriteHeader(http.StatusOK)
		writer := zip.NewWriter(c.Response())
		defer writer.Close()
		items := make([]exportPostResponse, 0, len(posts))
		for i, post := range posts {
			data, ok := images[post.ImageID]
			fromDisk := !ok && onDisk[post.ImageID]
			for !ok && !fromDisk && next < len(fetches.chunks) {
				if err := advance(); err != nil {
					return err
				}
				data, ok = images[post.ImageID]
			}
			if !ok && !fromDisk {
				// 元の JOIN image_objects と同じく、画像のない投稿は含めない
				continue
			}
			imagePath := fmt.Sprintf("images/%06d.jpg", len(items)+1)
			image, err := writer.CreateHeader(&zip.FileHeader{Name: imagePath, Method: zip.Store})
			if err != nil {
				return err
			}
			if fromDisk {
				err = copyImageFile(image, post.ImageID)
			} else {
				_, err = image.Write(data)
			}
			if err != nil {
				return err
			}
			items = append(items, exportPostResponse{createPostResponse: responses[i], ImagePath: imagePath})
		}
		manifest, err := writer.Create("posts.json")
		if err != nil {
			return err
		}
		if err := json.NewEncoder(manifest).Encode(items); err != nil {
			return err
		}
		return writer.Close()
	}
}

// 1本のクエリだと画像を1行ずつ順にディスクから読むので、投稿順に連続した塊に分けて同時に取る。
// 塊は全部同時に投げる（s2/s3 で待ちを1往復にするため）。DB 接続を食い尽くさないよう本数は抑える
const (
	exportFetchMaxChunks    = 3
	exportFetchMinChunkRows = 16
)

type exportImageChunk struct {
	images map[string][]byte
	err    error
	done   chan struct{}
}

type exportImageFetches struct {
	chunks []*exportImageChunk
	cancel func()
}

func startExportImageFetches(ctx context.Context, db *sqlx.DB, missing []Post) *exportImageFetches {
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	f := &exportImageFetches{cancel: func() { cancel(); wg.Wait() }}
	if len(missing) == 0 {
		return f
	}
	n := min(exportFetchMaxChunks, max(1, len(missing)/exportFetchMinChunkRows))
	for i := range n {
		part := missing[len(missing)*i/n : len(missing)*(i+1)/n]
		ch := &exportImageChunk{done: make(chan struct{})}
		f.chunks = append(f.chunks, ch)
		wg.Go(func() {
			defer close(ch.done)
			ch.images, ch.err = fetchExportImages(ctx, db, part)
		})
	}
	return f
}

func fetchExportImages(ctx context.Context, db *sqlx.DB, posts []Post) (map[string][]byte, error) {
	images := make(map[string][]byte, len(posts))
	missing := make([]string, 0, len(posts))
	postedAt := make([]time.Time, 0, len(posts))
	for _, post := range posts {
		missing = append(missing, post.ImageID)
		postedAt = append(postedAt, post.PostedAt)
	}
	// 投稿の画像は created_at = posts.posted_at なので、それでパーティションを絞る。
	// 時刻が合わずに見つからなかった分だけ、全パーティションを引く
	fetch := func(query string, args ...any) error {
		var rows []struct {
			ID   string `db:"id"`
			Data []byte `db:"data"`
		}
		if err := db.SelectContext(ctx, &rows, query, append([]any{binaryImage}, args...)...); err != nil {
			return err
		}
		for _, r := range rows {
			data, err := imageDataFromDB(r.ID, r.Data)
			if err != nil {
				return err
			}
			images[r.ID] = data
		}
		missing = slices.DeleteFunc(missing, func(id string) bool { _, ok := images[id]; return ok })
		return nil
	}
	// id = ANY AND created_at = ANY だと全パーティションで全 id を引くので、(id, created_at) の組で1パーティションずつ引く
	if err := fetch(`SELECT i.id, i.data FROM unnest($1::varchar[], $2::timestamptz[]) AS k(id, created_at) JOIN image_objects i ON i.id = k.id AND i.created_at = k.created_at`, missing, postedAt); err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		if err := fetch(`SELECT id, data FROM image_objects WHERE id = ANY($1)`, missing); err != nil {
			return nil, err
		}
	}
	return images, nil
}

// copyImageFile は openCompleteImageFile で確かめた画像ファイルを w に流す。ファイルは消さないので、確かめた後もある
func copyImageFile(w io.Writer, id string) error {
	f, _, ok := openCompleteImageFile(id)
	if !ok {
		return fmt.Errorf("image file disappeared: %s", id)
	}
	defer f.Close()
	_, err := io.Copy(w, f)
	return err
}
