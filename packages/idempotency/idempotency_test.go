package idempotency_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/cache"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/idempotency"
)

func TestIdempotencyMiddlewareReplays(t *testing.T) {
	store := cache.NewMemoryStore()
	manager := cache.NewManager("memory", map[string]cache.Store{"memory": store})
	calls := 0

	handler := idempotency.Middleware(manager, time.Hour)(func(req *http.Request) *http.Response {
		calls++
		return http.JSON(map[string]any{"n": calls}).Status(201)
	})

	makeReq := func() *http.Request {
		raw := httptest.NewRequest("POST", "/api/idempotent", nil)
		raw.Header.Set(idempotency.HeaderKey, "abc-123")
		return http.NewRequest(raw)
	}

	first := handler(makeReq())
	second := handler(makeReq())
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
	if first.StatusCode() != 201 || second.StatusCode() != 201 {
		t.Fatalf("unexpected statuses %d %d", first.StatusCode(), second.StatusCode())
	}
	if second.Headers().Get("Idempotent-Replayed") != "true" {
		t.Fatal("expected replay header")
	}
}
