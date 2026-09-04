package http_test

import (
	"crypto/tls"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/http"
)

func TestURLHelpers(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodGet, "http://example.test/path?x=1", nil)
	raw.Host = "example.test"
	raw.Header.Set("X-Requested-With", "XMLHttpRequest")
	req := http.NewRequest(raw)

	if req.Host() != "example.test" || req.Scheme() != "http" || req.Secure() {
		t.Fatalf("host/scheme=%s/%s secure=%v", req.Host(), req.Scheme(), req.Secure())
	}
	if req.Root() != "http://example.test" {
		t.Fatalf("root=%s", req.Root())
	}
	if req.FullURL() != "http://example.test/path?x=1" {
		t.Fatalf("full=%s", req.FullURL())
	}
	if !req.Ajax() {
		t.Fatal("expected ajax")
	}

	req.Set("_forwarded_proto", "https")
	req.Set("_forwarded_host", "proxy.example.test")
	if !req.Secure() || req.Scheme() != "https" || req.Host() != "proxy.example.test" {
		t.Fatalf("forwarded secure/host failed: %s %s", req.Scheme(), req.Host())
	}

	tlsReq := httptest.NewRequest(stdhttp.MethodGet, "https://secure.test/", nil)
	tlsReq.TLS = &tls.ConnectionState{}
	tlsReq.Host = "secure.test"
	secure := http.NewRequest(tlsReq)
	if !secure.Secure() || secure.Scheme() != "https" {
		t.Fatal("tls secure")
	}
}

type memSession struct {
	data map[string]any
}

func (s *memSession) Get(key string, fallback ...any) any {
	if v, ok := s.data[key]; ok {
		return v
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return nil
}
func (s *memSession) Put(key string, value any)   { s.data[key] = value }
func (s *memSession) Flash(key string, value any) { s.data[key] = value }
func (s *memSession) Pull(key string, fallback ...any) any {
	v := s.Get(key, fallback...)
	delete(s.data, key)
	return v
}
func (s *memSession) Forget(key string) { delete(s.data, key) }
func (s *memSession) Regenerate() error { return nil }
func (s *memSession) ID() string        { return "test" }

func TestRequestIdentityHelpers(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "http://example.test/path?tag=a&tag=b&q=1", nil)
	raw.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0")
	req := http.NewRequest(raw)

	if !req.IsMethod("POST", "PUT") || req.IsMethod("GET") {
		t.Fatal("IsMethod")
	}
	queries := req.Queries()
	if queries["q"] != "1" || queries["tag"] != "a" {
		t.Fatalf("queries=%v", queries)
	}
	all := req.QueryAll()
	if len(all["tag"]) != 2 {
		t.Fatalf("query_all=%v", all)
	}
	if req.UserAgent() == "" || req.Agent().Browser != "Chrome" || req.Agent().Platform != "Windows" {
		t.Fatalf("agent=%+v", req.Agent())
	}

	sess := &memSession{data: map[string]any{
		"_old_input": map[string]string{"name": "Ada"},
	}}
	req.SetSession(sess)
	if req.Old("name") != "Ada" || req.Old("missing", "fallback") != "fallback" {
		t.Fatalf("old=%q missing=%q", req.Old("name"), req.Old("missing", "fallback"))
	}
}

func TestRequestPathAndRouteHelpers(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodGet, "http://example.test/api/demo/request/path", nil)
	req := http.NewRequest(raw)
	req.SetRouteName("api.demo.request.path")

	if !req.ExactPath("/api/demo/request/path") || req.ExactPath("/api/demo") {
		t.Fatal("exact path")
	}
	if !req.PathIs("/api/*", "/other") || req.PathIs("/web/*") {
		t.Fatal("path is")
	}
	segs := req.Segments()
	if len(segs) != 4 || segs[0] != "api" || req.Segment(2) != "demo" || req.Segment(9, "x") != "x" {
		t.Fatalf("segments=%v", segs)
	}
	if !req.RouteIs("api.demo.*") || req.RouteIs("web.*") || req.RouteName() != "api.demo.request.path" {
		t.Fatal("route is/name")
	}
}

func TestRequestAcceptAndPjaxHelpers(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodGet, "http://example.test/api", nil)
	raw.Header.Set("Accept", "text/html, application/json;q=0.9")
	req := http.NewRequest(raw)

	if !req.Accepts("html", "json") || !req.AcceptsHtml() || !req.AcceptsJSON() {
		t.Fatal("accepts html/json")
	}
	if req.Prefers("json", "html") != "html" {
		t.Fatalf("prefers=%q", req.Prefers("json", "html"))
	}
	if req.Prefers("xml") != "" || req.Accepts("xml") {
		t.Fatal("should not accept xml")
	}

	jsonOnly := httptest.NewRequest(stdhttp.MethodGet, "http://example.test/api", nil)
	jsonOnly.Header.Set("Accept", "application/json")
	jsonReq := http.NewRequest(jsonOnly)
	if !jsonReq.AcceptsJSON() || jsonReq.AcceptsHtml() || !jsonReq.ExpectsJSON() {
		t.Fatal("json expects")
	}

	ajax := httptest.NewRequest(stdhttp.MethodGet, "http://example.test/api", nil)
	ajax.Header.Set("X-Requested-With", "XMLHttpRequest")
	ajax.Header.Set("Accept", "application/json")
	ajaxReq := http.NewRequest(ajax)
	if !ajaxReq.ExpectsJSON() || ajaxReq.Pjax() {
		t.Fatal("ajax expects json")
	}

	pjax := httptest.NewRequest(stdhttp.MethodGet, "http://example.test/api", nil)
	pjax.Header.Set("X-Requested-With", "XMLHttpRequest")
	pjax.Header.Set("X-PJAX", "true")
	pjax.Header.Set("Accept", "application/json")
	pjaxReq := http.NewRequest(pjax)
	if !pjaxReq.Pjax() {
		t.Fatal("expected pjax")
	}
	// WantsJSON still true via Accept; ExpectsJSON true via WantsJSON path
	if !pjaxReq.ExpectsJSON() {
		t.Fatal("pjax with json accept still expects json via WantsJSON")
	}

	star := httptest.NewRequest(stdhttp.MethodGet, "http://example.test/api", nil)
	star.Header.Set("Accept", "*/*")
	starReq := http.NewRequest(star)
	if starReq.Prefers("html", "json") != "html" {
		t.Fatalf("star prefers=%q", starReq.Prefers("html", "json"))
	}
}
