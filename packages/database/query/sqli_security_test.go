package query_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/database/query"
)

func TestSQLInjectionOrderByRejected(t *testing.T) {
	db, err := sql.Open("sqlite", "file:sqli_order?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, title TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO items(title) VALUES ('a'), ('b')`); err != nil {
		t.Fatal(err)
	}

	_, err = query.New(db, "sqlite", "items").OrderBy("id; DROP TABLE items--").Get()
	if err == nil {
		t.Fatal("expected order by injection to fail")
	}
	if !strings.Contains(err.Error(), "order by") {
		t.Fatalf("unexpected err: %v", err)
	}

	_, err = query.New(db, "sqlite", "items").OrderBy("id", "asc; delete from items").Get()
	if err != nil {
		t.Fatalf("direction should coerce to asc, got %v", err)
	}
	sqlStr, _ := query.New(db, "sqlite", "items").OrderBy("id", "DESC; DROP").ToSQL()
	if strings.Contains(strings.ToLower(sqlStr), "drop") {
		t.Fatalf("direction injection leaked into SQL: %s", sqlStr)
	}
	if !strings.Contains(strings.ToLower(sqlStr), "order by id asc") {
		t.Fatalf("junk direction must fall back to asc, got %s", sqlStr)
	}
	sqlDesc, _ := query.New(db, "sqlite", "items").OrderBy("id", "DESC").ToSQL()
	if !strings.Contains(strings.ToLower(sqlDesc), "order by id desc") {
		t.Fatalf("plain DESC should work, got %s", sqlDesc)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("table should remain intact count=%d err=%v", n, err)
	}
}

func TestSQLInjectionWhereOperatorRejected(t *testing.T) {
	db, err := sql.Open("sqlite", "file:sqli_where?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, title TEXT)`); err != nil {
		t.Fatal(err)
	}

	_, err = query.New(db, "sqlite", "items").Where("title", "=; DROP TABLE items--", "x").Get()
	if err == nil {
		t.Fatal("expected bad operator to fail")
	}
	_, err = query.New(db, "sqlite", "items").Where("title); DROP TABLE items--", "x").Get()
	if err == nil {
		t.Fatal("expected bad column to fail")
	}
}
