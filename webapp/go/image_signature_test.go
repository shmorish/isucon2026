package isureal

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSignedImageURLExpNotShorterThanLifetime(t *testing.T) {
	min := time.Now().Add(imageURLLifetime).Unix()
	u, err := url.Parse(signedImageURL(User{ImageKey: make([]byte, 32)}, "img"))
	if err != nil {
		t.Fatal(err)
	}
	exp, err := strconv.ParseInt(u.Query().Get("exp"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	if exp < min || exp > min+int64(imageExpStep/time.Second) || exp%int64(imageExpStep/time.Second) != 0 {
		t.Fatalf("exp=%d min=%d", exp, min)
	}
	if !strings.HasPrefix(u.Path, "/api/images/img") {
		t.Fatalf("path=%s", u.Path)
	}
}
