package isureal

import (
	"bytes"
	"context"
	"encoding/hex"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// An image remembered as visible (#89) is served without asking the DB, and stops being served as soon as the write
// that hides it responds: post deletion (post image and its RealMoji images), profile image replacement and account
// deletion, all through the real handlers. Needs ISUREAL_TEST_DSN; rows are committed and removed at the end.
func TestImageVisibilityCacheRevokedByWrites(t *testing.T) {
	db := openExecModeTestDB(t)
	ctx := context.Background()
	imageDiskCacheDir = t.TempDir()
	imageDataCache = newBoundedCache(imageDataCacheMaxBytes, func(v []byte) int { return len(v) })
	resetImageMetaCaches()
	enabled := imageVisibilityCacheEnabled
	imageVisibilityCacheEnabled = true
	t.Cleanup(func() { imageVisibilityCacheEnabled = enabled; imageVisibility.revokeAll() })
	region := "test-vis-" + uuid.NewString()[:8]

	authorID, reactorID := uuid.NewString(), uuid.NewString()
	keys := map[string][]byte{}
	for i, id := range []string{authorID, reactorID} {
		key := bytes.Repeat([]byte{byte(0x21 * (i + 1))}, 32)
		keys[id] = key
		if _, err := db.ExecContext(ctx, `INSERT INTO users (id, username, secret_key_hash, display_name, region) VALUES ($1, $1, $2, 'd', $3)`, id, hex.EncodeToString(key), region); err != nil {
			t.Fatal(err)
		}
	}
	windowID := uuid.NewString()
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `INSERT INTO posting_windows (id, region, local_date, starts_at, on_time_deadline) VALUES ($1, $2, '2099-01-02', $3, $4)`,
		windowID, region, now.Add(-time.Minute), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM login_sessions WHERE user_id = ANY($1)`,
			`DELETE FROM realmojis WHERE reactor_id = ANY($1)`,
			`DELETE FROM posts WHERE author_id = ANY($1)`,
			`DELETE FROM image_objects WHERE owner_user_id = ANY($1)`,
			`UPDATE users SET profile_image_id = NULL WHERE id = ANY($1)`,
			`DELETE FROM users WHERE id = ANY($1)`,
		} {
			if _, err := db.ExecContext(ctx, q, []string{authorID, reactorID}); err != nil {
				t.Error(err)
			}
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM posting_windows WHERE id = $1`, windowID); err != nil {
			t.Error(err)
		}
	})

	images := map[string][]byte{}
	owners := map[string]string{}
	jpg := func(n int) []byte { return append(append([]byte{0xff, 0xd8}, testImage(n)...), 0xff, 0xd9) }
	add := func(id, owner string) {
		images[id] = jpg(1000)
		owners[id] = owner
		imageDataCache.put(id, images[id])
	}
	lat, lon := 35.681236, 139.767125
	postImageID := uuid.NewString()
	add(postImageID, authorID)
	var post Post
	if err := db.GetContext(ctx, &post, createPostQuery, binaryImage, authorID, region, postImageID, images[postImageID], time.Now().UTC(), uuid.NewString(),
		"caption", "public", "exact", &lat, &lon, (*string)(nil)); err != nil {
		t.Fatal(err)
	}
	realMojiImageID, sent := uuid.NewString(), time.Now().UTC()
	add(realMojiImageID, reactorID)
	if _, err := db.ExecContext(ctx, replaceRealMojiImageQuery, binaryImage, post.ID, reactorID, realMojiImageID, images[realMojiImageID], sent); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, createRealMojiQuery, uuid.NewString(), post.ID, reactorID, realMojiImageID, "like", sent); err != nil {
		t.Fatal(err)
	}
	profileImageID, reactorProfileImageID := uuid.NewString(), uuid.NewString()
	add(profileImageID, authorID)
	add(reactorProfileImageID, reactorID)
	for id, owner := range map[string]string{profileImageID: authorID, reactorProfileImageID: reactorID} {
		var createdAt time.Time
		if err := db.GetContext(ctx, &createdAt, putProfileImageQuery, binaryImage, owner, id, images[id], time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
	}

	// Write handlers behind a stub session middleware (X-Test-User).
	e := echo.New()
	e.HTTPErrorHandler = APIErrorHandler
	h := &HandlerContext{DB: db, Config: &Config{Region: region}}
	session := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("session", &Session{UserID: c.Request().Header.Get("X-Test-User"), Region: region})
			return next(c)
		}
	}
	e.DELETE("/api/posts/:postId", deletePostHandler(h), session)
	e.PUT("/api/me/profile-image", putProfileImageHandler(h), session)
	e.POST("/api/me/account-deletion", deleteAccountHandler(h), session)
	writes := httptest.NewServer(e)
	t.Cleanup(writes.Close)
	write := func(method, path, user, contentType string, body []byte) {
		t.Helper()
		req, err := http.NewRequest(method, writes.URL+path, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Test-User", user)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("%s %s: status %d", method, path, resp.StatusCode)
		}
	}

	server := newImageGetServer(t, db)
	exp := time.Now().Add(time.Hour).Unix()
	get := func(id string) int {
		t.Helper()
		return server.get(id, exp, imageSignature(keys[owners[id]], id, exp)).Status
	}
	// Each image is served and then remembered as visible, so the next GET would be answered from memory.
	all := []string{postImageID, realMojiImageID, profileImageID, reactorProfileImageID}
	for _, id := range all {
		if got := get(id); got != http.StatusOK {
			t.Fatalf("before writes %s: status %d", id, got)
		}
		if _, ok := imageVisibility.get(id); !ok {
			t.Fatalf("%s was not remembered as visible", id)
		}
		if res := server.get(id, exp, imageSignature(keys[owners[id]], id, exp+1)); res.Status != http.StatusForbidden || res.Abuse != "invalid_signature" {
			t.Fatalf("cached %s with a bad signature: %+v", id, res)
		}
	}

	// Profile image replacement hides the old one at once.
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	write(http.MethodPut, "/api/me/profile-image", authorID, "image/jpeg", buf.Bytes())
	if got := get(profileImageID); got != http.StatusForbidden {
		t.Fatalf("replaced profile image: status %d", got)
	}

	// Post deletion hides the post image and the RealMoji images on it.
	write(http.MethodDelete, "/api/posts/"+post.ID, authorID, "", nil)
	for _, id := range []string{postImageID, realMojiImageID} {
		if got := get(id); got != http.StatusForbidden {
			t.Fatalf("deleted post's image %s: status %d", id, got)
		}
	}

	// Account deletion hides all of that user's images.
	if got := get(reactorProfileImageID); got != http.StatusOK {
		t.Fatalf("reactor profile image before deletion: status %d", got)
	}
	write(http.MethodPost, "/api/me/account-deletion", reactorID, "", nil)
	if got := get(reactorProfileImageID); got != http.StatusForbidden {
		t.Fatalf("withdrawn user's profile image: status %d", got)
	}
}
