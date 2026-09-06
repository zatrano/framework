package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestMergePackageEnvFileIdempotentAndSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.example")
	if err := os.WriteFile(path, []byte("APP_NAME=Demo\nCACHE_STORE=file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snippet := `# cache
CACHE_STORE=redis
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
`
	n, err := mergePackageEnvFile(path, "cache", snippet)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("added=%d want 2 (REDIS_HOST, REDIS_PORT)", n)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "CACHE_STORE=file") {
		t.Fatalf("must keep existing CACHE_STORE:\n%s", text)
	}
	if strings.Count(text, "CACHE_STORE=") != 1 {
		t.Fatalf("must not duplicate CACHE_STORE:\n%s", text)
	}
	if !strings.Contains(text, "# --- zatrano:package:cache ---") {
		t.Fatalf("missing section marker:\n%s", text)
	}
	if !strings.Contains(text, "REDIS_HOST=127.0.0.1") {
		t.Fatalf("missing REDIS_HOST:\n%s", text)
	}

	n, err = mergePackageEnvFile(path, "cache", snippet)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("second merge should be a no-op, added=%d", n)
	}
}

func TestMergePackageEnvFileCommentedKeysCountAsPresent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.example")
	if err := os.WriteFile(path, []byte("# DB_CONNECTION=sqlite\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := mergePackageEnvFile(path, "database", "DB_CONNECTION=mysql\nDB_HOST=127.0.0.1\n")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("added=%d want 1 (DB_HOST only)", n)
	}
	body, _ := os.ReadFile(path)
	if strings.Contains(string(body), "DB_CONNECTION=mysql") {
		t.Fatalf("must not overwrite commented DB_CONNECTION:\n%s", body)
	}
}

func TestApplyPackageEnvFromPackagesDir(t *testing.T) {
	packages := t.TempDir()
	cacheDir := filepath.Join(packages, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, ".env.example"), []byte("CACHE_STORE=file\nREDIS_HOST=127.0.0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PACKAGES_DIR", packages)

	app := kernel.NewApplication(t.TempDir())
	if err := os.WriteFile(app.BasePath(".env.example"), []byte("APP_NAME=Demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app.BasePath(".env"), []byte("APP_NAME=Demo\nAPP_KEY=secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	merged, err := applyPackageEnv(app, "cache")
	if err != nil {
		t.Fatal(err)
	}
	if !merged {
		t.Fatal("expected merge into .env.example")
	}
	example, _ := os.ReadFile(app.BasePath(".env.example"))
	if !strings.Contains(string(example), "CACHE_STORE=file") {
		t.Fatalf(".env.example missing CACHE_STORE:\n%s", example)
	}
	envBody, _ := os.ReadFile(app.BasePath(".env"))
	if !strings.Contains(string(envBody), "CACHE_STORE=file") {
		t.Fatalf(".env missing CACHE_STORE:\n%s", envBody)
	}
	if !strings.Contains(string(envBody), "APP_KEY=secret") {
		t.Fatalf(".env lost APP_KEY:\n%s", envBody)
	}

	merged, err = applyPackageEnv(app, "cache")
	if err != nil {
		t.Fatal(err)
	}
	if merged {
		t.Fatal("second apply should not rewrite")
	}
}

func TestApplyPackageEnvSkipsUnknownPackage(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	if err := os.WriteFile(app.BasePath(".env.example"), []byte("APP_NAME=Demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	merged, err := applyPackageEnv(app, "does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	if merged {
		t.Fatal("unknown package must not invent env keys")
	}
}
