// Package safepath prevents filesystem path traversal outside a configured root.
package safepath

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Under reports whether candidate resolves inside root (inclusive).
func Under(root, candidate string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absCand, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absCand)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Resolve joins root with a user-supplied relative path and rejects escapes.
// Absolute user paths, ".." segments that leave root, and null bytes are rejected.
func Resolve(root, userPath string) (string, error) {
	if strings.ContainsRune(userPath, 0) {
		return "", fmt.Errorf("path contains null byte")
	}
	slash := filepath.ToSlash(userPath)
	slash = strings.TrimPrefix(slash, "/")
	if slash == "" || slash == "." {
		full, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		return full, nil
	}
	// Reject Windows drive / UNC style after ToSlash normalization.
	if strings.Contains(slash, ":") || strings.HasPrefix(slash, "//") {
		return "", fmt.Errorf("absolute path rejected")
	}
	rel := filepath.FromSlash(slash)
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root")
	}
	full := filepath.Join(root, rel)
	if !Under(root, full) {
		return "", fmt.Errorf("path escapes root")
	}
	return full, nil
}
