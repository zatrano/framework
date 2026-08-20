package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/database"
	"github.com/zatrano/framework/packages/safepath"
)

// Config describes the database connection to back up and where files go.
type Config struct {
	Driver   string
	Host     string
	Port     string
	Database string // sqlite file path or DB name
	Username string
	Password string
	Charset  string
	SSLMode  string
	Service  string // oracle
	URI      string // mongo
	Dir      string // backup directory
	BasePath string // app base for relative sqlite paths
}

// Manager creates and restores database backups (SQLite file copy or native CLI tools).
type Manager struct {
	cfg Config
}

// New creates a SQLite file-copy manager (legacy helper).
// source is the database file path; dir is where backups are stored.
func New(source, dir string) *Manager {
	return NewManager(Config{
		Driver:   "sqlite",
		Database: source,
		Dir:      dir,
	})
}

// NewManager creates a backup manager for any supported driver.
func NewManager(cfg Config) *Manager {
	cfg.Driver = database.NormalizeDriverName(cfg.Driver)
	if cfg.Driver == "" {
		cfg.Driver = "sqlite"
	}
	return &Manager{cfg: cfg}
}

// Dir returns the backup directory.
func (m *Manager) Dir() string { return m.cfg.Dir }

// Source returns the sqlite source path (empty for non-sqlite).
func (m *Manager) Source() string {
	if m.cfg.Driver != "sqlite" {
		return ""
	}
	return m.sqlitePath()
}

// Driver returns the normalized driver name.
func (m *Manager) Driver() string { return m.cfg.Driver }

// Create writes a timestamped backup file; optional label is sanitized into the name.
func (m *Manager) Create(label ...string) (string, error) {
	if err := ensureBackupDir(m.cfg.Dir); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102_150405")
	safe := ""
	if len(label) > 0 && label[0] != "" {
		safe = "_" + sanitize(label[0])
	}
	ext := extensionFor(m.cfg.Driver)
	dest := filepath.Join(m.cfg.Dir, "backup_"+stamp+safe+ext)

	var err error
	switch m.cfg.Driver {
	case "sqlite":
		err = m.createSQLite(dest)
	case "mysql":
		err = m.createMySQL(dest)
	case "pgsql":
		err = m.createPostgres(dest)
	case "mssql":
		err = m.createMSSQL(dest)
	case "oracle":
		err = m.createOracle(dest)
	case "mongo":
		err = m.createMongo(dest)
	default:
		err = fmt.Errorf("backup: unsupported driver %q", m.cfg.Driver)
	}
	if err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	_ = os.Chmod(dest, 0o600)
	_ = writeMeta(dest, m.cfg.Driver)
	return dest, nil
}

// List returns backup files newest first.
func (m *Manager) List() ([]string, error) {
	entries, err := os.ReadDir(m.cfg.Dir)
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
		if strings.HasPrefix(name, "backup_") && isBackupFile(name) {
			files = append(files, filepath.Join(m.cfg.Dir, name))
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i] > files[j] })
	return files, nil
}

// Restore replaces / loads the database from the given backup file name or path.
// Paths must resolve inside the configured backup directory (absolute paths outside are rejected).
func (m *Manager) Restore(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("backup path required")
	}
	resolved, err := resolveBackupPath(m.cfg.Dir, backupPath)
	if err != nil {
		return err
	}
	backupPath = resolved
	if _, err := os.Stat(backupPath); err != nil {
		return err
	}
	driver := m.cfg.Driver
	if meta := readMeta(backupPath); meta != "" {
		driver = database.NormalizeDriverName(meta)
	} else {
		driver = driverFromExt(backupPath, driver)
	}
	switch driver {
	case "sqlite":
		return m.restoreSQLite(backupPath)
	case "mysql":
		return m.restoreMySQL(backupPath)
	case "pgsql":
		return m.restorePostgres(backupPath)
	case "mssql":
		return m.restoreMSSQL(backupPath)
	case "oracle":
		return m.restoreOracle(backupPath)
	case "mongo":
		return m.restoreMongo(backupPath)
	default:
		return fmt.Errorf("backup: unsupported driver %q for restore", driver)
	}
}

func ensureBackupDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	_ = os.Chmod(dir, 0o700)
	return nil
}

// resolveBackupPath confines restore targets to the backup directory.
func resolveBackupPath(dir, backupPath string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("backup: directory not configured")
	}
	if filepath.IsAbs(backupPath) {
		if !safepath.Under(dir, backupPath) {
			return "", fmt.Errorf("backup: path escapes backup directory")
		}
		return backupPath, nil
	}
	return safepath.Resolve(dir, backupPath)
}

func extensionFor(driver string) string {
	switch database.NormalizeDriverName(driver) {
	case "sqlite":
		return ".sqlite"
	case "mysql":
		return ".sql"
	case "pgsql":
		return ".dump"
	case "mssql":
		return ".bacpac"
	case "oracle":
		return ".dmp"
	case "mongo":
		return ".archive"
	default:
		return ".bak"
	}
}

func isBackupFile(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range []string{".sqlite", ".sql", ".dump", ".bacpac", ".dmp", ".archive", ".bak"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func driverFromExt(path, fallback string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".sqlite"), strings.HasSuffix(lower, ".bak"):
		return "sqlite"
	case strings.HasSuffix(lower, ".sql"):
		return "mysql"
	case strings.HasSuffix(lower, ".dump"):
		return "pgsql"
	case strings.HasSuffix(lower, ".bacpac"):
		return "mssql"
	case strings.HasSuffix(lower, ".dmp"):
		return "oracle"
	case strings.HasSuffix(lower, ".archive"):
		return "mongo"
	default:
		return fallback
	}
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
