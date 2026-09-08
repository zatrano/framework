package tests

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/distribution/registry"
)

func TestMeetsFrameworkMinAgreement(t *testing.T) {
	cases := []struct {
		have string
		min  string
	}{
		{"2.0.1", ""},
		{"2.0.1", "2.0.1"},
		{"v2.1.0", "2.0.1"},
		{"2.0.0", "2.0.1"},
		{"", "2.0.1"},
		{"v2.0.27", "2.0.27"},
		{"2.0.27", "v2.0.28"},
		{"2.0.27-dev", "2.0.27"},
		{"main", "2.0.1"},
		{"2.0.4", "2.0.0"},
		{"2.0.4", "2.0.4"},
		{"1.9.0", "2.0.0"},
	}
	for _, c := range cases {
		a := addons.MeetsFrameworkMin(c.have, c.min)
		r := registry.MeetsFrameworkMin(c.have, c.min)
		if a != r {
			t.Errorf("have=%q min=%q addons=%v registry=%v", c.have, c.min, a, r)
		}
	}
}

func TestMeetsFrameworkMinMultiplePackageMinimums(t *testing.T) {
	have := "2.0.3"
	mins := []string{"", "2.0.0", "2.0.1", "2.0.3", "2.0.4"}
	want := []bool{true, true, true, true, false}
	for i, min := range mins {
		if addons.MeetsFrameworkMin(have, min) != want[i] || registry.MeetsFrameworkMin(have, min) != want[i] {
			t.Errorf("min=%q: addons=%v registry=%v want %v", min, addons.MeetsFrameworkMin(have, min), registry.MeetsFrameworkMin(have, min), want[i])
		}
	}
}

func TestRuntimeDoesNotValidateFrameworkMin(t *testing.T) {
	root := moduleRoot(t)
	files := []string{
		filepath.Join(root, "kernel", "application.go"),
		filepath.Join(root, "bootstrap", "app.go"),
		filepath.Join(root, "bootstrap", "enablement.go"),
	}
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), "MeetsFrameworkMin") {
			t.Errorf("%s must not validate framework_min at runtime", filepath.Base(path))
		}
	}
}

func TestAcquireDoesNotRevalidateFrameworkMin(t *testing.T) {
	root := moduleRoot(t)
	err := filepath.WalkDir(filepath.Join(root, "distribution", "acquire"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(body), "MeetsFrameworkMin") {
			t.Errorf("%s must not revalidate framework_min", filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoctorReportsFrameworkMinViaAddons(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(moduleRoot(t), "console", "package_doctor.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	if !strings.Contains(src, "addons.MeetsFrameworkMin") {
		t.Fatal("package:doctor must report compatibility via addons.MeetsFrameworkMin")
	}
	if strings.Contains(src, "registry.MeetsFrameworkMin") {
		t.Fatal("do not add a third compatibility implementation in console")
	}
}

func TestConsoleHasNoMeetsFrameworkMinImplementation(t *testing.T) {
	err := filepath.WalkDir(filepath.Join(moduleRoot(t), "console"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(body), "func MeetsFrameworkMin") {
			t.Errorf("%s must not implement MeetsFrameworkMin", filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
