package middleware_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/middleware"
)

func TestRecoverHidesPanicInProduction(t *testing.T) {
	h := middleware.Recover(func(req *http.Request) *http.Response {
		panic("database password hunter2")
	})
	resp := h(http.NewRequest(httptest.NewRequest("GET", "/", nil)))
	if resp.StatusCode() != 500 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
	body := string(resp.Content())
	if strings.Contains(body, "hunter2") || strings.Contains(body, "password") {
		t.Fatalf("panic leaked: %s", body)
	}
}

func TestRecoverDebugIncludesPanic(t *testing.T) {
	h := middleware.RecoverDebug(func(req *http.Request) *http.Response {
		panic("visible")
	})
	resp := h(http.NewRequest(httptest.NewRequest("GET", "/", nil)))
	if !strings.Contains(string(resp.Content()), "visible") {
		t.Fatalf("body=%s", resp.Content())
	}
}

func TestRequestIDPropagatesIncoming(t *testing.T) {
	h := middleware.RequestID(func(req *http.Request) *http.Response {
		id, _ := req.Get("request_id").(string)
		return http.Text(id)
	})
	raw := httptest.NewRequest("GET", "/", nil)
	raw.Header.Set("X-Request-ID", "abc-123")
	resp := h(http.NewRequest(raw))
	if string(resp.Content()) != "abc-123" {
		t.Fatalf("id=%s", resp.Content())
	}
	if resp.Headers().Get("X-Request-ID") != "abc-123" {
		t.Fatalf("header=%s", resp.Headers().Get("X-Request-ID"))
	}
}

func TestRequestIDRejectsGarbageAndGenerates(t *testing.T) {
	h := middleware.RequestID(func(req *http.Request) *http.Response {
		id, _ := req.Get("request_id").(string)
		return http.Text(id)
	})
	raw := httptest.NewRequest("GET", "/", nil)
	raw.Header.Set("X-Request-ID", "not a valid id because spaces")
	resp := h(http.NewRequest(raw))
	id := string(resp.Content())
	if id == "" || strings.Contains(id, " ") {
		t.Fatalf("id=%q", id)
	}
	if len(id) != 32 {
		t.Fatalf("generated id length=%d", len(id))
	}
}

func TestRequestIDFromTraceparent(t *testing.T) {
	h := middleware.RequestID(func(req *http.Request) *http.Response {
		id, _ := req.Get("request_id").(string)
		return http.Text(id)
	})
	raw := httptest.NewRequest("GET", "/", nil)
	raw.Header.Set("Traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	resp := h(http.NewRequest(raw))
	if string(resp.Content()) != "0af7651916cd43dd8448eb211c80319c" {
		t.Fatalf("id=%s", resp.Content())
	}
	if resp.Headers().Get("Traceparent") != "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01" {
		t.Fatalf("traceparent=%s", resp.Headers().Get("Traceparent"))
	}
}
