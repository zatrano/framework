package console

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/manifest"
)

var (
	reRequires     = regexp.MustCompile(`Requires:\s*\[\]string\{([^}]+)\}`)
	reFieldCLI     = regexp.MustCompile(`\n\s+CLI:\s`)
	reFieldFactory = regexp.MustCompile(`\n\s+Factory:\s`)
	reFieldKey     = regexp.MustCompile(`\n\s+Key:\s*"([a-z0-9.]+)"`)
)

func TestEcosystemCatalogDerivesValidManifests(t *testing.T) {
	hints := loadRegisterHints(t)
	if len(ecosystemCatalog) < 80 {
		t.Fatalf("expected full ecosystem catalog, got %d", len(ecosystemCatalog))
	}
	for _, p := range ecosystemCatalog {
		in := manifest.Input{
			Name:        p.Name,
			Kind:        string(p.EffectiveKind()),
			Layer:       string(p.Layer),
			Description: p.Description,
			Heavy:       p.Heavy,
		}
		if h, ok := hints[p.Name]; ok {
			in.Factory = h.factory
			in.CLI = h.cli
			in.Requires = h.requires
			in.Key = h.key
		}
		d := manifest.Derive(in)
		if err := manifest.Validate(d); err != nil {
			t.Errorf("%s: %v", p.Name, err)
		}
		if p.Layer == kernel.LayerPrimitive {
			t.Errorf("%s: ecosystem catalog must not include primitives", p.Name)
		}
	}
}

type registerHint struct {
	factory  bool
	cli      bool
	requires []string
	key      string
}

func loadRegisterHints(t *testing.T) map[string]registerHint {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "packages")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Log("packages checkout not beside framework; deriving catalog identity only")
		return map[string]registerHint{}
	}
	out := map[string]registerHint{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		src := string(body)
		if !strings.Contains(src, "addons.Register") {
			return nil
		}
		for _, m := range reAddonName.FindAllStringSubmatch(src, -1) {
			if len(m) != 2 {
				continue
			}
			h := registerHint{
				factory: reFieldFactory.MatchString(src),
				cli:     reFieldCLI.MatchString(src),
			}
			if km := reFieldKey.FindStringSubmatch(src); len(km) == 2 {
				h.key = km[1]
			}
			if rm := reRequires.FindStringSubmatch(src); len(rm) == 2 {
				for _, part := range strings.Split(rm[1], ",") {
					part = strings.Trim(strings.TrimSpace(part), `"`)
					if part != "" {
						h.requires = append(h.requires, part)
					}
				}
			}
			out[m[1]] = h
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 40 {
		t.Fatalf("expected register hints, got %d", len(out))
	}
	auth := out["auth"]
	if len(auth.requires) != 3 {
		t.Fatalf("auth requires=%v", auth.requires)
	}
	if !out["factory"].cli || out["factory"].factory {
		t.Fatalf("factory should be CLI-only, got %+v", out["factory"])
	}
	return out
}
