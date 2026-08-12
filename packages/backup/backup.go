package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Manager creates and restores file-based backups (SQLite-friendly).
type Manager struct {
	source string
	dir    string
}

// New creates a backup manager.
// source is the database file path; dir is where backups are stored.
func New(source, dir string) *Manager {
	return &Manager{source: source, dir: dir}
}

// Dir returns the backup directory.
func (m *Manager) Dir() string { return m.dir }

// Source returns the source database path.
func (m *Manager) Source() string { return m.source }

// Create copies the source database into a timestamped backup file.
func (m *Manager) Create(label ...string) (string, error) {
	if m.source == "" {
		return "", fmt.Errorf("backup source is empty")
	}
	if _, err := os.Stat(m.source); err != nil {
		return "", fmt.Errorf("backup source: %w", err)
	}
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102_150405")
	name := "backup_" + stamp + ".sqlite"
	if len(label) > 0 && label[0] != "" {
		safe := sanitize(label[0])
		name = "backup_" + stamp + "_" + safe + ".sqlite"
	}
	dest := filepath.Join(m.dir, name)
	if err := copyFile(m.source, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// List returns backup files newest first.
func (m *Manager) List() ([]string, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "backup_") && (strings.HasSuffix(name, ".sqlite") || strings.HasSuffix(name, ".bak")) {
			files = append(files, filepath.Join(m.dir, name))
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i] > files[j] })
	return files, nil
}

// Restore replaces the source database with the given backup file.
func (m *Manager) Restore(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("backup path required")
	}
	if !filepath.IsAbs(backupPath) {
		backupPath = filepath.Join(m.dir, backupPath)
	}
	if _, err := os.Stat(backupPath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.source), 0o755); err != nil {
		return err
	}
	// Safety copy of current DB if it exists.
	if _, err := os.Stat(m.source); err == nil {
		safety := m.source + ".pre-restore"
		_ = copyFile(m.source, safety)
	}
	return copyFile(backupPath, m.source)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func sanitize(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "_")
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "manual"
	}
	return out
}
