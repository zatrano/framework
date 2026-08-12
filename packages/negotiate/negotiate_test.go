package negotiate_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/negotiate"
)

func TestNegotiate(t *testing.T) {
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	r.Header.Set("Accept", "text/html,application/json;q=0.9")
	req := http.NewRequest(r)
	if got := negotiate.Negotiate(req, "json", "html"); got != "html" {
		t.Fatalf("got=%s", got)
	}
	r2 := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	r2.Header.Set("Accept", "application/json")
	if got := negotiate.Negotiate(http.NewRequest(r2), "json", "html"); got != "json" {
		t.Fatalf("got=%s", got)
	}
}
