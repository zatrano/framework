package http_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
)

func TestJSONBytesMatchEncodingJSON(t *testing.T) {
	samples := []any{
		map[string]string{"message": "Hello, World!"},
		map[string]string{"html": "<tag>&\"'"},
		map[string]string{"b": "2", "a": "1"},
		map[string]string{"ü": "ok"},
		map[string]any{"message": "Hello, World!"},
		map[string]any{"html": "<tag>&\"'"},
		map[string]any{"n": 1, "ok": true, "s": "a/b"},
		[]any{"<", ">"},
	}
	for _, sample := range samples {
		want, err := json.Marshal(sample)
		if err != nil {
			t.Fatal(err)
		}
		got := http.JSON(sample).Content()
		if string(got) != string(want) {
			t.Fatalf("JSON %s want %s", got, want)
		}
	}
}

// The deprecated pool API must never hand a request or response back out.
func TestDeprecatedPoolAPINeverReuses(t *testing.T) {
	http.SetRequestPooling(true, true)
	if http.RequestPooling() {
		t.Fatal("SetRequestPooling(true) turned pooling on")
	}
	seen := map[*http.Request]bool{}
	for i := 0; i < 256; i++ {
		req := http.AcquireRequest(nil)
		if seen[req] {
			t.Fatalf("acquire %d returned a released request", i)
		}
		seen[req] = true
		http.ReleaseRequest(req)
	}
}

func TestReleasedRequestAndResponseStayReadable(t *testing.T) {
	req := http.RequestFromHTTP(httptest.NewRequest("POST", "/orders/7?q=1", strings.NewReader("payload")))
	req.Set("user", 42)
	resp := http.JSON(map[string]string{"ok": "1"})
	http.ReleaseAfterCommit(req, resp)
	http.ReleaseRequest(req)
	http.ReleaseResponse(resp)
	if req.Method() != "POST" || req.Path() != "/orders/7" || req.Query("q") != "1" {
		t.Fatalf("released request lost its values: %s %s q=%q", req.Method(), req.Path(), req.Query("q"))
	}
	body, err := req.Body()
	if err != nil || string(body) != "payload" || req.Get("user") != 42 {
		t.Fatalf("released request lost body or attributes")
	}
	if string(resp.Content()) != `{"ok":"1"}` || resp.StatusCode() != 200 {
		t.Fatalf("released response changed: %d %s", resp.StatusCode(), resp.Content())
	}
	if req.Generation() != 0 {
		t.Fatal("Generation is no longer meaningful and must stay 0")
	}
}

func BenchmarkNewRequest(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = http.NewRequest(nil)
	}
}
