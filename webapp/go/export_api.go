package isureal

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

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

		// 画像本体はキャッシュにないものだけ DB から1回で取る
		images := make(map[string][]byte, len(posts))
		var missing []string
		var missingPostedAt []time.Time
		for _, post := range posts {
			if data, ok := imageDataCache.get(post.ImageID); ok {
				images[post.ImageID] = data
			} else {
				missing = append(missing, post.ImageID)
				missingPostedAt = append(missingPostedAt, post.PostedAt)
			}
		}
		// 投稿の画像は created_at = posts.posted_at なので、それでパーティションを絞る。
		// 時刻が合わずに見つからなかった分だけ、全パーティションを引く
		fetch := func(query string, args ...any) error {
			var rows []struct {
				ID   string `db:"id"`
				Data []byte `db:"data"`
			}
			if err := h.DB.SelectContext(ctx, &rows, query, args...); err != nil {
				return err
			}
			for _, r := range rows {
				images[r.ID] = r.Data
				imageDataCache.put(r.ID, r.Data)
			}
			missing = slices.DeleteFunc(missing, func(id string) bool { _, ok := images[id]; return ok })
			return nil
		}
		if len(missing) > 0 {
			if err := fetch(`SELECT id, data FROM image_objects WHERE id = ANY($1) AND created_at = ANY($2::timestamptz[])`, missing, missingPostedAt); err != nil {
				return err
			}
		}
		if len(missing) > 0 {
			if err := fetch(`SELECT id, data FROM image_objects WHERE id = ANY($1)`, missing); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, "application/zip")
		c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="isureal-posts.zip"`)
		c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
		c.Response().WriteHeader(http.StatusOK)
		writer := zip.NewWriter(c.Response())
		defer writer.Close()
		items := make([]exportPostResponse, 0, len(posts))
		for _, post := range posts {
			data, ok := images[post.ImageID]
			if !ok {
				// 元の JOIN image_objects と同じく、画像のない投稿は含めない
				continue
			}
			imagePath := fmt.Sprintf("images/%06d.jpg", len(items)+1)
			image, err := writer.CreateHeader(&zip.FileHeader{Name: imagePath, Method: zip.Store})
			if err != nil {
				return err
			}
			if _, err := image.Write(data); err != nil {
				return err
			}
			items = append(items, exportPostResponse{createPostResponse: newPostResponse(post, author), ImagePath: imagePath})
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
