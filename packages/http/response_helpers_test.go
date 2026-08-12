package http_test

import (
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/http"
)

func TestResponseHeaderAndCookieHelpers(t *testing.T) {
	resp := http.JSON(map[string]any{"ok": true}).
		WithHeaders(map[string]string{
			"X-Demo":  "1",
			"X-Trace": "abc",
		}).
		Header("X-Temp", "gone").
		WithoutHeader("X-Temp").
		Cookie("session", "s1", 3600).
		WithoutCookie("old")

	if resp.Headers().Get("X-Demo") != "1" || resp.Headers().Get("X-Trace") != "abc" {
		t.Fatalf("headers=%v", resp.Headers())
	}
	if resp.Headers().Get("X-Temp") != "" {
		t.Fatal("WithoutHeader failed")
	}
	cookies := resp.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies=%d", len(cookies))
	}
	if cookies[0].Name != "session" || cookies[0].Value != "s1" || cookies[0].MaxAge != 3600 {
		t.Fatalf("cookie=%+v", cookies[0])
	}
	if cookies[1].Name != "old" || cookies[1].MaxAge != -1 {
		t.Fatalf("without cookie=%+v", cookies[1])
	}
}

func TestResponseCacheHelpers(t *testing.T) {
	noCache := http.JSON(map[string]any{"ok": true}).NoCache()
	if noCache.Headers().Get("Cache-Control") != "no-cache, no-store, must-revalidate" ||
		noCache.Headers().Get("Pragma") != "no-cache" ||
		noCache.Headers().Get("Expires") != "0" {
		t.Fatalf("no cache=%v", noCache.Headers())
	}

	cached := http.Text("ok").CacheFor(time.Hour).Vary("Accept", "Accept-Language")
	if cached.Headers().Get("Cache-Control") != "public, max-age=3600" {
		t.Fatalf("cache for=%q", cached.Headers().Get("Cache-Control"))
	}
	if cached.Headers().Get("Vary") != "Accept, Accept-Language" {
		t.Fatalf("vary=%q", cached.Headers().Get("Vary"))
	}

	private := http.Text("secret").PrivateCache(30 * time.Second)
	if private.Headers().Get("Cache-Control") != "private, max-age=30" {
		t.Fatalf("private cache=%q", private.Headers().Get("Cache-Control"))
	}
}

func TestResponseStatusHelpers(t *testing.T) {
	if http.Created(map[string]any{"id": 1}).StatusCode() != 201 {
		t.Fatal("created")
	}
	if http.Accepted(map[string]any{"queued": true}).StatusCode() != 202 {
		t.Fatal("accepted")
	}
	bad := http.BadRequest("invalid")
	if bad.StatusCode() != 400 || !strings.Contains(string(bad.Content()), "invalid") {
		t.Fatalf("bad request=%s", bad.Content())
	}
	if http.Unauthorized().StatusCode() != 401 {
		t.Fatal("unauthorized")
	}
	if http.NotFound("missing").StatusCode() != 404 {
		t.Fatal("not found")
	}
	if http.Forbidden("denied").StatusCode() != 403 {
		t.Fatal("forbidden")
	}
	if http.Conflict("exists").StatusCode() != 409 {
		t.Fatal("conflict")
	}
	if http.Gone("removed").StatusCode() != 410 {
		t.Fatal("gone")
	}
	unp := http.Unprocessable("invalid payload")
	if unp.StatusCode() != 422 || !strings.Contains(string(unp.Content()), "invalid payload") {
		t.Fatalf("unprocessable=%s", unp.Content())
	}
	if http.MethodNotAllowed("POST only").StatusCode() != 405 {
		t.Fatal("method not allowed")
	}
	if http.PaymentRequired("subscribe").StatusCode() != 402 {
		t.Fatal("payment required")
	}
	if http.TooManyRequests("slow down").StatusCode() != 429 {
		t.Fatal("too many requests")
	}
	if http.ServiceUnavailable("maintenance").StatusCode() != 503 {
		t.Fatal("service unavailable")
	}
}

