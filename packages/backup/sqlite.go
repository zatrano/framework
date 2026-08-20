package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (m *Manager) sqlitePath() string {
	path := m.cfg.Database
	if path == "" {
		path = "database/database.sqlite"
	}
	if !filepath.IsAbs(path) {
		base := m.cfg.BasePath
		if base == "" {
			base = "."
		}
		path = filepath.Join(base, path)
	}
	return path
}

func (m *Manager) createSQLite(dest string) error {
	src := m.sqlitePath()
	if src == "" {
		return fmt.Errorf("backup source is empty")
	}
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("backup source: %w", err)
	}
	return copyFile(src, dest)
}

func (m *Manager) restoreSQLite(backupPath string) error {
	src := m.sqlitePath()
	if src == "" {
		return fmt.Errorf("backup source is empty")
	}
	if err := os.MkdirAll(filepath.Dir(src), 0o700); err != nil {
		return err
	}
	_ = os.Chmod(filepath.Dir(src), 0o700)
	if _, err := os.Stat(src); err == nil {
		_ = copyFile(src, src+".pre-restore")
	}
	return copyFile(backupPath, src)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	_ = os.Chmod(dst, 0o600)
	return nil
}
