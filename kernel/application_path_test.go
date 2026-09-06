package kernel_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

const leakedSecret = "LEAKED_SECRET"

type pathFixture struct {
	app    *kernel.Application
	public string
	secret string
}

func bootPathApp(t *testing.T) pathFixture {
	t.Helper()
	base := t.TempDir()
	public := filepath.Join(base, "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(base, "secret.txt")
	if err := os.WriteFile(secret, []byte(leakedSecret), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(base)
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return pathFixture{app: app, public: public, secret: secret}
}

func (f pathFixture) get(path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	f.app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func assertNotLeaked(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if strings.Contains(rec.Body.String(), leakedSecret) {
		t.Fatalf("secret leaked: status=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Code == http.StatusOK && rec.Body.String() == leakedSecret {
		t.Fatal("outside-root file was served")
	}
}

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	target, err := filepath.Abs(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err == nil {
		return
	}
	info, statErr := os.Stat(target)
	if statErr == nil && info.IsDir() {
		out, jerr := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
		if jerr == nil {
			t.Cleanup(func() { _ = os.Remove(link) })
			return
		}
		t.Skipf("symlinks/junctions not supported: %v (%s)", jerr, out)
	}
	t.Skipf("symlinks not supported: %v", err)
}

func TestTHP01NormalPublicFile(t *testing.T) {
	f := bootPathApp(t)
	rec := f.get("/app.css")
	if rec.Code != http.StatusOK || rec.Body.String() != "body{}" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestTHP02DotDotTraversal(t *testing.T) {
	f := bootPathApp(t)
	assertNotLeaked(t, f.get("/../secret.txt"))
}

func TestTHP03NestedDotDotTraversal(t *testing.T) {
	f := bootPathApp(t)
	assertNotLeaked(t, f.get("/css/../../secret.txt"))
	assertNotLeaked(t, f.get("/a/b/../../../secret.txt"))
}

func TestTHP04AbsolutePath(t *testing.T) {
	f := bootPathApp(t)
	assertNotLeaked(t, f.get(filepath.ToSlash(f.secret)))
}

func TestTHP05SymlinkOutsideRoot(t *testing.T) {
	f := bootPathApp(t)
	// File symlink when the OS allows it; otherwise a directory junction
	// to the secret's parent (Windows without SeCreateSymbolicLink).
	if err := os.Symlink(f.secret, filepath.Join(f.public, "link")); err == nil {
		assertNotLeaked(t, f.get("/link"))
		return
	}
	mustSymlink(t, filepath.Dir(f.secret), filepath.Join(f.public, "link"))
	assertNotLeaked(t, f.get("/link/"+filepath.Base(f.secret)))
}

func TestTHP06NestedSymlinkOutsideRoot(t *testing.T) {
	f := bootPathApp(t)
	outsideDir := filepath.Dir(f.secret)
	sub := filepath.Join(f.public, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, outsideDir, filepath.Join(sub, "link"))
	assertNotLeaked(t, f.get("/sub/link/"+filepath.Base(f.secret)))
}

func TestTHP07SymlinkInsidePublic(t *testing.T) {
	f := bootPathApp(t)
	alias := filepath.Join(f.public, "alias.css")
	if err := os.Symlink(filepath.Join(f.public, "app.css"), alias); err == nil {
		rec := f.get("/alias.css")
		if rec.Code != http.StatusOK || rec.Body.String() != "body{}" {
			t.Fatalf("inside file symlink rejected: status=%d body=%q", rec.Code, rec.Body.String())
		}
		return
	}
	cssDir := filepath.Join(f.public, "css")
	if err := os.MkdirAll(cssDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cssDir, "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, cssDir, filepath.Join(f.public, "static"))
	rec := f.get("/static/app.css")
	if rec.Code != http.StatusOK || rec.Body.String() != "body{}" {
		t.Fatalf("inside dir junction rejected: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestTHP08BrokenSymlink(t *testing.T) {
	f := bootPathApp(t)
	missing := filepath.Join(f.public, "missing.css")
	broken := filepath.Join(f.public, "broken.css")
	if err := os.Symlink(missing, broken); err == nil {
		rec := f.get("/broken.css")
		if rec.Code == http.StatusOK {
			t.Fatalf("broken symlink served: body=%q", rec.Body.String())
		}
		assertNotLeaked(t, rec)
		return
	}
	ghost := filepath.Join(f.public, "ghost")
	if err := os.MkdirAll(ghost, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(f.public, "broken")
	mustSymlink(t, ghost, link)
	if err := os.Remove(ghost); err != nil {
		t.Fatal(err)
	}
	rec := f.get("/broken/app.css")
	if rec.Code == http.StatusOK {
		t.Fatalf("dangling junction served: body=%q", rec.Body.String())
	}
	assertNotLeaked(t, rec)
}

func TestTHP09EncodedTraversal(t *testing.T) {
	f := bootPathApp(t)
	for _, path := range []string{
		"/%2e%2e/secret.txt",
		"/%2e%2e/%2e%2e/secret.txt",
		"/..%2fsecret.txt",
		"/%2e%2e%2fsecret.txt",
	} {
		assertNotLeaked(t, f.get(path))
	}
}

func TestTHP10SymlinkThenNormalizedEscape(t *testing.T) {
	f := bootPathApp(t)
	mustSymlink(t, filepath.Dir(f.secret), filepath.Join(f.public, "link"))
	assertNotLeaked(t, f.get("/link/"+filepath.Base(f.secret)))
}
