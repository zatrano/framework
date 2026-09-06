package middleware_test

import (
	crand "crypto/rand"
	"errors"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel/encryption"
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/middleware"
)

type nthFailRand struct {
	real io.Reader
	n    int
	fail int
}

func (r *nthFailRand) Read(p []byte) (int, error) {
	r.n++
	if r.fail > 0 && r.n == r.fail {
		return 0, errors.New("entropy exhausted")
	}
	return io.ReadFull(r.real, p)
}

func withFailingRand(t *testing.T, failOn int) {
	t.Helper()
	old := crand.Reader
	t.Cleanup(func() { crand.Reader = old })
	crand.Reader = &nthFailRand{real: old, fail: failOn}
}

func cookieByName(cookies []*stdhttp.Cookie, name string) *stdhttp.Cookie {
	for _, c := range cookies {
		if c != nil && c.Name == name {
			return c
		}
	}
	return nil
}

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

func TestEncryptCookiesDropsPlaintextOnEncryptError(t *testing.T) {
	enc, err := encryption.New(strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	withFailingRand(t, 1)
	mw := middleware.EncryptCookies(enc, "secret")
	resp := mw(func(req *http.Request) *http.Response {
		return http.Text("ok").WithCookie(&stdhttp.Cookie{Name: "secret", Value: "plain-secret", Path: "/"})
	})(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil)))

	c := cookieByName(resp.Cookies(), "secret")
	if c != nil && c.Value == "plain-secret" {
		t.Fatal("encrypt failure sent plaintext cookie")
	}
	if c != nil && !strings.HasPrefix(c.Value, "ZATRANO:") {
		t.Fatalf("encrypt failure must not send cookie %q", c.Value)
	}
}

func TestEncryptCookiesPartialEncryptFailureKeepsSuccess(t *testing.T) {
	enc, err := encryption.New(strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	withFailingRand(t, 2)
	mw := middleware.EncryptCookies(enc, "a", "b", "c")
	resp := mw(func(req *http.Request) *http.Response {
		return http.Text("ok").
			WithCookie(&stdhttp.Cookie{Name: "a", Value: "va", Path: "/"}).
			WithCookie(&stdhttp.Cookie{Name: "b", Value: "vb", Path: "/"}).
			WithCookie(&stdhttp.Cookie{Name: "c", Value: "vc", Path: "/"})
	})(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil)))

	a := cookieByName(resp.Cookies(), "a")
	b := cookieByName(resp.Cookies(), "b")
	c := cookieByName(resp.Cookies(), "c")
	if a == nil || !strings.HasPrefix(a.Value, "ZATRANO:") {
		t.Fatalf("cookie a lost: %+v", a)
	}
	if b != nil {
		t.Fatalf("failed cookie b must be dropped, not sent: %+v", b)
	}
	if c == nil || !strings.HasPrefix(c.Value, "ZATRANO:") {
		t.Fatalf("cookie c lost: %+v", c)
	}
}

func TestEncryptCookiesDecryptFailureKeepsCiphertext(t *testing.T) {
	enc, err := encryption.New(strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	mw := middleware.EncryptCookies(enc, "secret")
	raw := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.AddCookie(&stdhttp.Cookie{Name: "secret", Value: "ZATRANO:not-valid-ciphertext"})
	var seen string
	mw(func(req *http.Request) *http.Response {
		seen = req.Cookie("secret")
		return http.Text("ok")
	})(http.NewRequest(raw))
	if seen != "ZATRANO:not-valid-ciphertext" {
		t.Fatalf("decrypt failure must leave ciphertext, got %q", seen)
	}
}

func TestEncryptCookiesEmptyValueStillEncrypts(t *testing.T) {
	enc, err := encryption.New(strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	mw := middleware.EncryptCookies(enc, "secret")
	resp := mw(func(req *http.Request) *http.Response {
		return http.Text("ok").WithCookie(&stdhttp.Cookie{Name: "secret", Value: "", Path: "/"})
	})(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil)))
	c := cookieByName(resp.Cookies(), "secret")
	if c == nil || !strings.HasPrefix(c.Value, "ZATRANO:") {
		t.Fatalf("empty value cookie=%+v", c)
	}
}

func TestEncryptCookiesLeavesUnlistedCookies(t *testing.T) {
	enc, err := encryption.New(strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	mw := middleware.EncryptCookies(enc, "secret")
	resp := mw(func(req *http.Request) *http.Response {
		return http.Text("ok").
			WithCookie(&stdhttp.Cookie{Name: "secret", Value: "plain", Path: "/"}).
			WithCookie(&stdhttp.Cookie{Name: "theme", Value: "dark", Path: "/"})
	})(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil)))
	theme := cookieByName(resp.Cookies(), "theme")
	if theme == nil || theme.Value != "dark" {
		t.Fatalf("unlisted cookie mutated: %+v", theme)
	}
	secret := cookieByName(resp.Cookies(), "secret")
	if secret == nil || !strings.HasPrefix(secret.Value, "ZATRANO:") {
		t.Fatalf("listed cookie=%+v", secret)
	}
}
