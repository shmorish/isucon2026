package isureal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

// isureal-guard は info レベルでも次の 3 種類の行を読んで BAN する
func TestActionLogInfoKeepsGuardLines(t *testing.T) {
	var out bytes.Buffer
	e := echo.New()
	e.IPExtractor = extractClientIP
	e.Use(ActionLogMiddleware("info", &out))
	e.HTTPErrorHandler = APIErrorHandler
	e.POST("/api/auth/login", func(c *echo.Context) error {
		return NewAPIError(http.StatusUnauthorized, "invalid_credentials")
	})
	e.PUT("/api/me/profile-image", func(c *echo.Context) error {
		c.Set("abuse", "invalid_image")
		return NewAPIError(http.StatusBadRequest, "invalid_profile_image")
	})
	e.GET("/api/images/:imageId", func(c *echo.Context) error {
		if c.QueryParam("sig") == "bad" {
			c.Set("abuse", "invalid_signature")
		}
		return NewAPIError(http.StatusForbidden, "image_forbidden")
	})
	e.GET("/api/me", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{})
	})
	e.POST("/api/friend-requests", func(c *echo.Context) error {
		return c.JSON(http.StatusCreated, map[string]string{})
	})

	requests := []struct {
		method, target string
	}{
		{http.MethodPost, "/api/auth/login"},
		{http.MethodPut, "/api/me/profile-image"},
		{http.MethodGet, "/api/images/x?sig=bad"},
		{http.MethodGet, "/api/images/x"},
		{http.MethodGet, "/api/me"},
		{http.MethodPost, "/api/friend-requests"},
	}
	for _, r := range requests {
		req := httptest.NewRequest(r.method, r.target, nil)
		req.Header.Set("X-Client-IP", "203.0.113.7")
		e.ServeHTTP(httptest.NewRecorder(), req)
	}

	type line struct {
		Level  string  `json:"level"`
		Action *string `json:"action"`
		Status int     `json:"status"`
		IP     string  `json:"ip"`
		Abuse  string  `json:"abuse"`
	}
	var got []line
	for _, raw := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var l line
		if err := json.Unmarshal([]byte(raw), &l); err != nil {
			t.Fatalf("invalid log line %q: %v", raw, err)
		}
		got = append(got, l)
	}
	want := []struct {
		action string
		status int
		abuse  string
	}{
		{"login", http.StatusUnauthorized, ""},
		{"put_profile_image", http.StatusBadRequest, "invalid_image"},
		{"get_image", http.StatusForbidden, "invalid_signature"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %s", len(got), len(want), out.String())
	}
	for i, w := range want {
		g := got[i]
		if g.Action == nil || *g.Action != w.action || g.Status != w.status || g.Abuse != w.abuse || g.IP != "203.0.113.7" || g.Level != "info" {
			t.Errorf("line %d = %+v, want action=%s status=%d abuse=%q", i, g, w.action, w.status, w.abuse)
		}
	}
}
