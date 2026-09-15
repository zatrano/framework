package routing_test

import (
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

func TestFromRequestHeaders(t *testing.T) {
	raw := httptest.NewRequest("GET", "/api/v1/ping", nil)
	raw.Header.Set("X-API-Version", "v2")
	req := http.NewRequest(raw)
	if got := routing.FromRequest(req); got != "v2" {
		t.Fatalf("expected v2, got %s", got)
	}

	raw2 := httptest.NewRequest("GET", "/", nil)
	raw2.Header.Set("Accept", "application/vnd.zatrano.v1+json")
	req2 := http.NewRequest(raw2)
	if got := routing.FromRequest(req2); got != "v1" {
		t.Fatalf("expected v1, got %s", got)
	}
}

func TestVersionMountsPrefix(t *testing.T) {
	r := routing.New()
	routing.Version(r, "v1", func(api *routing.Router) {
		api.Get("/ping", func(req *http.Request) *http.Response {
			return http.Text("ok")
		})
	})
	raw := httptest.NewRequest("GET", "/api/v1/ping", nil)
	resp := r.Dispatch(http.NewRequest(raw))
	if resp == nil || resp.StatusCode() != 200 {
		t.Fatalf("status=%v", resp)
	}
	if resp.GetHeader(routing.HeaderVersion) != "v1" {
		t.Fatalf("version header=%q", resp.GetHeader(routing.HeaderVersion))
	}
}

func TestRequireVersion(t *testing.T) {
	h := routing.RequireVersion("v1")(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	raw := httptest.NewRequest("GET", "/", nil)
	raw.Header.Set("X-API-Version", "v2")
	resp := h(http.NewRequest(raw))
	if resp == nil || resp.StatusCode() != 406 {
		t.Fatalf("status=%v", resp)
	}
}
