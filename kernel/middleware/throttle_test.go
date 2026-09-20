package middleware_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/middleware"
)

type memAttempts struct {
	mu   sync.Mutex
	hits map[string]int
}

func (m *memAttempts) Take(key string, maxAttempts int, decay time.Duration) (bool, int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hits == nil {
		m.hits = map[string]int{}
	}
	if m.hits[key] >= maxAttempts {
		return false, 0, 7
	}
	m.hits[key]++
	remaining := maxAttempts - m.hits[key]
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining, 0
}

func TestThrottleHTTPHeaders(t *testing.T) {
	lim := &memAttempts{}
	handler := middleware.Throttle(lim, 1, time.Minute, func(req *http.Request) string {
		return "k"
	})(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})

	first := handler(&http.Request{})
	if first.StatusCode() != 200 || first.GetHeader("X-RateLimit-Remaining") != "0" {
		t.Fatalf("first status=%d remaining=%q", first.StatusCode(), first.GetHeader("X-RateLimit-Remaining"))
	}
	second := handler(&http.Request{})
	if second.StatusCode() != 429 || second.GetHeader("Retry-After") != "7" {
		t.Fatalf("second status=%d retry=%q", second.StatusCode(), second.GetHeader("Retry-After"))
	}
	if second.GetHeader("X-RateLimit-Limit") != "1" || second.GetHeader("X-RateLimit-Remaining") != "0" {
		t.Fatalf("429 headers limit=%q remaining=%q", second.GetHeader("X-RateLimit-Limit"), second.GetHeader("X-RateLimit-Remaining"))
	}
}

func TestThrottleNilLimiterFailOpen(t *testing.T) {
	handler := middleware.Throttle(nil, 1, time.Minute, func(req *http.Request) string {
		return "k"
	})(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	if handler(&http.Request{}).StatusCode() != 200 {
		t.Fatal("nil limiter must pass through")
	}

	handler = middleware.Throttle(&memAttempts{}, 1, time.Minute, nil)(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	if handler(&http.Request{}).StatusCode() != 200 {
		t.Fatal("nil key func must pass through")
	}
}

func TestThrottleConcurrentTakes(t *testing.T) {
	lim := &memAttempts{}
	handler := middleware.Throttle(lim, 10, time.Minute, func(req *http.Request) string {
		return "shared"
	})(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})

	var allowed atomic.Int32
	var blocked atomic.Int32
	var unexpected atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := handler(&http.Request{})
			switch resp.StatusCode() {
			case 429:
				blocked.Add(1)
			case 200:
				allowed.Add(1)
			default:
				unexpected.Add(1)
			}
		}()
	}
	wg.Wait()
	if unexpected.Load() != 0 {
		t.Fatalf("unexpected status count=%d", unexpected.Load())
	}
	if allowed.Load() != 10 || blocked.Load() != 30 {
		t.Fatalf("allowed=%d blocked=%d", allowed.Load(), blocked.Load())
	}
}