func TestResponseCompletionHelpers(t *testing.T) {
	text := http.Text("hello").Charset("utf-8").
		ETag(`"v1"`).
		LastModified(time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)).
		ExpiresAt(time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)).
		MaxAge(120).
		SharedMaxAge(60).
		MustRevalidate().
		Immutable().
		Public().
		StaleWhileRevalidate(30).
		StaleIfError(10).
		Allow("GET", "HEAD").
		ContentLanguage("tr").
		AppendHeader("X-Demo", "a").
		AppendHeader("X-Demo", "b").
		CookieMinutes("mins", "1", 5).
		CookieForever("forever", "1").
		SecureCookie("sec", "1", 60)

	if !strings.Contains(text.ContentType(), "charset=utf-8") {
		t.Fatalf("charset=%q", text.ContentType())
	}
	if !text.HasHeader("ETag") || text.GetHeader("ETag") != `"v1"` {
		t.Fatalf("etag=%q", text.GetHeader("ETag"))
	}
	if text.GetHeader("Allow") != "GET, HEAD" {
		t.Fatalf("allow=%q", text.GetHeader("Allow"))
	}
	if text.GetHeader("Content-Language") != "tr" {
		t.Fatal("content language")
	}
	cc := text.GetHeader("Cache-Control")
	for _, part := range []string{"public", "max-age=120", "s-maxage=60", "must-revalidate", "immutable", "stale-while-revalidate=30", "stale-if-error=10"} {
		if !strings.Contains(cc, part) {
			t.Fatalf("cache-control missing %q in %q", part, cc)
		}
	}
	if got := text.Headers().Values("X-Demo"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("append header=%v", got)
	}
	if len(text.Cookies()) != 3 {
		t.Fatalf("cookies=%d", len(text.Cookies()))
	}
	if !text.IsSuccessful() || !text.IsOk() || !text.IsText() || text.Failed() {
		t.Fatal("text predicates")
	}

	text.WithoutHeaders("ETag", "Allow")
	if text.HasHeader("ETag") || text.HasHeader("Allow") {
		t.Fatal("WithoutHeaders failed")
	}

	ok := http.OK(map[string]any{"ok": true})
	if !ok.IsJSON() || ok.ContentLength() == 0 {
		t.Fatal("OK/json")
	}
	if !http.HTML("<p>hi</p>").IsHTML() {
		t.Fatal("IsHTML")
	}
	if !http.Empty().IsEmpty() || http.Empty().StatusCode() != 204 {
		t.Fatal("Empty")
	}
	if http.NotModified().StatusCode() != 304 || !http.NotModified().IsEmpty() {
		t.Fatal("NotModified")
	}
	if http.InternalServerError("boom").StatusCode() != 500 || !http.InternalServerError("boom").IsServerError() {
		t.Fatal("InternalServerError")
	}
	if !http.NotFound().IsNotFound() || !http.NotFound().IsClientError() || !http.NotFound().Failed() {
		t.Fatal("NotFound predicates")
	}
	if !http.Forbidden().IsForbidden() || !http.Unauthorized().IsUnauthorized() {
		t.Fatal("auth predicates")
	}
	xml := http.XML("<ok/>")
	if !strings.Contains(xml.ContentType(), "xml") || string(xml.Content()) != "<ok/>" {
		t.Fatalf("xml=%s %q", xml.ContentType(), xml.Content())
	}
	jp := http.JSONP("cb", map[string]any{"n": 1})
	if !strings.Contains(string(jp.Content()), "cb({") || !strings.Contains(jp.ContentType(), "javascript") {
		t.Fatalf("jsonp=%s", jp.Content())
	}
	if http.Bytes([]byte("ab"), "").ContentLength() != 2 {
		t.Fatal("Bytes")
	}
	if http.Make(418, []byte("teapot"), "text/plain").StatusCode() != 418 {
		t.Fatal("Make")
	}
	if http.PartialContent([]byte("x"), "text/plain").StatusCode() != 206 {
		t.Fatal("PartialContent")
	}
	if !http.Found("/home").IsRedirection() || http.Found("/home").RedirectURL() != "/home" {
		t.Fatal("Found")
	}
	if http.SeeOther("/x").StatusCode() != 303 {
		t.Fatal("SeeOther")
	}
	if http.TemporaryRedirect("/y").StatusCode() != 307 {
		t.Fatal("TemporaryRedirect")
	}

	opts := http.Text("c").WithCookieOptions("n", "v", http.CookieOptions{
		MaxAge: 10, Path: "/app", Secure: true, HTTPOnly: true,
	})
	c := opts.Cookies()[0]
	if c.Name != "n" || c.Value != "v" || c.Path != "/app" || !c.Secure || !c.HttpOnly || c.MaxAge != 10 {
		t.Fatalf("cookie options=%+v", c)
	}
}
