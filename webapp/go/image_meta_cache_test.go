package isureal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
)

type imageGetResult struct {
	Status      int
	ContentType string
	CacheCtl    string
	Body        []byte
	Abuse       string
}

// imageGetServer serves GET /api/images through the real handler and records the abuse flag of each response.
type imageGetServer struct {
	t      *testing.T
	url    string
	abuses chan string
}

func newImageGetServer(t *testing.T, db *sqlx.DB) *imageGetServer {
	t.Helper()
	s := &imageGetServer{t: t, abuses: make(chan string, 1)}
	e := echo.New()
	e.HTTPErrorHandler = APIErrorHandler
	h := &HandlerContext{DB: db, Config: &Config{Region: "asia"}}
	e.GET("/api/images/:imageId", getImageHandler(h), func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			err := next(c)
			abuse, _ := c.Get("abuse").(string)
			s.abuses <- abuse
			return err
		}
	})
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	s.url = server.URL
	return s
}

func (s *imageGetServer) get(id string, exp int64, sig []byte) imageGetResult {
	s.t.Helper()
	resp, err := http.Get(s.url + "/api/images/" + id + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + base64.RawURLEncoding.EncodeToString(sig))
	if err != nil {
		s.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.t.Fatal(err)
	}
	return imageGetResult{Status: resp.StatusCode, ContentType: resp.Header.Get("Content-Type"), CacheCtl: resp.Header.Get("Cache-Control"), Body: body, Abuse: <-s.abuses}
}

