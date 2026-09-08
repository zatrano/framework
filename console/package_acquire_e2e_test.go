package console

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/distribution/acquire"
	"github.com/zatrano/framework/v2/distribution/manifest"
	"github.com/zatrano/framework/v2/distribution/registry"
	"github.com/zatrano/framework/v2/kernel"
)

func frameworkRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func packagesCheckout(t *testing.T) string {
	t.Helper()
	if p := strings.TrimSpace(os.Getenv("PACKAGES_DIR")); p != "" {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	sibling := filepath.Join(filepath.Dir(frameworkRoot(t)), "packages")
	if st, err := os.Stat(sibling); err == nil && st.IsDir() {
		return sibling
	}
	return ""
}

func requireGoTool(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go executable not on PATH")
	}
}

func requirePackagesCheckout(t *testing.T) string {
	t.Helper()
	dir := packagesCheckout(t)
	if dir == "" {
		t.Skip("packages checkout not found (set PACKAGES_DIR or clone sibling packages/)")
	}
	return dir
}

func writeIsolatedConsumer(t *testing.T, packagesDir string, extraReplace map[string]string) string {
	t.Helper()
	root := t.TempDir()
	fw := filepath.ToSlash(frameworkRoot(t))
	var b strings.Builder
	b.WriteString("module example.com/acq-e2e\n\ngo 1.25.0\n\n")
	b.WriteString("replace github.com/zatrano/framework/v2 => " + fw + "\n")
	if packagesDir != "" {
		b.WriteString("replace github.com/zatrano/packages => " + filepath.ToSlash(packagesDir) + "\n")
	}
	for path, dir := range extraReplace {
		b.WriteString("replace " + path + " => " + filepath.ToSlash(dir) + "\n")
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bootstrap"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOTOOLCHAIN", "local")
	return root
}

