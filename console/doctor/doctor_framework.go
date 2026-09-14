package doctor

import (
	"os"
	"path/filepath"
)

func isFrameworkRepo(root string) bool {
	mod, err := modulePath(root)
	return err == nil && mod == "github.com/zatrano/framework/v2"
}

func checkFrameworkRepoLayout(root string) ([]Finding, error) {
	banned := []string{"storage", "app", "public", "views", "routes"}
	var out []Finding
	for _, name := range banned {
		st, err := os.Stat(filepath.Join(root, name))
		if err != nil || !st.IsDir() {
			continue
		}
		out = append(out, Finding{
			Rule:     "FW-ROOT-001",
			Check:    "layout",
			Severity: "error",
			File:     name,
			Found:    "app-shaped directory " + name + "/ at the framework module root",
			Why:      "The framework repository is not a consumer application.",
			How:      "Delete " + name + "/ from the framework root. Runtime dirs belong in generated apps, tests/fixtures/, or t.TempDir().",
		})
	}
	return out, nil
}
