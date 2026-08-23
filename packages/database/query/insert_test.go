package query

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestCompileInsertSQLNoReturningOnPlainInsert(t *testing.T) {
	b := &Builder{driver: "pgsql", table: "password_reset_tokens"}
	sqlStr, args, err := b.compileInsert(map[string]any{
		"email": "ada@example.com",
		"token": "hash",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToUpper(sqlStr), "RETURNING") {
		t.Fatalf("Insert must not use RETURNING: %s", sqlStr)
	}
	if strings.Contains(strings.ToUpper(sqlStr), "OUTPUT") {
		t.Fatalf("Insert must not use OUTPUT: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "$1") || !strings.Contains(sqlStr, "$2") {
		t.Fatalf("expected pgsql placeholders: %s", sqlStr)
	}
	if len(args) != 2 {
		t.Fatalf("args=%d", len(args))
	}
}

func TestCompileInsertGetIDSQLReturning(t *testing.T) {
	pg := &Builder{driver: "pgsql", table: "users"}
	sqlStr, _, err := pg.compileInsert(map[string]any{"name": "Ada"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sqlStr, "RETURNING id") {
		t.Fatalf("InsertGetID pgsql missing RETURNING id: %s", sqlStr)
	}

	mssql := &Builder{driver: "sqlserver", table: "users"}
	sqlStr, _, err = mssql.compileInsert(map[string]any{"name": "Ada"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sqlStr, "OUTPUT INSERTED.id") {
		t.Fatalf("InsertGetID mssql missing OUTPUT: %s", sqlStr)
	}

	sqlite := &Builder{driver: "sqlite", table: "users"}
	sqlStr, _, err = sqlite.compileInsert(map[string]any{"name": "Ada"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sqlStr, "RETURNING") || strings.Contains(sqlStr, "OUTPUT") {
		t.Fatalf("sqlite InsertGetID should be plain INSERT: %s", sqlStr)
	}
}

func TestInsertIntoTableWithoutID(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE password_reset_tokens (
		email TEXT NOT NULL,
		token TEXT NOT NULL,
		created_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	n, err := New(db, "sqlite", "password_reset_tokens").Insert(map[string]any{
		"email":      "ada@example.com",
		"token":      "hashed",
		"created_at": time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Insert without id column: %v", err)
	}
	_ = n
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM password_reset_tokens`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestInsertGetIDReturnsPrimaryKey(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	id, err := New(db, "sqlite", "items").InsertGetID(map[string]any{"name": "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}
}
