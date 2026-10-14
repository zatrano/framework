package http_test

import (
	"encoding/json"
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
	t.Cleanup(func() { http.SetRequestPooling(true, false) })
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
	t.Cleanup(func() { http.SetRequestPooling(true, false) })
	resp := http.JSON(map[string]string{"message": "Hello, World!"})
	http.ReleaseResponse(resp)
	defer func() {
		if recover() == nil {
			t.Fatal("expected a use-after-release panic")
		}
	}()
	_ = resp.Content()
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
	b.Cleanup(func() { http.SetRequestPooling(true, false) })
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = http.AcquireRequest(nil)
	}
}

func BenchmarkPooledRequest(b *testing.B) {
	http.SetRequestPooling(true, false)
	b.Cleanup(func() { http.SetRequestPooling(true, false) })
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := http.AcquireRequest(nil)
		http.ReleaseRequest(r)
	}
}
