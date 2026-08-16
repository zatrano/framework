package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	_ "modernc.org/sqlite"
)

// Config holds database connection settings.
type Config struct {
	Default     string
	Connections map[string]ConnectionConfig
}

// ConnectionConfig describes a single connection.
type ConnectionConfig struct {
	Driver   string
	Host     string
	Port     string
	Database string
	Username string
	Password string
	Charset  string
	SSLMode  string // pgsql: disable|require|verify-full|…
}

// Manager manages database connections.
type Manager struct {
	mu          sync.RWMutex
	config      Config
	connections map[string]*sql.DB
	basePath    string
}

// NewManager creates a database manager.
func NewManager(cfg Config, basePath string) *Manager {
	return &Manager{
		config:      cfg,
		connections: make(map[string]*sql.DB),
		basePath:    basePath,
	}
}

// Connection returns a named connection or the default one.
func (m *Manager) Connection(name ...string) (*sql.DB, error) {
	connName := m.config.Default
	if len(name) > 0 && name[0] != "" {
		connName = name[0]
	}

	m.mu.RLock()
	if db, ok := m.connections[connName]; ok {
		m.mu.RUnlock()
		return db, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if db, ok := m.connections[connName]; ok {
		return db, nil
	}

	cfg, ok := m.config.Connections[connName]
	if !ok {
		return nil, fmt.Errorf("database connection [%s] not configured", connName)
	}

	db, err := m.open(cfg)
	if err != nil {
		return nil, err
	}
	m.connections[connName] = db
	return db, nil
}

// DB returns the default connection.
func (m *Manager) DB() (*sql.DB, error) {
	return m.Connection()
}

// Transaction runs fn inside a database transaction.
// Commits on nil error; rolls back otherwise (including panics).
func (m *Manager) Transaction(fn func(tx *sql.Tx) error, name ...string) (err error) {
	db, err := m.Connection(name...)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Close closes all connections.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var first error
	for name, db := range m.connections {
		if err := db.Close(); err != nil && first == nil {
			first = err
		}
		delete(m.connections, name)
	}
	return first
}

func (m *Manager) open(cfg ConnectionConfig) (*sql.DB, error) {
	driver, dsn, err := m.dsn(cfg)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (m *Manager) dsn(cfg ConnectionConfig) (string, string, error) {
	switch strings.ToLower(cfg.Driver) {
	case "sqlite", "sqlite3":
		path := cfg.Database
		if path == "" {
			path = "database/database.sqlite"
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(m.basePath, path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", "", err
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			f, createErr := os.Create(path)
			if createErr != nil {
				return "", "", createErr
			}
			_ = f.Close()
		}
		return "sqlite", path, nil
	case "mysql":
		charset := cfg.Charset
		if charset == "" {
			charset = "utf8mb4"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=true&loc=Local",
			cfg.Username, cfg.Password, cfg.Host, defaultPort(cfg.Port, "3306"), cfg.Database, charset)
		return "mysql", dsn, nil
	case "pgsql", "postgres", "postgresql":
		ssl := cfg.SSLMode
		if ssl == "" {
			ssl = "disable"
		}
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, defaultPort(cfg.Port, "5432"), cfg.Username, cfg.Password, cfg.Database, ssl)
		return "postgres", dsn, nil
	case "mssql", "sqlserver":
		query := url.Values{}
		query.Set("database", cfg.Database)
		u := &url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(cfg.Username, cfg.Password),
			Host:     fmt.Sprintf("%s:%s", cfg.Host, defaultPort(cfg.Port, "1433")),
			RawQuery: query.Encode(),
		}
		return "sqlserver", u.String(), nil
	default:
		return "", "", fmt.Errorf("unsupported database driver [%s]", cfg.Driver)
	}
}

// DriverName returns the driver for a connection.
func (m *Manager) DriverName(name ...string) (string, error) {
	connName := m.config.Default
	if len(name) > 0 && name[0] != "" {
		connName = name[0]
	}
	cfg, ok := m.config.Connections[connName]
	if !ok {
		return "", fmt.Errorf("database connection [%s] not configured", connName)
	}
	return strings.ToLower(cfg.Driver), nil
}

// BuildDSN exposes DSN building for tests (driver name, dsn, error).
func BuildDSN(cfg ConnectionConfig, basePath string) (string, string, error) {
	m := &Manager{basePath: basePath}
	return m.dsn(cfg)
}

func defaultPort(port, fallback string) string {
	if port == "" {
		return fallback
	}
	return port
}