func TestPackageAcquireE2ECatalogResolveAcquireInspect(t *testing.T) {
	requireGoTool(t)
	pkgDir := requirePackagesCheckout(t)
	root := writeIsolatedConsumer(t, pkgDir, nil)

	var searchBuf bytes.Buffer
	if err := (&PackageSearchCommand{out: &searchBuf}).Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(searchBuf.String(), `"selected"`) {
		t.Fatal("package:search is discovery and must not select a version")
	}
	var hits []searchHit
	if err := json.Unmarshal(searchBuf.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, h := range hits {
		if h.Name == "session" && h.Module == manifest.DefaultModule {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("search must return official session module, got %s", searchBuf.String())
	}

	var infoBuf bytes.Buffer
	if err := (&PackageInfoCommand{out: &infoBuf}).Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(infoBuf.String(), `"selected"`) {
		t.Fatal("package:info must not select a version")
	}
	if !strings.Contains(infoBuf.String(), `"name"`) || !strings.Contains(infoBuf.String(), "session") {
		t.Fatalf("info: %s", infoBuf.String())
	}

	var resolveBuf bytes.Buffer
	if err := (&PackageResolveCommand{out: &resolveBuf}).Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var resolved resolveView
	if err := json.Unmarshal(resolveBuf.Bytes(), &resolved); err != nil {
		t.Fatal(err)
	}
	if resolved.Name != "session" || resolved.Module != manifest.DefaultModule || resolved.Selected == "" {
		t.Fatalf("resolve must select a release: %#v", resolved)
	}

	var acquireBuf bytes.Buffer
	cmd := &PackageAcquireCommand{out: &acquireBuf, root: root}
	if err := cmd.Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatalf("package:acquire: %v\n%s", err, acquireBuf.String())
	}
	var view acquireCLIView
	if err := json.Unmarshal(acquireBuf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	wantArg := manifest.DefaultModule + "@main"
	if len(view.GoGetArgs) != 1 || view.GoGetArgs[0] != wantArg {
		t.Fatalf("CLI must pass the same GoGetArg as Resolve, got %v", view.GoGetArgs)
	}
	if view.Acquisition != acquireStatusSuccess {
		t.Fatalf("acquisition=%q view=%#v", view.Acquisition, view)
	}
	if view.Enablement != enablementNotRequested || view.Enabled {
		t.Fatalf("default acquire must not enable: %#v", view)
	}
	if _, err := os.Stat(filepath.Join(root, "zatrano.lock")); !os.IsNotExist(err) {
		t.Fatal("acquisition must not create zatrano.lock")
	}

	in, err := acquire.Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	req, ok := in.Requirement(manifest.DefaultModule)
	if !ok {
		t.Fatalf("Inspect must observe %s in go.mod, requirements=%#v", manifest.DefaultModule, in.Requirements)
	}
	if req.Path != manifest.DefaultModule || strings.EqualFold(req.Version, "latest") {
		t.Fatalf("requirement=%#v", req)
	}
}

func TestPackageAcquireEnablePreservesSelectedPin(t *testing.T) {
	requireGoTool(t)
	pkgDir := requirePackagesCheckout(t)
	root := writeIsolatedConsumer(t, pkgDir, nil)
	app := kernel.NewApplication(root)

	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{app: app, out: &buf, root: root}
	if err := cmd.Handle([]string{"session", "--enable", "--format=json"}); err != nil {
		t.Fatalf("package:acquire --enable: %v\n%s", err, buf.String())
	}
	before, err := acquire.Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	req, ok := before.Requirement(manifest.DefaultModule)
	if !ok {
		t.Fatal("expected packages module after acquire")
	}
	if strings.EqualFold(req.Version, "latest") {
		t.Fatalf("go.mod must not record latest: %#v", req)
	}
	pinned := req.Version

	if _, err := enablePackage(app, "session"); err != nil {
		t.Fatal(err)
	}
	if err := wireEnablement(app, "session"); err != nil {
		t.Fatal(err)
	}
	after, err := acquire.Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := after.Requirement(manifest.DefaultModule)
	if !ok || got.Version != pinned {
		t.Fatalf("enablement must preserve acquire pin %q, got %#v", pinned, got)
	}
}

func TestPackageAcquireE2EEnableThenBoot(t *testing.T) {
	requireGoTool(t)
	pkgDir := requirePackagesCheckout(t)
	root := writeIsolatedConsumer(t, pkgDir, nil)
	app := kernel.NewApplication(root)

	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{app: app, out: &buf, root: root}
	if err := cmd.Handle([]string{"session", "--enable", "--format=json"}); err != nil {
		t.Fatalf("package:acquire --enable: %v\n%s", err, buf.String())
	}
	var view acquireCLIView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Acquisition != acquireStatusSuccess || view.Enablement != enablementSuccess || !view.Enabled {
		t.Fatalf("expected acquisition+enablement success: %#v", view)
	}

	enabled, err := os.ReadFile(filepath.Join(root, "bootstrap", "enabled.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(enabled), `"session"`) {
		t.Fatalf("enablement must write session into bootstrap/enabled.go:\n%s", enabled)
	}
	addons, err := os.ReadFile(filepath.Join(root, "bootstrap", "addons.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(addons), `github.com/zatrano/packages/session`) {
		t.Fatalf("enablement must wire blank-import:\n%s", addons)
	}

	boot := filepath.Join(root, "cmd", "bootcheck")
	if err := os.MkdirAll(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	main := `package main

import (
	"fmt"
	"os"

	_ "example.com/acq-e2e/bootstrap"
	"github.com/zatrano/framework/v2/bootstrap"
)

func main() {
	app := bootstrap.App()
	if err := app.Bootstrap(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !app.Bound("session") {
		fmt.Fprintln(os.Stderr, "session not bound")
		os.Exit(2)
	}
}
`
	if err := os.WriteFile(filepath.Join(boot, "main.go"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "storage", "framework", "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	env := append(os.Environ(),
		"APP_KEY=zatrano-ci-test-key-not-prod!!!!",
		"APP_ENV=testing",
		"GOTOOLCHAIN=local",
		"DB_CONNECTION=",
		"DB_CONNECTIONS=",
	)
	// Last proxy-published tag. go.mod replace still binds this checkout (HEAD).
	get := exec.CommandContext(ctx, "go", "get", "github.com/zatrano/framework/v2@v2.0.27", "github.com/zatrano/packages/session")
	get.Dir = root
	get.Env = env
	if out, err := get.CombinedOutput(); err != nil {
		t.Fatalf("boot setup go get framework: %v\n%s", err, out)
	}
	c := exec.CommandContext(ctx, "go", "run", "./cmd/bootcheck")
	c.Dir = root
	c.Env = env
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("boot/runtime validation failed: %v\n%s", err, out)
	}
}

func TestPackageAcquireE2ESharedOfficialModule(t *testing.T) {
	requireGoTool(t)
	pkgDir := requirePackagesCheckout(t)
	root := writeIsolatedConsumer(t, pkgDir, nil)
	cmd := &PackageAcquireCommand{out: ioDiscard(), root: root}
	if err := cmd.Handle([]string{"session"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Handle([]string{"auth"}); err != nil {
		t.Fatal(err)
	}
	in, err := acquire.Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement(manifest.DefaultModule); !ok {
		t.Fatal("shared-module packages must pin github.com/zatrano/packages once")
	}
}

func TestPackageAcquireE2EHeavyMongoModule(t *testing.T) {
	requireGoTool(t)
	pkgDir := requirePackagesCheckout(t)
	mongoDir := filepath.Join(pkgDir, "mongo")
	if _, err := os.Stat(filepath.Join(mongoDir, "go.mod")); err != nil {
		t.Skip("mongo module checkout missing")
	}
	root := writeIsolatedConsumer(t, pkgDir, map[string]string{
		"github.com/zatrano/packages/mongo": mongoDir,
	})
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{out: &buf, root: root}
	if err := cmd.Handle([]string{"mongo", "--format=json"}); err != nil {
		t.Fatalf("heavy mongo acquire: %v\n%s", err, buf.String())
	}
	var view acquireCLIView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	want := "github.com/zatrano/packages/mongo@main"
	if len(view.GoGetArgs) != 1 || view.GoGetArgs[0] != want {
		t.Fatalf("GoGetArg=%v want %s", view.GoGetArgs, want)
	}
	in, err := acquire.Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Requirement("github.com/zatrano/packages/mongo"); !ok {
		t.Fatalf("Inspect must observe heavy module, got %#v", in.Requirements)
	}
}

func TestPackageAcquireE2ETaggedReleaseUsesGoGetArg(t *testing.T) {
	requireGoTool(t)
	pkgDir := requirePackagesCheckout(t)
	root := writeIsolatedConsumer(t, pkgDir, nil)
	idx := registry.Index{
		Schema: registry.SchemaV1,
		Packages: []registry.Package{{
			Name: "session", Import: manifest.DefaultModule + "/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
			Releases: []registry.Release{{Version: "v1.6.6"}},
		}},
	}
	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{out: &buf, root: root, index: &idx}
	if err := cmd.Handle([]string{"session@v1.6.6", "--format=json"}); err != nil {
		t.Fatalf("tagged acquire: %v\n%s", err, buf.String())
	}
	var view acquireCLIView
	if err := json.Unmarshal(buf.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	want := manifest.DefaultModule + "@v1.6.6"
	if len(view.GoGetArgs) != 1 || view.GoGetArgs[0] != want {
		t.Fatalf("tagged GoGetArg=%v want %s", view.GoGetArgs, want)
	}
	if view.Acquisition != acquireStatusSuccess {
		t.Fatalf("%#v", view)
	}
}

func TestPackageAcquireE2ERuntimeLifecycle(t *testing.T) {
	requireGoTool(t)
	pkgDir := requirePackagesCheckout(t)
	root := writeIsolatedConsumer(t, pkgDir, nil)
	app := kernel.NewApplication(root)

	var buf bytes.Buffer
	cmd := &PackageAcquireCommand{app: app, out: &buf, root: root}
	if err := cmd.Handle([]string{"session", "--enable", "--format=json"}); err != nil {
		t.Fatalf("package:acquire --enable: %v\n%s", err, buf.String())
	}

	boot := filepath.Join(root, "cmd", "runtimecheck")
	if err := os.MkdirAll(boot, 0o755); err != nil {
		t.Fatal(err)
	}
	main := `package main

import (
	"context"
	"fmt"
	"os"
	"time"

	_ "example.com/acq-e2e/bootstrap"
	"github.com/zatrano/framework/v2/bootstrap"
)

func main() {
	app := bootstrap.App()
	if err := app.Bootstrap(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !app.Bound("session") {
		fmt.Fprintln(os.Stderr, "session not bound")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.StartContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	if err := app.Stop(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(4)
	}
}
`
	if err := os.WriteFile(filepath.Join(boot, "main.go"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "storage", "framework", "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	env := append(os.Environ(),
		"APP_KEY=zatrano-ci-test-key-not-prod!!!!",
		"APP_ENV=testing",
		"GOTOOLCHAIN=local",
		"DB_CONNECTION=",
		"DB_CONNECTIONS=",
	)
	// Last proxy-published tag. go.mod replace still binds this checkout (HEAD).
	get := exec.CommandContext(ctx, "go", "get", "github.com/zatrano/framework/v2@v2.0.27", "github.com/zatrano/packages/session")
	get.Dir = root
	get.Env = env
	if out, err := get.CombinedOutput(); err != nil {
		t.Fatalf("runtime setup go get: %v\n%s", err, out)
	}
	c := exec.CommandContext(ctx, "go", "run", "./cmd/runtimecheck")
	c.Dir = root
	c.Env = env
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("runtime Start/Stop validation failed: %v\n%s", err, out)
	}
}
