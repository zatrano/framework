package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteExclusive creates path with body; it fails if the file already exists.
func WriteExclusive(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("already exists: %s", path)
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// WriteFile creates or replaces path with body.
func WriteFile(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}
