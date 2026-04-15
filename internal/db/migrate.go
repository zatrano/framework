package db

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrateUp applies SQL migrations from dir (filesystem path) using databaseURL (postgres DSN).
func MigrateUp(databaseURL, dir string, steps int) (version uint, dirty bool, err error) {
	if strings.TrimSpace(databaseURL) == "" {
		return 0, false, fmt.Errorf("database URL is empty (set DATABASE_URL)")
	}
	src, err := fileSourceURL(dir)
	if err != nil {
		return 0, false, err
	}
	m, err := migrate.New(src, databaseURL)
	if err != nil {
		return 0, false, fmt.Errorf("migrate init: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if steps > 0 {
		if err := m.Steps(steps); err != nil && err != migrate.ErrNoChange {
			v, d, _ := m.Version()
			return v, d, err
		}
	} else {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			v, d, _ := m.Version()
			return v, d, err
		}
	}
	return m.Version()
}

// MigrateDown rolls back migrations (one step by default when steps is 0, use steps for more).
func MigrateDown(databaseURL, dir string, steps int) (version uint, dirty bool, err error) {
	if strings.TrimSpace(databaseURL) == "" {
		return 0, false, fmt.Errorf("database URL is empty (set DATABASE_URL)")
	}
	src, err := fileSourceURL(dir)
	if err != nil {
		return 0, false, err
	}
	m, err := migrate.New(src, databaseURL)
	if err != nil {
		return 0, false, fmt.Errorf("migrate init: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if steps <= 0 {
		steps = 1
	}
	if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
		v, d, _ := m.Version()
		return v, d, err
	}
	return m.Version()
}

func fileSourceURL(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	abs = filepath.ToSlash(filepath.Clean(abs))
	if vol := filepath.VolumeName(abs); vol != "" {
		// C:/path -> /C:/path for file:// URLs on Windows
		abs = "/" + abs
	}
	return "file://" + abs, nil
}

