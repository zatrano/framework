package http

import (
	stdhttp "net/http"
	"strings"
	"testing"
)

// Regression tests for findings in docs/REVIEW-LOG.md.

func TestRedirectBackRejectsBackslashPaths(t *testing.T) {
	for _, ref := range []string{
		`/\evil.example`,
		`https://app.example/\evil.example`,
		`https://app.example/a\b`,
	} {
		raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "https://app.example/form", nil)
		raw.Header.Set("Referer", ref)
		resp := RedirectBack(RequestFromHTTP(raw), "/home")
		if got := resp.RedirectURL(); got != "/home" {
			t.Fatalf("Referer %q redirected to %q, want the fallback", ref, got)
		}
		if got := resp.GetHeader("Location"); got != "/home" {
			t.Fatalf("Referer %q Location %q, want the fallback", ref, got)
		}
	}
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "https://app.example/form", nil)
	raw.Header.Set("Referer", "https://app.example/dashboard?tab=1")
	resp := RedirectBack(RequestFromHTTP(raw), "/home")
	if got := resp.RedirectURL(); got != "/dashboard?tab=1" {
		t.Fatalf("same-origin Referer redirected to %q", got)
	}
}

func TestContentDispositionCannotBreakQuoting(t *testing.T) {
	for _, name := range []string{`report\`, `a"; filename*=UTF-8''evil.exe`, "x\x01y", `a\"b`} {
		got := contentDisposition("attachment", name)
		inner := strings.TrimSuffix(strings.TrimPrefix(got, `attachment; filename="`), `"`)
		if strings.ContainsAny(inner, "\"\\\x01") {
			t.Fatalf("filename %q produced %q", name, got)
		}
	}
}

// Content returns a copy, so a middleware cannot write into the body that a
// cached response shares with every request.
func TestContentReturnsACopyOfSharedBody(t *testing.T) {
	cached := Bytes([]byte("SECRET"), "text/plain")
	req := NewRequest(nil)
	own := req.Own(cached)
	own.Content()[0] = 'X'
	if got := string(cached.Content()); got != "SECRET" {
		t.Fatalf("shared body changed to %q", got)
	}
	if got := string(own.Content()); got != "SECRET" {
		t.Fatalf("owned copy body changed to %q", got)
	}
}

func TestHeaderValuesReturnsEveryLine(t *testing.T) {
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.Header.Add("Origin", "https://evil.example")
	raw.Header.Add("Origin", "https://app.example")
	req := RequestFromHTTP(raw)
	vals := req.HeaderValues("Origin")
	if len(vals) != 2 || vals[0] != "https://evil.example" {
		t.Fatalf("HeaderValues=%v", vals)
	}
	vals[0] = "changed"
	if req.HeaderValues("Origin")[0] != "https://evil.example" {
		t.Fatal("HeaderValues returned the overlay slice")
	}
}
