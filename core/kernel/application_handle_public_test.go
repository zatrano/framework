package kernel_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel"
	zhttp "github.com/zatrano/framework/v3/core/kernel/http"
)

func TestTH01PublicFileSecurityHeadersAndRequestID(t *testing.T) {
	app := bootPublicApp(t)
	er := serveHandle(t, app, http.MethodGet, "/app.css", map[string]string{
		"X-Request-ID": "static-req-1",
	}, "")
	if er.Status != http.StatusOK {
		t.Fatalf("status=%d", er.Status)
	}
	if string(er.Body) != "body{}" {
		t.Fatalf("body=%q", er.Body)
	}
	if er.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff: %v", er.Header)
	}
	if er.Header.Get("X-Frame-Options") == "" {
		t.Fatalf("missing X-Frame-Options: %v", er.Header)
	}
	if er.Header.Get("X-Request-ID") != "static-req-1" {
		t.Fatalf("request id=%q", er.Header.Get("X-Request-ID"))
	}
	if er.Header.Get("Strict-Transport-Security") != "" {
		t.Fatal("development public files must not emit default HSTS")
	}
	if cd := er.Header.Get("Content-Disposition"); strings.Contains(strings.ToLower(cd), "attachment") {
		t.Fatalf("public asset must not be an attachment: %q", cd)
	}
}

func TestTH03PostPublicFileDoesNotServeFile(t *testing.T) {
	app := bootPublicApp(t)

	post := serveHandle(t, app, http.MethodPost, "/app.css", nil, "")
	if post.Status == http.StatusOK && string(post.Body) == "body{}" {
		t.Fatal("POST served public file, bypassing method policy")
	}

	override := serveHandle(t, app, http.MethodPost, "/app.css", map[string]string{
		"X-HTTP-Method-Override": "DELETE",
	}, "")
	if override.Status == http.StatusOK && string(override.Body) == "body{}" {
		t.Fatal("method-overridden POST served public file")
	}
}

func TestPublicFileHeadKeepsSecurityHeaders(t *testing.T) {
	app := bootPublicApp(t)
	er := serveHandle(t, app, http.MethodHead, "/app.css", nil, "")
	if er.Status != http.StatusOK {
		t.Fatalf("status=%d", er.Status)
	}
	if er.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff on HEAD: %v", er.Header)
	}
	if len(er.Body) != 0 {
		t.Fatalf("HEAD body=%q", er.Body)
	}
}

func TestProductionPublicFileIndexServesKnownFile(t *testing.T) {
	app, _ := bootProductionPublicApp(t, nil)
	er := serveHandle(t, app, http.MethodGet, "/app.css", nil, "")
	if er.Status != http.StatusOK || string(er.Body) != "body{}" {
		t.Fatalf("status=%d body=%q", er.Status, er.Body)
	}
	nested := serveHandle(t, app, http.MethodGet, "/css/app.css", nil, "")
	if nested.Status != http.StatusOK || string(nested.Body) != "nested{}" {
		t.Fatalf("nested status=%d body=%q", nested.Status, nested.Body)
	}
}

func TestProductionPublicFileIndexMissGoesToRouter(t *testing.T) {
	app, _ := bootProductionPublicApp(t, func(app *kernel.Application) {
		app.Router().Get("/json", func(req *zhttp.Request) *zhttp.Response {
			return zhttp.JSON(map[string]any{"ok": true})
		})
	})
	miss := serveHandle(t, app, http.MethodGet, "/no-such-static.css", nil, "")
	if miss.Status != http.StatusNotFound {
		t.Fatalf("miss status=%d body=%q", miss.Status, miss.Body)
	}
	json := serveHandle(t, app, http.MethodGet, "/json", nil, "")
	if json.Status != http.StatusOK || !strings.Contains(string(json.Body), `"ok"`) {
		t.Fatalf("route status=%d body=%q", json.Status, json.Body)
	}
}

func TestProductionPublicFileHeadAndSlashAndPost(t *testing.T) {
	app, _ := bootProductionPublicApp(t, func(app *kernel.Application) {
		app.Router().Post("/app.css", func(req *zhttp.Request) *zhttp.Response {
			return zhttp.Text("posted")
		})
		app.Router().Get("/", func(req *zhttp.Request) *zhttp.Response {
			return zhttp.Text("home")
		})
	})
	head := serveHandle(t, app, http.MethodHead, "/app.css", nil, "")
	if head.Status != http.StatusOK {
		t.Fatalf("HEAD status=%d", head.Status)
	}
	if len(head.Body) != 0 {
		t.Fatalf("HEAD body=%q", head.Body)
	}
	root := serveHandle(t, app, http.MethodGet, "/", nil, "")
	if root.Status != http.StatusOK || string(root.Body) != "home" {
		t.Fatalf("GET / status=%d body=%q", root.Status, root.Body)
	}
	post := serveHandle(t, app, http.MethodPost, "/app.css", nil, "")
	if string(post.Body) == "body{}" {
		t.Fatal("POST served public file")
	}
	if post.Status != http.StatusOK || string(post.Body) != "posted" {
		t.Fatalf("POST status=%d body=%q", post.Status, post.Body)
	}
}

func TestLocalPublicFileSeesLiveAdds(t *testing.T) {
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.IsProduction() {
		t.Fatal("expected non-production")
	}
	if err := os.WriteFile(filepath.Join(public, "late.css"), []byte("late"), 0o644); err != nil {
		t.Fatal(err)
	}
	er := serveHandle(t, app, http.MethodGet, "/late.css", nil, "")
	if er.Status != http.StatusOK || string(er.Body) != "late" {
		t.Fatalf("local live add: status=%d body=%q", er.Status, er.Body)
	}
}
