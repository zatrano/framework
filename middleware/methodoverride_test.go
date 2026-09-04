package middleware_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/http"
	"github.com/zatrano/framework/middleware"
)

func TestMethodOverride(t *testing.T) {
	var seen string
	handler := middleware.MethodOverride(func(req *http.Request) *http.Response {
		seen = req.Method()
		return http.Text("ok")
	})
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader("_method=DELETE"))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = handler(http.NewRequest(raw))
	if seen != stdhttp.MethodDelete {
		t.Fatalf("method=%q", seen)
	}

	headerReq := httptest.NewRequest(stdhttp.MethodPost, "/", nil)
	headerReq.Header.Set("X-HTTP-Method-Override", "PATCH")
	req := http.NewRequest(headerReq)
	middleware.ApplyMethodOverride(req)
	if req.Method() != stdhttp.MethodPatch {
		t.Fatalf("header override=%q", req.Method())
	}
}
