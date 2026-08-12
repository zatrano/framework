package api_test

import (
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/api"
	"github.com/zatrano/framework/packages/http"
)

func TestFromRequestHeaders(t *testing.T) {
	raw := httptest.NewRequest("GET", "/api/v1/ping", nil)
	raw.Header.Set("X-API-Version", "v2")
	req := http.NewRequest(raw)
	if got := api.FromRequest(req); got != "v2" {
		t.Fatalf("expected v2, got %s", got)
	}

	raw2 := httptest.NewRequest("GET", "/", nil)
	raw2.Header.Set("Accept", "application/vnd.zatrano.v1+json")
	req2 := http.NewRequest(raw2)
	if got := api.FromRequest(req2); got != "v1" {
		t.Fatalf("expected v1, got %s", got)
	}
}
