package files

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomic writes data to path via a temp file + rename.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	if perm == 0 {
		perm = 0o644
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".zatrano-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("files: rename: %w", err)
	}
	cleanup = false
	return nil
}

// WriteStringAtomic is WriteAtomic for strings.
func WriteStringAtomic(path, content string, perm os.FileMode) error {
	return WriteAtomic(path, []byte(content), perm)
}

// TempDir creates a temporary directory under base (or os temp).
func TempDir(base, pattern string) (string, error) {
	if pattern == "" {
		pattern = "zatrano-*"
	}
	if base == "" {
		return os.MkdirTemp("", pattern)
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, pattern)
}

// Exists reports whether path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
