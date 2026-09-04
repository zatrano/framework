package routing_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/http"
	"github.com/zatrano/framework/routing"
)

func TestDispatchTrailingSlashNormalized(t *testing.T) {
	r := routing.New()
	r.Get("/dashboard", func(req *http.Request) *http.Response {
		return http.Text("ok:" + req.Path() + "?" + req.QueryString())
	})

	plain := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/dashboard", nil)))
	if plain == nil || plain.StatusCode() != 200 || string(plain.Content()) != "ok:/dashboard?" {
		t.Fatalf("plain path: status=%v body=%q", plain.StatusCode(), plain.Content())
	}

	slash := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/dashboard/?tab=1", nil)))
	if slash == nil || slash.StatusCode() != 200 {
		t.Fatalf("trailing slash should match, status=%v", slash.StatusCode())
	}
	if string(slash.Content()) != "ok:/dashboard?tab=1" {
		t.Fatalf("expected normalized path with query preserved, got %q", slash.Content())
	}
}

func TestDispatchRootSlashUnchanged(t *testing.T) {
	r := routing.New()
	r.Get("/", func(req *http.Request) *http.Response {
		return http.Text("home:" + req.Path())
	})
	resp := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil)))
	if resp == nil || string(resp.Content()) != "home:/" {
		t.Fatalf("root path broken: %v %q", resp.StatusCode(), resp.Content())
	}
}
