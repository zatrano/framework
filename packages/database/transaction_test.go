package database_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/database"
)

func TestTransactionCommitAndRollback(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tx.sqlite")
	mgr := database.NewManager(database.Config{
		Default: "sqlite",
		Connections: map[string]database.ConnectionConfig{
			"sqlite": {Driver: "sqlite", Database: dbPath},
		},
	}, dir)

	if err := mgr.Transaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
			return err
		}
		_, err := tx.Exec(`INSERT INTO items (name) VALUES ('a')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	db, err := mgr.DB()
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}

	err = mgr.Transaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO items (name) VALUES ('b')`); err != nil {
			return err
		}
		return fmt.Errorf("boom")
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("after rollback count=%d err=%v", count, err)
	}
	_ = mgr.Close()
}
