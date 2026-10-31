package isureal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

type uploadPart struct {
	name, filename, value string
}

func multipartBody(t testing.TB, parts []uploadPart) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for _, p := range parts {
		var (
			fw  io.Writer
			err error
		)
		if p.filename != "" {
			fw, err = w.CreateFormFile(p.name, p.filename)
		} else {
			fw, err = w.CreateFormField(p.name)
		}
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(p.value))
	}
	w.Close()
	return &body, w.FormDataContentType()
}

func postContext(body *bytes.Buffer, contentType string) *echo.Context {
	req := httptest.NewRequest(http.MethodPost, "/api/posts?x=1", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(body.Len())
	return echo.New().NewContext(req, httptest.NewRecorder())
}

// the reading the handlers did before upload_read.go: ParseMultipartForm via Bind, then Open + io.ReadAll
func readCreatePostOld(c *echo.Context) (createPostRequest, []byte, error) {
	var input createPostRequest
	if err := c.Bind(&input); err != nil {
		return input, nil, err
	}
	if input.Image == nil {
		return input, nil, nil
	}
	f, err := input.Image.Open()
	if err != nil {
		return input, nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	return input, data, err
}

func readCreatePostNew(c *echo.Context) (createPostRequest, []byte, error) {
	var input createPostRequest
	data, err := readMultipartUpload(c.Request(), "image")
	if err != nil {
		return input, nil, echo.ErrBadRequest.Wrap(err)
	}
	if err := c.Bind(&input); err != nil {
		return input, nil, err
	}
	if input.Image == nil {
		return input, nil, nil
	}
	return input, data, nil
}

func fields(in createPostRequest) string {
	deref := func(p any) string {
		switch v := p.(type) {
		case *float64:
			if v != nil {
				return fmt.Sprint(*v)
			}
		case *string:
			if v != nil {
				return *v
			}
		}
		return "<nil>"
	}
	return fmt.Sprintf("%q %q %q %s %s %s", in.Caption, in.Visibility, in.LocationKind, deref(in.Latitude), deref(in.Longitude), deref(in.Locality))
}

func errText(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

// readMultipartUpload + Bind binds the same fields, bytes and errors as ParseMultipartForm + Bind + Open + ReadAll
func TestReadMultipartUploadMatchesParseMultipartForm(t *testing.T) {
	big := strings.Repeat("\xff\xd8jpeg", 300_000) // 1.8MB
	lat, lon := 35.5, 139.7
	cases := map[string][]uploadPart{
		"image first": {{"image", "a.jpg", big}, {"caption", "", "hi"}, {"visibility", "", "friends"}, {"location_kind", "", "coordinates"}, {"latitude", "", "35.5"}, {"longitude", "", "139.7"}},
		"image last":  {{"caption", "", "こんにちは"}, {"visibility", "", "public"}, {"location_kind", "", "locality"}, {"locality", "", "Tokyo"}, {"image", "a.jpg", big}},
		"empty image": {{"image", "a.jpg", ""}, {"caption", "", "x"}},
		"two images":  {{"image", "1.jpg", "first"}, {"image", "2.jpg", "second"}, {"other", "o.bin", "ignored"}},
		"no image":    {{"caption", "", "x"}, {"visibility", "", "public"}},
		"bad float":   {{"image", "a.jpg", "x"}, {"latitude", "", "north"}},
		"no name":     {{"", "", "anonymous"}, {"image", "a.jpg", "x"}},
	}
	for name, parts := range cases {
		t.Run(name, func(t *testing.T) {
			body, ct := multipartBody(t, parts)
			oldIn, oldData, oldErr := readCreatePostOld(postContext(body, ct))
			newIn, newData, newErr := readCreatePostNew(postContext(body, ct))
			if errText(oldErr) != errText(newErr) {
				t.Fatalf("error: old %q, new %q", errText(oldErr), errText(newErr))
			}
			if !bytes.Equal(oldData, newData) {
				t.Fatalf("image bytes differ: old %d, new %d", len(oldData), len(newData))
			}
			if (oldIn.Image == nil) != (newIn.Image == nil) {
				t.Fatalf("image header: old %v, new %v", oldIn.Image, newIn.Image)
			}
			if oldIn.Image != nil && (oldIn.Image.Filename != newIn.Image.Filename || oldIn.Image.Size != newIn.Image.Size) {
				t.Fatalf("image header: old %+v, new %+v", oldIn.Image, newIn.Image)
			}
			if fields(oldIn) != fields(newIn) {
				t.Fatalf("fields: old %s, new %s", fields(oldIn), fields(newIn))
			}
			if newData != nil && cap(newData)-len(newData) > uploadSlackBytes {
				t.Fatalf("buffer slack %d > %d: not sized from Content-Length", cap(newData)-len(newData), uploadSlackBytes)
			}
			if name == "image first" && (*newIn.Latitude != lat || *newIn.Longitude != lon) {
				t.Fatalf("coordinates %v %v", *newIn.Latitude, *newIn.Longitude)
			}
		})
	}
}

func TestReadMultipartUploadErrorsMatchParseMultipartForm(t *testing.T) {
	good, ct := multipartBody(t, []uploadPart{{"image", "a.jpg", "x"}, {"caption", "", "c"}})
	cases := map[string]struct {
		body        []byte
		contentType string
	}{
		"json":             {[]byte(`{}`), "application/json"},
		"no content type":  {good.Bytes(), ""},
		"missing boundary": {good.Bytes(), "multipart/form-data"},
		"wrong boundary":   {good.Bytes(), "multipart/form-data; boundary=nope"},
		"truncated":        {good.Bytes()[:good.Len()/2], ct},
		"garbage":          {[]byte("--x\r\nnot a header\r\n"), ct},
		"empty body":       {nil, ct},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, oldData, oldErr := readCreatePostOld(postContext(bytes.NewBuffer(tc.body), tc.contentType))
			_, newData, newErr := readCreatePostNew(postContext(bytes.NewBuffer(tc.body), tc.contentType))
			if errText(oldErr) != errText(newErr) || !bytes.Equal(oldData, newData) {
				t.Fatalf("old (%q, %d bytes), new (%q, %d bytes)", errText(oldErr), len(oldData), errText(newErr), len(newData))
			}
		})
	}
}

// an upload that fills nginx's client_max_body_size still arrives in one buffer, not on disk
func TestReadMultipartUploadMaxSize(t *testing.T) {
	image := bytes.Repeat([]byte{0xff}, maxUploadBytes-1024)
	body, ct := multipartBody(t, []uploadPart{{"caption", "", "big"}, {"image", "a.jpg", string(image)}})
	if body.Len() > maxUploadBytes {
		t.Fatalf("body %d exceeds the nginx limit", body.Len())
	}
	input, data, err := readCreatePostNew(postContext(body, ct))
	if err != nil || input.Image == nil || input.Image.Size != int64(len(image)) || !bytes.Equal(data, image) {
		t.Fatalf("err=%v header=%+v len=%d", err, input.Image, len(data))
	}
	if cap(data)-len(data) > uploadSlackBytes {
		t.Fatalf("buffer slack %d", cap(data)-len(data))
	}
}

func jpegBytes(t testing.TB, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestReadCreatePostInputReturnsImageBytes(t *testing.T) {
	img := jpegBytes(t, 64, 48)
	body, ct := multipartBody(t, []uploadPart{{"caption", "", "c"}, {"visibility", "", "public"}, {"location_kind", "", "none"}, {"image", "a.jpg", string(img)}})
	input, loc, data, err := readCreatePostInput(postContext(body, ct))
	if err != nil || !bytes.Equal(data, img) || input.Caption != "c" || loc.Kind != "none" {
		t.Fatalf("err=%v input=%+v loc=%+v len=%d", err, input, loc, len(data))
	}
	var apiErr *APIError
	for name, parts := range map[string][]uploadPart{
		"no image":       {{"caption", "", "c"}, {"visibility", "", "public"}, {"location_kind", "", "none"}},
		"not a jpeg":     {{"visibility", "", "public"}, {"location_kind", "", "none"}, {"image", "a.jpg", "plain"}},
		"too large":      {{"visibility", "", "public"}, {"location_kind", "", "none"}, {"image", "a.jpg", string(jpegBytes(t, 4097, 1))}},
		"bad visibility": {{"visibility", "", "secret"}, {"location_kind", "", "none"}, {"image", "a.jpg", string(img)}},
	} {
		body, ct := multipartBody(t, parts)
		_, _, _, err := readCreatePostInput(postContext(body, ct))
		if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest || apiErr.Code != "invalid_post" {
			t.Fatalf("%s: err=%v", name, err)
		}
	}
}

func TestReadReplaceRealMojiInputReturnsImageBytes(t *testing.T) {
	img := jpegBytes(t, 512, 512)
	body, ct := multipartBody(t, []uploadPart{{"image", "a.jpg", string(img)}, {"reactionType", "", "heart"}})
	c := postContext(body, ct)
	c.SetPathValues(echo.PathValues{{Name: "postId", Value: "p1"}})
	input, data, err := readReplaceRealMojiInput(c)
	if err != nil || !bytes.Equal(data, img) || input.ReactionType != "heart" || input.PostID != "p1" {
		t.Fatalf("err=%v input=%+v len=%d", err, input, len(data))
	}
	var apiErr *APIError
	for name, parts := range map[string][]uploadPart{
		"no image":     {{"reactionType", "", "heart"}},
		"bad reaction": {{"image", "a.jpg", string(img)}, {"reactionType", "", "meh"}},
		"too large":    {{"image", "a.jpg", string(jpegBytes(t, 513, 1))}, {"reactionType", "", "heart"}},
		"not a jpeg":   {{"image", "a.jpg", "plain"}, {"reactionType", "", "heart"}},
	} {
		body, ct := multipartBody(t, parts)
		_, _, err := readReplaceRealMojiInput(postContext(body, ct))
		if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest || apiErr.Code != "invalid_realmoji" {
			t.Fatalf("%s: err=%v", name, err)
		}
	}
}

// readBodyExact returns the same bytes and errors as io.ReadAll for a full body, a body shorter than its
// Content-Length (the connection closes early) and a chunked body
func TestReadBodyExactMatchesReadAll(t *testing.T) {
	type result struct {
		n   int
		err string
	}
	results := make(chan [2]result, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var out [2]result
		if r.URL.Path == "/old" {
			data, err := io.ReadAll(r.Body)
			out[0] = result{len(data), errText(err)}
		} else {
			data, err := readBodyExact(r)
			out[1] = result{len(data), errText(err)}
		}
		results <- out
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	addr := strings.TrimPrefix(server.URL, "http://")

	send := func(path string, contentLength int, body []byte, chunked bool) result {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if chunked {
			fmt.Fprintf(conn, "PUT %s HTTP/1.1\r\nHost: x\r\nTransfer-Encoding: chunked\r\n\r\n%x\r\n%s\r\n0\r\n\r\n", path, len(body), body)
		} else {
			fmt.Fprintf(conn, "PUT %s HTTP/1.1\r\nHost: x\r\nContent-Length: %d\r\n\r\n%s", path, contentLength, body)
		}
		if contentLength > len(body) {
			conn.(*net.TCPConn).CloseWrite()
		} else {
			http.ReadResponse(bufio.NewReader(conn), nil)
		}
		out := <-results
		if path == "/old" {
			return out[0]
		}
		return out[1]
	}
	body := bytes.Repeat([]byte("ab"), 100_000)
	for name, tc := range map[string]struct {
		contentLength int
		chunked       bool
	}{"exact": {len(body), false}, "short": {len(body) + 10, false}, "chunked": {0, true}} {
		old := send("/old", tc.contentLength, body, tc.chunked)
		got := send("/new", tc.contentLength, body, tc.chunked)
		if old != got {
			t.Fatalf("%s: old %+v, new %+v", name, old, got)
		}
		if name != "short" && got.n != len(body) {
			t.Fatalf("%s: read %d of %d bytes", name, got.n, len(body))
		}
		if name == "short" && got.err != io.ErrUnexpectedEOF.Error() {
			t.Fatalf("short body: %+v", got)
		}
	}
}

func TestCacheUploadedImageKeepsExactBuffer(t *testing.T) {
	imageDataCache = newBoundedCache(imageDataCacheMaxBytes, func(v []byte) int { return len(v) })
	exact := make([]byte, 100, 100+uploadSlackBytes)
	cacheUploadedImage("exact", exact)
	if got, _ := imageDataCache.get("exact"); &got[0] != &exact[0] {
		t.Fatal("exact-size buffer was copied")
	}
	slack := make([]byte, 100, 100+uploadSlackBytes+1)
	cacheUploadedImage("slack", slack)
	if got, _ := imageDataCache.get("slack"); &got[0] == &slack[0] || cap(got)-len(got) > uploadSlackBytes {
		t.Fatalf("buffer with slack was kept (cap %d)", cap(got))
	}
}

func benchmarkUpload(b *testing.B, read func(*echo.Context) (createPostRequest, []byte, error)) {
	img := bytes.Repeat([]byte{0xff}, 1<<20)
	body, ct := multipartBody(b, []uploadPart{{"caption", "", "c"}, {"visibility", "", "public"}, {"image", "a.jpg", string(img)}})
	b.SetBytes(int64(body.Len()))
	b.ReportAllocs()
	for b.Loop() {
		c := postContext(body, ct)
		if _, data, err := read(c); err != nil || len(data) != len(img) {
			b.Fatal(err, len(data))
		}
	}
}

func BenchmarkReadMultipartUploadOld(b *testing.B) { benchmarkUpload(b, readCreatePostOld) }
func BenchmarkReadMultipartUploadNew(b *testing.B) { benchmarkUpload(b, readCreatePostNew) }

func benchmarkBody(b *testing.B, read func(*http.Request) ([]byte, error)) {
	img := bytes.Repeat([]byte{0xff}, 1<<20)
	b.SetBytes(int64(len(img)))
	b.ReportAllocs()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPut, "/api/me/profile-image", bytes.NewReader(img))
		req.ContentLength = int64(len(img))
		if data, err := read(req); err != nil || len(data) != len(img) {
			b.Fatal(err, len(data))
		}
	}
}

func BenchmarkReadBodyOld(b *testing.B) {
	benchmarkBody(b, func(r *http.Request) ([]byte, error) { return io.ReadAll(r.Body) })
}
func BenchmarkReadBodyNew(b *testing.B) { benchmarkBody(b, readBodyExact) }
