package http_test

import (
	"encoding/json"
	"net"
	stdhttp "net/http"
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

func TestPooledRequestPanicsAfterReleaseInDebug(t *testing.T) {
	http.SetRequestPooling(true, true)
	t.Cleanup(func() { http.SetRequestPooling(false, false) })
	req := http.AcquireRequest(nil)
	gen := req.Generation()
	if req.Method() != "" && gen == 0 {
		t.Fatal("generation")
	}
	http.ReleaseRequest(req)
	if req.Generation() == gen {
		t.Fatal("release did not bump the generation")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected a use-after-release panic")
		}
	}()
	_ = req.Method()
}

func TestPooledResponsePanicsAfterReleaseInDebug(t *testing.T) {
	http.SetRequestPooling(true, true)
	t.Cleanup(func() { http.SetRequestPooling(false, false) })
	resp := http.JSON(map[string]string{"message": "Hello, World!"})
	http.ReleaseResponse(resp)
	defer func() {
		if recover() == nil {
			t.Fatal("expected a use-after-release panic")
		}
	}()
	_ = resp.Content()
}

func TestDebugPoolNeverReissuesReleasedRequest(t *testing.T) {
	http.SetRequestPooling(true, true)
	t.Cleanup(func() { http.SetRequestPooling(false, false) })
	stale := http.AcquireRequest(nil)
	http.ReleaseRequest(stale)
	for i := 0; i < 256; i++ {
		fresh := http.AcquireRequest(nil)
		if fresh == stale {
			t.Fatalf("acquire %d returned the released request", i)
		}
		http.ReleaseRequest(fresh)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("stale request did not panic after later acquires")
		}
	}()
	_ = stale.Path()
}

func TestReleasedRequestAccessorsPanicInDebug(t *testing.T) {
	http.SetRequestPooling(true, true)
	t.Cleanup(func() { http.SetRequestPooling(false, false) })
	calls := map[string]func(*http.Request){
		"Path":         func(r *http.Request) { _ = r.Path() },
		"Query":        func(r *http.Request) { _ = r.Query("q") },
		"Input":        func(r *http.Request) { _ = r.Input("q") },
		"Cookie":       func(r *http.Request) { _ = r.Cookie("sid") },
		"IP":           func(r *http.Request) { _ = r.IP() },
		"Host":         func(r *http.Request) { _ = r.Host() },
		"Context":      func(r *http.Request) { _ = r.Context() },
		"Get":          func(r *http.Request) { _ = r.Get("user") },
		"Set":          func(r *http.Request) { r.Set("user", 1) },
		"RouteParams":  func(r *http.Request) { _ = r.RouteParams() },
		"HeaderValues": func(r *http.Request) { _ = r.HeaderValues("Accept") },
		"Session":      func(r *http.Request) { _ = r.Session() },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			req := http.AcquireRequest(nil)
			http.ReleaseRequest(req)
			defer func() {
				if recover() == nil {
					t.Fatalf("%s after release did not panic", name)
				}
			}()
			call(req)
		})
	}
}

func TestReleaseAfterCommitKeepsStreamPair(t *testing.T) {
	http.SetRequestPooling(true, true)
	t.Cleanup(func() { http.SetRequestPooling(false, false) })
	noop := func(stdhttp.ResponseWriter, stdhttp.Flusher) error { return nil }
	kept := map[string]*http.Response{
		"Stream":     http.Stream("text/event-stream", noop),
		"StreamBody": http.StreamBody("text/plain", strings.NewReader("x"), 1),
		"Hijack":     http.Hijack(func(net.Conn, []byte) error { return nil }),
	}
	for name, resp := range kept {
		t.Run(name, func(t *testing.T) {
			req := http.AcquireRequest(nil)
			http.ReleaseAfterCommit(req, resp)
			_ = req.Path()
			_ = resp.StatusCode()
		})
	}

	req := http.AcquireRequest(nil)
	resp := http.JSON(map[string]string{"ok": "1"})
	http.ReleaseAfterCommit(req, resp)
	defer func() {
		if recover() == nil {
			t.Fatal("plain response pair was not released")
		}
	}()
	_ = req.Path()
}

func TestPoolDebugOffStillReuses(t *testing.T) {
	http.SetRequestPooling(true, false)
	t.Cleanup(func() { http.SetRequestPooling(false, false) })
	allocs := testing.AllocsPerRun(100, func() {
		http.ReleaseRequest(http.AcquireRequest(nil))
	})
	if allocs != 0 {
		t.Fatalf("pooled acquire/release allocates %.1f per run", allocs)
	}
}

func TestPoolingOffAllocatesDistinctRequests(t *testing.T) {
	http.SetRequestPooling(false, false)
	a := http.AcquireRequest(nil)
	http.ReleaseRequest(a)
	b := http.AcquireRequest(nil)
	if a == b {
		t.Fatal("pooling off returned the same request")
	}
}

func BenchmarkNewRequest(b *testing.B) {
	http.SetRequestPooling(false, false)
	b.Cleanup(func() { http.SetRequestPooling(false, false) })
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = http.AcquireRequest(nil)
	}
}

func BenchmarkPooledRequest(b *testing.B) {
	http.SetRequestPooling(true, false)
	b.Cleanup(func() { http.SetRequestPooling(false, false) })
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := http.AcquireRequest(nil)
		http.ReleaseRequest(r)
	}
}
