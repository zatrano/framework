package middleware_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/encryption"
	"github.com/zatrano/framework/http"
	"github.com/zatrano/framework/middleware"
)

func TestEncryptCookies(t *testing.T) {
	enc, err := encryption.New(strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	mw := middleware.EncryptCookies(enc, "secret")
	handler := mw(func(req *http.Request) *http.Response {
		return http.Text("ok").WithCookie(&stdhttp.Cookie{Name: "secret", Value: "plain-value", Path: "/"})
	})

	raw := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req := http.NewRequest(raw)
	resp := handler(req)
	if resp == nil || len(resp.Cookies()) == 0 {
		t.Fatal("expected cookie")
	}
	value := resp.Cookies()[0].Value
	if !strings.HasPrefix(value, "ZATRANO:") {
		t.Fatalf("expected encrypted cookie, got %q", value)
	}

	// Round-trip decrypt on next request.
	raw2 := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	raw2.AddCookie(&stdhttp.Cookie{Name: "secret", Value: value})
	var seen string
	handler2 := mw(func(req *http.Request) *http.Response {
		seen = req.Cookie("secret")
		return http.Text("ok")
	})
	_ = handler2(http.NewRequest(raw2))
	if seen != "plain-value" {
		t.Fatalf("decrypted=%q", seen)
	}
}
