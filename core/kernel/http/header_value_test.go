package http

import (
	"testing"

	"github.com/zatrano/rawhttp"
)

func TestHeaderValueMissDoesNotAllocateOrBuildMap(t *testing.T) {
	req := NewRequest(&rawhttp.Ctx{})
	n := testing.AllocsPerRun(200, func() {
		_, _ = req.HeaderValue("X-Request-ID")
		_, _ = req.HeaderValue("X-Correlation-ID")
		_, _ = req.HeaderValue("Traceparent")
		_, _ = req.HeaderValue("Origin")
		_ = req.headerBytes("X-Forwarded-For")
	})
	if n != 0 {
		t.Fatalf("allocs=%v", n)
	}
	if req.HeaderCacheBuilt() {
		t.Fatal("HeaderValue built the header map")
	}
	if req.Header("Accept", "fallback") != "fallback" {
		t.Fatal("fallback")
	}
	if !req.HeaderCacheBuilt() {
		t.Fatal("Header did not build the map")
	}
}
