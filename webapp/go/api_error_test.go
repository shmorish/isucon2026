package isureal

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

// 本文を読まずにエラーを返しても接続を閉じない（閉じると送信中の nginx が 502 にする）
func TestAPIErrorKeepsConnectionWithUnreadBody(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = APIErrorHandler
	e.PUT("/api/me/profile-image", func(c *echo.Context) error {
		return NewAPIError(http.StatusForbidden, "ip_banned")
	})
	server := httptest.NewServer(e)
	defer server.Close()

	req, _ := http.NewRequest(http.MethodPut, server.URL+"/api/me/profile-image", bytes.NewReader(make([]byte, 4<<20)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden || resp.Close {
		t.Fatalf("status=%d close=%v", resp.StatusCode, resp.Close)
	}
}
