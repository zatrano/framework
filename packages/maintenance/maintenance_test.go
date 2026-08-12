package maintenance_test

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/maintenance"
)

func TestMaintenanceToggleAndMiddleware(t *testing.T) {
	dir := t.TempDir()
	mgr := maintenance.New(filepath.Join(dir, "framework"))
	if mgr.Active() {
		t.Fatal("expected inactive")
	}
	if err := mgr.Enable(maintenance.Payload{Message: "down for update", Secret: "bypass"}); err != nil {
		t.Fatal(err)
	}
	if !mgr.Active() {
		t.Fatal("expected active")
	}

	handler := mgr.Middleware()(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})

	blocked := handler(http.NewRequest(httptest.NewRequest("GET", "/", nil)))
	if blocked.StatusCode() != 503 {
		t.Fatalf("expected 503, got %d", blocked.StatusCode())
	}

	raw := httptest.NewRequest("GET", "/?secret=bypass", nil)
	allowed := handler(http.NewRequest(raw))
	if allowed.StatusCode() != 200 {
		t.Fatalf("expected 200 with secret, got %d", allowed.StatusCode())
	}

	up := httptest.NewRequest("GET", "/up", nil)
	health := handler(http.NewRequest(up))
	if health.StatusCode() != 200 {
		t.Fatalf("expected /up allowed, got %d", health.StatusCode())
	}

	if err := mgr.Disable(); err != nil {
		t.Fatal(err)
	}
}
