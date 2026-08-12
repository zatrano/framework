package ratelimit_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/ratelimit"
)

func TestNamedLimiter(t *testing.T) {
	limiter := ratelimit.New()
	limiter.For("demo", ratelimit.Limit{MaxAttempts: 2, Decay: time.Minute})
	if !limiter.Has("demo") {
		t.Fatal("expected named limit")
	}

	mw := limiter.Named("demo")
	handler := mw(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})

	call := func() int {
		r := httptest.NewRequest(stdhttp.MethodGet, "/x", nil)
		r.RemoteAddr = "1.2.3.4:1234"
		return handler(http.NewRequest(r)).StatusCode()
	}

	first := call()
	second := call()
	if first != 200 || second != 200 {
		t.Fatal("first two should pass")
	}
	if call() != 429 {
		t.Fatal("expected 429")
	}
}
