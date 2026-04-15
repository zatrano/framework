package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zatrano/framework/pkg/config"
	"github.com/zatrano/framework/pkg/core"
	"go.uber.org/zap"
)

func TestHTTPMiddleware_CORS_Preflight(t *testing.T) {
	cfg := &config.Config{
		Env:      "dev",
		AppName:  "ZATRANO-test",
		HTTPAddr: ":0",
		LogLevel: "error",
		HTTP: config.HTTP{
			CORSEnabled:      true,
			CORSAllowOrigins: []string{"https://app.example"},
			CORSAllowMethods: []string{"GET", "OPTIONS"},
		},
	}
	log := zap.NewNop()
	a := &core.App{Config: cfg, Log: log}
	app := core.NewFiber(a)
	Mount(a, app, MountOptions{})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/public/ping", nil)
	req.Header.Set("Origin", "https://app.example")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestHTTPMiddleware_RateLimit(t *testing.T) {
	cfg := &config.Config{
		Env:      "dev",
		AppName:  "ZATRANO-test",
		HTTPAddr: ":0",
		LogLevel: "error",
		HTTP: config.HTTP{
			RateLimitEnabled: true,
			RateLimitMax:     2,
			RateLimitWindow:  time.Minute,
		},
	}
	log := zap.NewNop()
	a := &core.App{Config: cfg, Log: log}
	app := core.NewFiber(a)
	Mount(a, app, MountOptions{})

	getHealth := func() (int, string, http.Header) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b), resp.Header
	}
	code1, _, h1 := getHealth()
	if code1 != http.StatusOK {
		t.Fatalf("first request: %d", code1)
	}
	if h1.Get("X-RateLimit-Limit") != "2" {
		t.Fatalf("first X-RateLimit-Limit: got %q", h1.Get("X-RateLimit-Limit"))
	}

	if code, _, _ := getHealth(); code != http.StatusOK {
		t.Fatalf("second request: %d", code)
	}

	code, body, hdr := getHealth()
	if code != http.StatusTooManyRequests {
		t.Fatalf("third request: want 429, got %d body %s", code, body)
	}
	if hdr.Get("Retry-After") == "" {
		t.Fatal("expected Retry-After on 429 (RFC 6585)")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %#v", payload)
	}
	if errObj["code"] != float64(429) {
		t.Fatalf("error.code: %#v", errObj["code"])
	}
}
