package console

import (
	"os"
	"path/filepath"
	"strings"
)

// currentRelease is the fallback product version when VERSION cannot be read
// (for example a generated app with no VERSION file). Keep in sync with the
// repository VERSION file.
const currentRelease = "2.2.0"

func productVersion() string {
	root, err := frameworkModuleRoot()
	if err != nil {
		return currentRelease
	}
	return productVersionAt(root)
}

func productVersionAt(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return currentRelease
	}
	v := strings.TrimSpace(string(raw))
	if v == "" {
		return currentRelease
	}
	return v
}
