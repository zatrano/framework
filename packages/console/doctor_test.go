package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorMisplacedRoute(t *testing.T) {
	root := filepath.Join("testdata", "doctor", "misplaced_route")
	findings, err := RunDoctor(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDoctorCheck(findings, "routes", "app/services/oops.go") {
		t.Fatalf("expected routes finding in app/services/oops.go, got:\n%s", FormatDoctorText(root, findings))
	}
}

func TestDoctorConcreteLeak(t *testing.T) {
	root := filepath.Join("testdata", "doctor", "concrete_leak")
	findings, err := RunDoctor(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDoctorCheck(findings, "concrete", "app/http/controllers/web/home.go") {
		t.Fatalf("expected concrete finding, got:\n%s", FormatDoctorText(root, findings))
	}
}

func TestDoctorLegacyLayout(t *testing.T) {
	root := filepath.Join("testdata", "doctor", "legacy_layout")
	findings, err := RunDoctor(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDoctorCheck(findings, "layout", "application") {
		t.Fatalf("expected layout finding for application/, got:\n%s", FormatDoctorText(root, findings))
	}
}

func TestDoctorMissingProviderBoot(t *testing.T) {
	root := filepath.Join("testdata", "doctor", "missing_boot")
	findings, err := RunDoctor(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDoctorCheck(findings, "providers", "app/providers/broken.go") {
		t.Fatalf("expected providers finding, got:\n%s", FormatDoctorText(root, findings))
	}
}

func TestDoctorAllowsRoutePrimitives(t *testing.T) {
	root := t.TempDir()
	writeDoctorFile(t, root, filepath.Join("app", "routes", "web", "web.go"), `package web

import "github.com/zatrano/framework/packages/routing"

func init() {
	routing.RegisterWeb(func(r *routing.Router) {
		r.Get("/", nil)
	})
}
`)
	writeDoctorFile(t, root, filepath.Join("app", "providers", "route_service_provider.go"), `package providers

import "github.com/zatrano/framework/packages/routing"

func Boot() {
	routing.ApplyWeb(nil)
}
`)
	// Satisfy layout enough that we only care about routes/concrete.
	findings, err := RunDoctor(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.Check == "routes" || f.Check == "concrete" {
			t.Fatalf("route primitive should be allowed: %+v\n%s", f, FormatDoctorText(root, findings))
		}
	}
}

func TestDoctorParseContractConcretes(t *testing.T) {
	fw, err := frameworkModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	got, err := collectContractConcretes(fw)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["github.com/zatrano/framework/packages/routing"]; !ok {
		t.Fatalf("routing missing: %v", got)
	}
	if _, ok := got["github.com/zatrano/framework/packages/config"]; !ok {
		t.Fatalf("config missing: %v", got)
	}
}

func TestDoctorCleanStarterHasNoRouteOrConcrete(t *testing.T) {
	fw, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "demo")
	cmd := &NewCommand{}
	if err := cmd.Handle([]string{dest, "--module", "example.com/demo", "--replace", fw}); err != nil {
		t.Fatal(err)
	}
	findings, err := RunDoctor(dest)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.Check == "routes" || f.Check == "concrete" || f.Check == "providers" {
			t.Fatalf("starter should be clean for %s: %+v\n%s", f.Check, f, FormatDoctorText(dest, findings))
		}
	}
}

func hasDoctorCheck(findings []Finding, check, file string) bool {
	file = filepath.ToSlash(file)
	for _, f := range findings {
		if f.Check == check && (f.File == file || strings.Contains(filepath.ToSlash(f.File), file)) {
			return true
		}
	}
	return false
}

func writeDoctorFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