// GET /api/images answers the same (status, headers, body, abuse flag) whether the image's owner/media_type/created_at
// come from imageMetaCache (users/posts/realmojis only) or from image_objects, in every visibility state: served,
// bad signature, expired, post deleted (rejected at once, well within the 5 s rule), RealMoji replaced, profile image
// replaced (row deleted), owner withdrawn. Needs ISUREAL_TEST_DSN; rows are committed and removed at the end.
func TestGetImageMetaCacheHitMatchesMiss(t *testing.T) {
	db := openExecModeTestDB(t)
	ctx := context.Background()
	imageDiskCacheDir = t.TempDir()
	imageDataCache = newBoundedCache(imageDataCacheMaxBytes, func(v []byte) int { return len(v) })
	resetImageMetaCaches()
	region := "test-meta-" + uuid.NewString()[:8]

	authorID, reactorID := uuid.NewString(), uuid.NewString()
	keys := map[string][]byte{}
	for i, id := range []string{authorID, reactorID} {
		key := bytes.Repeat([]byte{byte(0x11 * (i + 1))}, 32)
		keys[id] = key
		if _, err := db.ExecContext(ctx, `INSERT INTO users (id, username, secret_key_hash, display_name, region) VALUES ($1, $1, $2, 'd', $3)`, id, hex.EncodeToString(key), region); err != nil {
			t.Fatal(err)
		}
	}
	windowID := uuid.NewString()
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `INSERT INTO posting_windows (id, region, local_date, starts_at, on_time_deadline) VALUES ($1, $2, '2099-01-01', $3, $4)`,
		windowID, region, now.Add(-time.Minute), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, q := range []string{
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
	jpeg := func(n int) []byte { return append(append([]byte{0xff, 0xd8}, testImage(n)...), 0xff, 0xd9) }
	lat, lon := 35.681236, 139.767125
	postImageID := uuid.NewString()
	images[postImageID] = jpeg(1000)
	var post Post
	if err := db.GetContext(ctx, &post, createPostQuery, binaryImage, authorID, region, postImageID, images[postImageID], time.Now().UTC(), uuid.NewString(),
		"caption", "public", "exact", &lat, &lon, (*string)(nil)); err != nil {
		t.Fatal(err)
	}
	realMojiImageID, sent := uuid.NewString(), time.Now().UTC()
	images[realMojiImageID] = jpeg(1000)
	if _, err := db.ExecContext(ctx, replaceRealMojiImageQuery, binaryImage, post.ID, reactorID, realMojiImageID, images[realMojiImageID], sent); err != nil {
		t.Fatal(err)
	}
	var reaction RealMoji
	if err := db.GetContext(ctx, &reaction, createRealMojiQuery, uuid.NewString(), post.ID, reactorID, realMojiImageID, "like", sent); err != nil {
		t.Fatal(err)
	}
	profileImageID := uuid.NewString()
	images[profileImageID] = jpeg(1000)
	var profileCreatedAt time.Time
	if err := db.GetContext(ctx, &profileCreatedAt, putProfileImageQuery, binaryImage, authorID, profileImageID, images[profileImageID], time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	owners := map[string]string{postImageID: authorID, realMojiImageID: reactorID, profileImageID: authorID}
	// metas holds what the upload handlers remember (rememberUploadedImageMeta) for each image.
	metas := map[string]imageMeta{
		postImageID:     {Owner: authorID, MediaType: "image/jpeg", CreatedAt: post.PostedAt},
		realMojiImageID: {Owner: reactorID, MediaType: "image/jpeg", CreatedAt: reaction.CreatedAt},
		profileImageID:  {Owner: authorID, MediaType: "image/jpeg", CreatedAt: profileCreatedAt},
	}
	for id, data := range images {
		imageDataCache.put(id, data)
	}

	server := newImageGetServer(t, db)
	exp := time.Now().Add(time.Hour).Unix()
	// compare runs the request on a warm cache (what the upload handler remembered) and on an empty one, and checks
	// both answers against want.
	compare := func(name, id string, sig []byte, want imageGetResult) {
		t.Helper()
		resetImageMetaCaches()
		imageMetaCache.put(id, metas[id])
		hit := server.get(id, exp, sig)
		// miss goes through imageReferrerQuery when the bytes are at hand (#90); old is the image_objects path.
		resetImageMetaCaches()
		miss := server.get(id, exp, sig)
		resetImageMetaCaches()
		imageReferrerLookupEnabled = false
		old := server.get(id, exp, sig)
		imageReferrerLookupEnabled = true
		for label, got := range map[string]imageGetResult{"hit": hit, "miss": miss, "old": old} {
			if got.Status != want.Status || got.ContentType != want.ContentType || got.CacheCtl != want.CacheCtl || !bytes.Equal(got.Body, want.Body) || got.Abuse != want.Abuse {
				t.Fatalf("%s (%s): got status=%d type=%q cache=%q abuse=%q body=%q\nwant status=%d type=%q cache=%q abuse=%q body=%q",
					name, label, got.Status, got.ContentType, got.CacheCtl, got.Abuse, got.Body, want.Status, want.ContentType, want.CacheCtl, want.Abuse, want.Body)
			}
		}
		if want.Status == http.StatusOK {
			if _, ok := imageMetaCache.get(id); !ok {
				t.Fatalf("%s: the image_objects path did not fill imageMetaCache", name)
			}
		}
	}
	forbidden := imageGetResult{Status: http.StatusForbidden, ContentType: "application/json", CacheCtl: "no-store", Body: []byte(`{"error":"image_forbidden"}` + "\n")}
	withAbuse := forbidden
	withAbuse.Abuse = "invalid_signature"
	served := func(id string) imageGetResult {
		return imageGetResult{Status: http.StatusOK, ContentType: "image/jpeg", CacheCtl: "no-store", Body: images[id]}
	}
	sign := func(id string) []byte { return imageSignature(keys[owners[id]], id, exp) }
	badSig := func(id string) []byte {
		sig := bytes.Clone(sign(id))
		sig[0] ^= 1
		return sig
	}

	// The upload handlers fill both caches from the values they return.
	rememberUploadedImageMeta(postImageID, authorID, post.PostedAt)
	if got, ok := imageMetaCache.get(postImageID); !ok || got != metas[postImageID] {
		t.Fatalf("rememberUploadedImageMeta stored %+v, %v", got, ok)
	}
	if got, ok := imageCreatedAtCache.get(postImageID); !ok || !got.Equal(post.PostedAt) {
		t.Fatalf("rememberUploadedImageMeta stored created_at %v, %v", got, ok)
	}
	for _, id := range []string{postImageID, realMojiImageID, profileImageID} {
		compare("served "+id, id, sign(id), served(id))
		compare("bad signature "+id, id, badSig(id), withAbuse)
		compare("wrong owner's key "+id, id, imageSignature(keys[reactorID], id, exp), map[bool]imageGetResult{true: withAbuse, false: served(id)}[owners[id] != reactorID])
	}
	for _, id := range []string{postImageID, profileImageID} {
		imageMetaCache.put(id, metas[id])
		res := server.get(id, time.Now().Unix()-1, imageSignature(keys[owners[id]], id, time.Now().Unix()-1))
		if res.Status != http.StatusForbidden || res.Abuse != "" {
			t.Fatalf("expired: %+v", res)
		}
	}

	// Replacing the RealMoji hides the old image at once; the new one is served.
	replacementImageID, sent := uuid.NewString(), time.Now().UTC()
	images[replacementImageID] = jpeg(1000)
	owners[replacementImageID] = reactorID
	imageDataCache.put(replacementImageID, images[replacementImageID])
	if _, err := db.ExecContext(ctx, replaceRealMojiImageQuery, binaryImage, post.ID, reactorID, replacementImageID, images[replacementImageID], sent); err != nil {
		t.Fatal(err)
	}
	if err := db.GetContext(ctx, &reaction, createRealMojiQuery, uuid.NewString(), post.ID, reactorID, replacementImageID, "smile", sent); err != nil {
		t.Fatal(err)
	}
	metas[replacementImageID] = imageMeta{Owner: reactorID, MediaType: "image/jpeg", CreatedAt: reaction.CreatedAt}
	compare("replaced realmoji", realMojiImageID, sign(realMojiImageID), forbidden)
	compare("replaced realmoji, bad signature", realMojiImageID, badSig(realMojiImageID), withAbuse)
	compare("replacement realmoji", replacementImageID, sign(replacementImageID), served(replacementImageID))

	// Replacing the profile image deletes the old image_objects row: no row means 403 before the signature check.
	newProfileImageID := uuid.NewString()
	images[newProfileImageID] = jpeg(1000)
	owners[newProfileImageID] = authorID
	imageDataCache.put(newProfileImageID, images[newProfileImageID])
	tx := beginTestTx(t, db)
	var old profileImageRef
	if err := tx.GetContext(ctx, &old, lockProfileImageQuery, authorID); err != nil {
		t.Fatal(err)
	}
	if err := tx.GetContext(ctx, &profileCreatedAt, putProfileImageQuery, binaryImage, authorID, newProfileImageID, images[newProfileImageID], time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := deleteImageObject(ctx, tx, old); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	metas[newProfileImageID] = imageMeta{Owner: authorID, MediaType: "image/jpeg", CreatedAt: profileCreatedAt}
	if _, ok := imageMetaCache.get(profileImageID); ok {
		t.Fatal("deleted profile image still in imageMetaCache")
	}
	// A stale entry for the deleted image (left by a request racing the delete) must not change the answer either.
	compare("replaced profile image", profileImageID, sign(profileImageID), forbidden)
	compare("replaced profile image, bad signature", profileImageID, badSig(profileImageID), forbidden)
	compare("new profile image", newProfileImageID, sign(newProfileImageID), served(newProfileImageID))

	// Deleting the post hides its image at once (the rule allows 5 s); the RealMoji on it goes with it.
	if _, err := db.ExecContext(ctx, `UPDATE posts SET deleted_at = clock_timestamp() WHERE id = $1`, post.ID); err != nil {
		t.Fatal(err)
	}
	compare("deleted post", postImageID, sign(postImageID), forbidden)
	compare("deleted post, bad signature", postImageID, badSig(postImageID), withAbuse)
	compare("realmoji on deleted post", replacementImageID, sign(replacementImageID), forbidden)
	if _, err := db.ExecContext(ctx, `UPDATE posts SET deleted_at = NULL WHERE id = $1`, post.ID); err != nil {
		t.Fatal(err)
	}
	compare("restored post", postImageID, sign(postImageID), served(postImageID))

	// Withdrawal: the owner's images stop at once, and so do RealMojis sent to the withdrawn author's posts.
	if _, err := db.ExecContext(ctx, `UPDATE users SET status = 'deleted' WHERE id = $1`, authorID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{postImageID, newProfileImageID, replacementImageID} {
		compare("withdrawn author: "+id, id, sign(id), forbidden)
		compare("withdrawn author, bad signature: "+id, id, badSig(id), map[bool]imageGetResult{true: forbidden, false: withAbuse}[owners[id] == authorID])
	}

	// Without the image bytes at hand the cache is not used and the data still comes from image_objects.
	if _, err := db.ExecContext(ctx, `UPDATE users SET status = 'active' WHERE id = $1`, authorID); err != nil {
		t.Fatal(err)
	}
	imageDataCache = newBoundedCache(imageDataCacheMaxBytes, func(v []byte) int { return len(v) })
	compare("served from image_objects", postImageID, sign(postImageID), served(postImageID))
	// A wrong created_at in the cache (never written by the app) only costs the id-only query.
	imageDataCache.put(postImageID, images[postImageID])
	metas[postImageID] = imageMeta{Owner: authorID, MediaType: "image/jpeg", CreatedAt: time.Now()}
	compare("wrong created_at", postImageID, sign(postImageID), served(postImageID))
	if !strings.Contains(imageOwnerQuery, "FROM users u WHERE") || strings.Contains(imageOwnerQuery, "image_objects") {
		t.Fatal("imageOwnerQuery must not read image_objects")
	}
}
