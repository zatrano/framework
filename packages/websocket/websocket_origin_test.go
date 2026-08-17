package websocket

import (
	stdhttp "net/http"
	"testing"

	"github.com/zatrano/framework/packages/http"
)

func TestSameOriginRejectsCrossSite(t *testing.T) {
	raw := &stdhttp.Request{Host: "app.example.com", Header: stdhttp.Header{}}
	raw.Header.Set("Origin", "https://evil.example")
	req := http.NewRequest(raw)
	if SameOrigin(req) {
		t.Fatal("cross-site Origin must be rejected")
	}
}

func TestSameOriginAllowsMatchingHost(t *testing.T) {
	raw := &stdhttp.Request{Host: "app.example.com", Header: stdhttp.Header{}}
	raw.Header.Set("Origin", "https://app.example.com")
	req := http.NewRequest(raw)
	if !SameOrigin(req) {
		t.Fatal("same-origin must be allowed")
	}
}

func TestSameOriginAllowsMissingOrigin(t *testing.T) {
	raw := &stdhttp.Request{Host: "app.example.com", Header: stdhttp.Header{}}
	req := http.NewRequest(raw)
	if !SameOrigin(req) {
		t.Fatal("missing Origin (non-browser) should be allowed")
	}
}
