package octane_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/octane"
)

func TestOctaneStats(t *testing.T) {
	rt := octane.New(4)
	if rt.Workers() != 4 {
		t.Fatal(rt.Workers())
	}
	mw := rt.Middleware()
	handler := mw(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	resp := handler(http.NewRequest(r))
	if resp.StatusCode() != 200 {
		t.Fatal(resp.StatusCode())
	}
	stats := rt.Stats()
	if stats["requests"].(int64) < 1 {
		t.Fatalf("%v", stats)
	}
}
