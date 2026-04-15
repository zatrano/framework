package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/zatrano/framework/pkg/config"
	"github.com/zatrano/framework/pkg/core"
	"go.uber.org/zap"
)

func TestHealthAndRoot(t *testing.T) {
	cfg := &config.Config{
		Env:             "dev",
		AppName:         "ZATRANO-test",
		HTTPAddr:        ":0",
		HTTPReadTimeout: 5 * time.Second,
		LogLevel:        "error",
		LogDevelopment:  false,
	}

	log := zap.NewNop()

	a := &core.App{Config: cfg, Log: log}
	app := core.NewFiber(a)
	Mount(a, app, MountOptions{})

	t.Run("GET /health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["status"] != "ok" {
			t.Fatalf("unexpected body: %#v", body)
		}
	})

	t.Run("GET /ready without database", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status %d body %s", resp.StatusCode, b)
		}
	})

	t.Run("GET /", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		i18nObj, ok := body["i18n"].(map[string]any)
		if !ok {
			t.Fatalf("missing i18n in index JSON")
		}
		if i18nObj["enabled"] != false {
			t.Fatalf("expected i18n.enabled false by default, got %#v", i18nObj["enabled"])
		}
	})

	t.Run("GET /api/v1/public/ping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/public/ping", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}
	})

	app.Get("/__test_error", func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "test error body")
	})
	t.Run("error JSON includes request_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/__test_error", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status %d", resp.StatusCode)
		}
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		errObj, ok := body["error"].(map[string]any)
		if !ok {
			t.Fatalf("missing error object: %#v", body)
		}
		if errObj["message"] != "test error body" {
			t.Fatalf("message: %#v", errObj["message"])
		}
		rid, _ := errObj["request_id"].(string)
		if rid == "" {
			t.Fatalf("expected request_id in error payload, got %#v", errObj)
		}
	})
}
