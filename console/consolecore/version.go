package consolecore

import (
	"os"
	"path/filepath"
	"strings"
)

// CurrentRelease is the fallback product version when VERSION cannot be read.
const CurrentRelease = "2.6.1"

// ProductVersion reads VERSION from the framework module root.
func ProductVersion() string {
	root, err := FrameworkModuleRoot()
	if err != nil {
		return CurrentRelease
	}
	return ProductVersionAt(root)
}

// ProductVersionAt reads VERSION from root.
func ProductVersionAt(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return CurrentRelease
	}
	v := strings.TrimSpace(string(raw))
	if v == "" {
		return CurrentRelease
	}
	return v
}
