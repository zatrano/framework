package middleware_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/middleware"
)

func TestAllowIP(t *testing.T) {
	mw := middleware.AllowIP("127.0.0.1", "10.0.0.0/8")
	handler := mw(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})

	okReq := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	okReq.RemoteAddr = "127.0.0.1:1234"
	if handler(http.NewRequest(okReq)).StatusCode() != 200 {
		t.Fatal("expected allow")
	}

	denyReq := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	denyReq.RemoteAddr = "8.8.8.8:1234"
	if handler(http.NewRequest(denyReq)).StatusCode() != 403 {
		t.Fatal("expected deny")
	}
}

func TestDenyIP(t *testing.T) {
	mw := middleware.DenyIP("192.168.1.0/24")
	handler := mw(func(req *http.Request) *http.Response {
		return http.NoContent()
	})
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.1.10:9"
	if handler(http.NewRequest(r)).StatusCode() != 403 {
		t.Fatal("expected deny")
	}
}
