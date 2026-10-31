package isureal

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBoundedCacheKeepsFirstValue(t *testing.T) {
	c := newBoundedCache(10, func(v []byte) int { return len(v) })
	c.put("a", []byte("x"))
	c.put("a", []byte("y"))
	if v, _ := c.get("a"); string(v) != "x" {
		t.Fatalf("got %q, want first value", v)
	}
}

func TestBoundedCacheDropsAllWhenOverMax(t *testing.T) {
	c := newBoundedCache(3, func(int) int { return 1 })
	for i, k := range []string{"a", "b", "c"} {
		c.put(k, i)
	}
	for _, k := range []string{"a", "b", "c"} {
		if _, ok := c.get(k); !ok {
			t.Fatalf("%s evicted before reaching max", k)
		}
	}
	c.put("d", 3)
	for _, k := range []string{"a", "b", "c"} {
		if _, ok := c.get(k); ok {
			t.Fatalf("%s survived overflow", k)
		}
	}
	if v, ok := c.get("d"); !ok || v != 3 {
		t.Fatalf("d = %v, %v after overflow", v, ok)
	}
}

func TestBoundedCacheCountsBytes(t *testing.T) {
	c := newBoundedCache(4, func(v []byte) int { return len(v) })
	c.put("a", []byte("abc"))
	c.put("b", []byte("d"))
	if _, ok := c.get("a"); !ok {
		t.Fatal("a evicted at exactly max bytes")
	}
	c.put("c", []byte("e"))
	if _, ok := c.get("a"); ok {
		t.Fatal("a survived exceeding max bytes")
	}
}

func TestImageDiskCacheRoundTripAndRejectsNonUUID(t *testing.T) {
	imageDiskCacheDir = t.TempDir()
	id := "0123abcd-0000-4000-8000-000000000001"
	writeImageFile(id, []byte("\xff\xd8jpeg\xff\xd9"))
	imageDataCache = newBoundedCache(imageDataCacheMaxBytes, func(v []byte) int { return len(v) })
	if data, ok := getImageData(id); !ok || !bytes.Equal(data, []byte("\xff\xd8jpeg\xff\xd9")) {
		t.Fatalf("got %q, %v from disk", data, ok)
	}
	for _, bad := range []string{"../../etc/passwd", "0123ABCD-0000-4000-8000-000000000001", "x"} {
		writeImageFile(bad, []byte("x"))
		if _, ok := getImageData(bad); ok {
			t.Fatalf("%q was cached", bad)
		}
	}
	entries, _ := filepath.Glob(filepath.Join(imageDiskCacheDir, "*", "*"))
	if len(entries) != 1 { // only the valid UUID was written
		t.Fatalf("files on disk: %v", entries)
	}
	// a truncated file (no JPEG end marker) is not used
	id3 := "0123abcd-0000-4000-8000-000000000003"
	os.MkdirAll(filepath.Join(imageDiskCacheDir, "01"), 0o755)
	os.WriteFile(filepath.Join(imageDiskCacheDir, "01", id3), []byte("\xff\xd8trunc"), 0o644)
	os.WriteFile(filepath.Join(imageDiskCacheDir, "01", "0123abcd-0000-4000-8000-000000000004"), nil, 0o644)
	for _, id := range []string{id3, "0123abcd-0000-4000-8000-000000000004"} {
		if _, ok := getImageData(id); ok {
			t.Fatalf("%s: incomplete file was used", id)
		}
	}
	// putImageData writes in the background
	id2 := "0123abcd-0000-4000-8000-000000000002"
	putImageData(id2, []byte("jpeg2"))
	for i := 0; i < 100; i++ {
		if b, err := os.ReadFile(filepath.Join(imageDiskCacheDir, "01", id2)); err == nil && string(b) == "jpeg2" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("putImageData did not write the file")
}

func TestImageAccelPathOnlyForCompleteReadableFiles(t *testing.T) {
	imageDiskCacheDir = t.TempDir()
	dir := filepath.Join(imageDiskCacheDir, "ab")
	os.MkdirAll(dir, 0o755)
	png := append(append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte("p"), 100)...), "IEND\xaeB`\x82"...)
	cases := map[string]struct {
		data []byte
		perm os.FileMode
		want bool
	}{
		"abcd0000-0000-4000-8000-000000000001": {[]byte("\xff\xd8jpeg\xff\xd9"), 0o644, true}, // shorter than 16 bytes
		"abcd0000-0000-4000-8000-000000000002": {png, 0o644, true},
		"abcd0000-0000-4000-8000-000000000003": {png[:len(png)-1], 0o644, false}, // truncated
		"abcd0000-0000-4000-8000-000000000004": {nil, 0o644, false},
		"abcd0000-0000-4000-8000-000000000005": {png, 0o600, false}, // nginx (www-data) cannot read it
	}
	for id, tc := range cases {
		os.WriteFile(filepath.Join(dir, id), tc.data, tc.perm)
		os.Chmod(filepath.Join(dir, id), tc.perm)
		path, ok := imageAccelPath(id)
		if ok != tc.want || (ok && path != "/_image_cache/ab/"+id) {
			t.Errorf("%s: got %q, %v", id, path, ok)
		}
	}
	if _, ok := imageAccelPath("abcd0000-0000-4000-8000-000000000009"); ok {
		t.Error("missing file was accepted")
	}
	// files written by writeImageFile are readable by nginx
	id := "abcd0000-0000-4000-8000-000000000006"
	writeImageFile(id, png)
	if _, ok := imageAccelPath(id); !ok {
		t.Error("file written by writeImageFile cannot be handed to nginx")
	}
}

