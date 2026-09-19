package kernel_test

import (
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/http"
)

func bootProductionPublicApp(t testing.TB, routes func(*kernel.Application)) (*kernel.Application, string) {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(filepath.Join(public, "css"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "css", "app.css"), []byte("nested{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	t.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	if routes != nil {
		routes(app)
	}
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return app, dir
}

func servePath(app *kernel.Application, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestProductionPublicFileIndexServesKnownFile(t *testing.T) {
	app, _ := bootProductionPublicApp(t, nil)
	rec := servePath(app, stdhttp.MethodGet, "/app.css")
	if rec.Code != stdhttp.StatusOK || rec.Body.String() != "body{}" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	nested := servePath(app, stdhttp.MethodGet, "/css/app.css")
	if nested.Code != stdhttp.StatusOK || nested.Body.String() != "nested{}" {
		t.Fatalf("nested status=%d body=%q", nested.Code, nested.Body.String())
	}
}

func TestProductionPublicFileIndexMissGoesToRouter(t *testing.T) {
	app, _ := bootProductionPublicApp(t, func(app *kernel.Application) {
		app.Router().Get("/json", func(req *http.Request) *http.Response {
			return http.JSON(map[string]any{"ok": true})
		})
	})
	miss := servePath(app, stdhttp.MethodGet, "/no-such-static.css")
	if miss.Code != stdhttp.StatusNotFound {
		t.Fatalf("miss status=%d body=%q", miss.Code, miss.Body.String())
	}
	json := servePath(app, stdhttp.MethodGet, "/json")
	if json.Code != stdhttp.StatusOK || !strings.Contains(json.Body.String(), `"ok"`) {
		t.Fatalf("route status=%d body=%q", json.Code, json.Body.String())
	}
}

func TestProductionPublicFileHeadAndSlashAndPost(t *testing.T) {
	app, _ := bootProductionPublicApp(t, func(app *kernel.Application) {
		app.Router().Post("/app.css", func(req *http.Request) *http.Response {
			return http.Text("posted")
		})
		app.Router().Get("/", func(req *http.Request) *http.Response {
			return http.Text("home")
		})
	})
	head := servePath(app, stdhttp.MethodHead, "/app.css")
	if head.Code != stdhttp.StatusOK {
		t.Fatalf("HEAD status=%d", head.Code)
	}
	if head.Body.Len() != 0 {
		t.Fatalf("HEAD body=%q", head.Body.String())
	}
	root := servePath(app, stdhttp.MethodGet, "/")
	if root.Code != stdhttp.StatusOK || root.Body.String() != "home" {
		t.Fatalf("GET / status=%d body=%q", root.Code, root.Body.String())
	}
	post := servePath(app, stdhttp.MethodPost, "/app.css")
	if post.Body.String() == "body{}" {
		t.Fatal("POST served public file")
	}
	if post.Code != stdhttp.StatusOK || post.Body.String() != "posted" {
		t.Fatalf("POST status=%d body=%q", post.Code, post.Body.String())
	}
}

func TestProductionPublicFileSymlinkDirStillServed(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	uploads := filepath.Join(public, "uploads")
	if err := os.MkdirAll(uploads, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploads, "pic.txt"), []byte("pic"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, uploads, filepath.Join(public, "storage"))
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := servePath(app, stdhttp.MethodGet, "/storage/pic.txt")
	if rec.Code != stdhttp.StatusOK || rec.Body.String() != "pic" {
		t.Fatalf("symlink dir not served: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestProductionPublicFileSymlinkEscapeStillRejected(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte(leakedSecret), 0o644); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, filepath.Dir(secret), filepath.Join(public, "link"))
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := servePath(app, stdhttp.MethodGet, "/link/"+filepath.Base(secret))
	assertNotLeaked(t, rec)
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
	rec := servePath(app, stdhttp.MethodGet, "/late.css")
	if rec.Code != stdhttp.StatusOK || rec.Body.String() != "late" {
		t.Fatalf("local live add: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestProductionPublicFileIgnoresPostBootRegularAdds(t *testing.T) {
	app, dir := bootProductionPublicApp(t, nil)
	if err := os.WriteFile(filepath.Join(dir, "public", "late.css"), []byte("late"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := servePath(app, stdhttp.MethodGet, "/late.css")
	if rec.Code == stdhttp.StatusOK && rec.Body.String() == "late" {
		t.Fatal("production served a regular file added after boot")
	}
}

func TestProductionPublicFileMissAfterDeletingPublicDir(t *testing.T) {
	app, dir := bootProductionPublicApp(t, func(app *kernel.Application) {
		app.Router().Get("/json", func(req *http.Request) *http.Response {
			return http.JSON(map[string]any{"ok": true})
		})
	})
	if err := os.RemoveAll(filepath.Join(dir, "public")); err != nil {
		t.Fatal(err)
	}
	miss := servePath(app, stdhttp.MethodGet, "/ghost.css")
	if miss.Code != stdhttp.StatusNotFound {
		t.Fatalf("deleted public miss status=%d", miss.Code)
	}
	json := servePath(app, stdhttp.MethodGet, "/json")
	if json.Code != stdhttp.StatusOK {
		t.Fatalf("route after public delete status=%d", json.Code)
	}
}

func BenchmarkServeHTTPNoStaticMatch(b *testing.B) {
	app, _ := bootProductionPublicApp(b, func(app *kernel.Application) {
		app.Router().Get("/json", func(req *http.Request) *http.Response {
			return http.JSON(map[string]any{"ok": true})
		})
	})
	req := httptest.NewRequest(stdhttp.MethodGet, "/json", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != stdhttp.StatusOK {
			b.Fatalf("status=%d", rec.Code)
		}
		_, _ = io.Copy(io.Discard, rec.Body)
	}
}
