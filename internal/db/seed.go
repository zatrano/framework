package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// RunSeeds executes every *.sql file in dir in lexical order inside a single transaction.
func RunSeeds(databaseURL, dir string) error {
	if strings.TrimSpace(databaseURL) == "" {
		return fmt.Errorf("database URL is empty (set DATABASE_URL)")
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("seeds dir: %w", err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("seeds path is not a directory: %s", dir)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return nil
	}
	sort.Strings(matches)

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("gorm open: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, f := range matches {
		b, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := tx.Exec(string(b)); err != nil {
			return fmt.Errorf("exec %s: %w", f, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