func TestFromLocalNginx(t *testing.T) {
	for _, tc := range []struct {
		remote, marker string
		want           bool
	}{
		{"127.0.0.1:51234", "1", true},
		{"127.0.0.1:51234", "", false},
		{"192.168.2.10:51234", "1", false}, // s2/s3 nginx pass the client's headers through to s1:8080
	} {
		r := httptest.NewRequest(http.MethodGet, "/api/images/x", nil)
		r.RemoteAddr = tc.remote
		if tc.marker != "" {
			r.Header.Set("X-Isureal-Image-Accel", tc.marker)
		}
		if got := fromLocalNginx(r); got != tc.want {
			t.Errorf("%+v: got %v", tc, got)
		}
	}
}

// アップロードされた画像は DB に空の data で入るので、DB から読んだ空の data はディスクの本体に置き換わる（#81）
func TestUploadedImageReadsFromDiskWhenDBDataIsEmpty(t *testing.T) {
	saved := imageDiskCacheDir
	imageDiskCacheDir = t.TempDir()
	defer func() { imageDiskCacheDir = saved }()
	id := "0123abcd-0000-4000-8000-000000000081"
	jpeg := []byte{0xff, 0xd8, 'x', 0xff, 0xd9}
	if err := storeUploadedImage(id, jpeg); err != nil {
		t.Fatal(err)
	}
	imageDataCache = newBoundedCache(imageDataCacheMaxBytes, func(v []byte) int { return len(v) })
	got, err := imageDataFromDB(id, fileBackedImageData)
	if err != nil || !bytes.Equal(got, jpeg) {
		t.Fatalf("got %v, %v; want the stored file", got, err)
	}
	if _, err := imageDataFromDB("0123abcd-0000-4000-8000-000000000000", fileBackedImageData); err == nil {
		t.Fatal("empty data without a file must be an error, not an empty image")
	}
}

// 初期画像のメタは [16]byte とマイクロ秒で持つので、引いた値が DB の値（マイクロ秒の timestamptz）と一致すること
func TestLookupInitialImageMetaRoundTrip(t *testing.T) {
	saved := initialImageMeta.Load()
	defer initialImageMeta.Store(saved)
	id := "c74e7f85-dfbd-48aa-bf2f-601bdc05f29f"
	createdAt := time.Date(2024, 10, 29, 16, 49, 20, 259487000, time.UTC)
	table := &initialImageMetaTable{byID: map[[16]byte]initialImageMetaEntry{}, owners: []string{"owner-a"}, types: []string{"image/jpeg"}}
	table.byID[uuid.MustParse(id)] = initialImageMetaEntry{createdAt: createdAt.UnixMicro(), owner: 0, mediaType: 0}
	initialImageMeta.Store(table)
	got, ok := lookupInitialImageMeta(id)
	if !ok || got.Owner != "owner-a" || got.MediaType != "image/jpeg" || !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("got %+v, %v", got, ok)
	}
	if _, ok := lookupInitialImageMeta("not-a-uuid"); ok {
		t.Fatal("a non-UUID id must miss")
	}
}
