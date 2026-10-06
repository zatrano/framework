package http

import "testing"

func TestRequestPoolResetDropsFields(t *testing.T) {
	req := testRequest("GET", "/keep?q=1", []byte("body"))
	req.Set("k", "v")
	req.SetHeader("X-Trace", "abc")
	path := req.Path()
	ReleaseRequest(req)
	if req.ctx != nil || req.attrs != nil || req.headerOverlay != nil || req.bodyOverride != nil {
		t.Fatal("request still holds data after release")
	}
	if path != "/keep" {
		t.Fatalf("path copy=%q", path)
	}
	next := NewRequest(nil)
	if next.Get("k") != nil || next.Path() != "" {
		t.Fatalf("pooled request leaked: path=%q attr=%v", next.Path(), next.Get("k"))
	}
	ReleaseRequest(next)
}

func TestPathAndQueryAreCopies(t *testing.T) {
	req := testRequest("GET", "/docs?q=ab", nil)
	path := req.Path()
	query := req.QueryString()
	value, ok := req.HeaderValue("X-Trace")
	req.ctx.Path[1] = 'X'
	req.ctx.Query[0] = 'Z'
	if path != "/docs" {
		t.Fatalf("path aliased the buffer: %q", path)
	}
	if query != "q=ab" {
		t.Fatalf("query aliased the buffer: %q", query)
	}
	if ok || value != "" {
		t.Fatalf("missing header value=%q ok=%v", value, ok)
	}
	if req.HeaderCacheBuilt() {
		t.Fatal("HeaderValue built the map")
	}
}
