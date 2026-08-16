package database

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
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
	SSLMode  string // pgsql / oracle
	Service  string // oracle service name
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

// ConnectionNames returns configured connection names.
func (m *Manager) ConnectionNames() []string {
	out := make([]string, 0, len(m.config.Connections))
	for name := range m.config.Connections {
		out = append(out, name)
	}
	return out
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

// Transaction runs fn inside a database transaction on the given (or default) connection.
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
	driver, dsn, err := BuildDSNFor(cfg, m.basePath)
	if err != nil {
		return nil, err
	}
	if !sqlDriverRegistered(driver) {
		canon := NormalizeDriverName(cfg.Driver)
		return nil, fmt.Errorf("SQL driver %q is not linked — run: go run ./cmd/zatrano db:setup --drivers=%s", driver, canon)
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

func sqlDriverRegistered(name string) bool {
	for _, d := range sql.Drivers() {
		if d == name {
			return true
		}
	}
	return false
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

// BuildDSN exposes DSN building for tests.
func BuildDSN(cfg ConnectionConfig, basePath string) (string, string, error) {
	return BuildDSNFor(cfg, basePath)
}
