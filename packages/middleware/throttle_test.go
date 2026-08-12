package middleware_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/middleware"
	"github.com/zatrano/framework/packages/ratelimit"
)

func TestThrottleRequestsWith(t *testing.T) {
	limiter := ratelimit.New()
	handler := middleware.ThrottleRequestsWith(limiter, 1, time.Minute)(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	first := handler(req)
	if first.StatusCode() != 200 {
		t.Fatalf("first=%d", first.StatusCode())
	}
	second := handler(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil)))
	if second.StatusCode() != 429 {
		t.Fatalf("second=%d", second.StatusCode())
	}
}
