package kernel

import (
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
)

func TestPublicFileLookupKeyMirrorsResolve(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		wantOK  bool
		wantKey string
	}{
		{in: "/css/app.css", wantOK: true, wantKey: foldPublicKey("/css/app.css")},
		{in: "/css/./app.css", wantOK: true, wantKey: foldPublicKey("/css/app.css")},
		{in: "/css//app.css", wantOK: true, wantKey: foldPublicKey("/css/app.css")},
		{in: "/../secret.txt", wantOK: false},
		{in: "/css/../../secret.txt", wantOK: false},
		{in: "/css/foo/../app.css", wantOK: false},
		{in: "/", wantOK: false},
		{in: "", wantOK: false},
	}
	for _, tc := range cases {
		key, ok := publicFileLookupKey(tc.in)
		if ok != tc.wantOK {
			t.Errorf("%q ok=%v want %v", tc.in, ok, tc.wantOK)
		}
		if tc.wantOK && key != tc.wantKey {
			t.Errorf("%q key=%q want %q", tc.in, key, tc.wantKey)
		}
	}
}

func TestPublicFileIndexConcurrentOnce(t *testing.T) {
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(filepath.Join(public, "css"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "css", "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := NewApplication(dir)
	app.environment = "production"
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/missing.json", nil))
			_ = app.publicFile(req)
		}()
	}
	wg.Wait()
	if app.publicFiles == nil {
		t.Fatal("index must be built once under concurrent first hits")
	}
	if _, ok := app.publicFiles.files[foldPublicKey("/css/app.css")]; !ok {
		t.Fatalf("index missing /css/app.css: %#v", app.publicFiles.files)
	}
}

func TestPublicFileOracleMatchesSlowPath(t *testing.T) {
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
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("LEAKED_SECRET"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := NewApplication(dir)
	app.environment = "production"
	app.ensurePublicFileIndex()

	paths := []string{
		"/app.css",
		"/css/app.css",
		"/css/./app.css",
		"/css//app.css",
		"/missing.css",
		"/",
		"/../secret.txt",
		"/css/../../secret.txt",
		"/a/b/../../../secret.txt",
		"/%2e%2e/secret.txt",
		"/%2e%2e/%2e%2e/secret.txt",
		"/..%2fsecret.txt",
		"/%2e%2e%2fsecret.txt",
		"/css/foo/../app.css",
	}
	for _, path := range paths {
		raw := httptest.NewRequest(stdhttp.MethodGet, path, nil)
		req := http.NewRequest(raw)
		got := app.publicFile(req)
		slow := app.servePublicFile(req)
		if !samePublicResponse(got, slow) {
			t.Errorf("path %q indexed=%v slow=%v", path, describePublic(got), describePublic(slow))
		}
		if got != nil && got.FilePath() != "" {
			if body, err := os.ReadFile(got.FilePath()); err == nil && string(body) == "LEAKED_SECRET" {
				t.Errorf("leaked secret via %q", path)
			}
		}
	}
}

func samePublicResponse(a, b *http.Response) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.StatusCode() == b.StatusCode() && a.FilePath() == b.FilePath()
}

func describePublic(resp *http.Response) string {
	if resp == nil {
		return "nil"
	}
	return resp.FilePath()
}
