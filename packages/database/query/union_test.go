package query_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/database/query"
)

func TestUnionForPageAndClone(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE items (id INTEGER PRIMARY KEY, title TEXT, kind TEXT);
		INSERT INTO items (id, title, kind) VALUES
			(1, 'a', 'x'), (2, 'b', 'x'), (3, 'c', 'y'), (4, 'd', 'y');
	`)
	if err != nil {
		t.Fatal(err)
	}

	left := query.New(db, "sqlite", "items").Where("kind", "x")
	right := query.New(db, "sqlite", "items").Where("kind", "y")
	sqlStr, _ := left.Clone().Union(right).OrderBy("id").ToSQL()
	if !strings.Contains(sqlStr, "UNION") || !strings.Contains(sqlStr, "ORDER BY id") {
		t.Fatalf("union sql=%s", sqlStr)
	}

	rows, err := query.New(db, "sqlite", "items").
		Where("kind", "x").
		Union(query.New(db, "sqlite", "items").Where("kind", "y")).
		OrderBy("id").
		Get()
	if err != nil || len(rows) != 4 {
		t.Fatalf("union get: len=%d err=%v", len(rows), err)
	}

	rows, err = query.New(db, "sqlite", "items").
		Where("kind", "x").
		UnionAll(query.New(db, "sqlite", "items").Where("kind", "x")).
		Get()
	if err != nil || len(rows) != 4 {
		t.Fatalf("union all: len=%d err=%v", len(rows), err)
	}

	page, err := query.New(db, "sqlite", "items").OrderBy("id").ForPage(2, 2).Get()
	if err != nil || len(page) != 2 {
		t.Fatalf("for page: len=%d err=%v", len(page), err)
	}
	if id, _ := page[0]["id"].(int64); id != 3 {
		t.Fatalf("expected id 3, got %#v", page[0]["id"])
	}
}
