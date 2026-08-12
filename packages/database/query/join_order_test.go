package query_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/database/query"
)

func TestJoinOrderHavingHelpers(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE books (id INTEGER PRIMARY KEY, author_id INTEGER, title TEXT, copies INTEGER);
		INSERT INTO authors (id, name) VALUES (1, 'Ada'), (2, 'Grace');
		INSERT INTO books (id, author_id, title, copies) VALUES
			(1, 1, 'A', 2), (2, 1, 'B', 5), (3, 2, 'C', 1);
	`)
	if err != nil {
		t.Fatal(err)
	}

	sqlStr, _ := query.New(db, "sqlite", "books").
		Join("authors", "authors.id", "=", "books.author_id").
		OrderByDesc("books.id").
		ToSQL()
	if !strings.Contains(sqlStr, "INNER JOIN authors") || !strings.Contains(sqlStr, "ORDER BY books.id desc") {
		t.Fatalf("join/order sql=%s", sqlStr)
	}

	sqlStr, _ = query.New(db, "sqlite", "books").RightJoin("authors", "authors.id", "=", "books.author_id").ToSQL()
	if !strings.Contains(sqlStr, "RIGHT JOIN authors") {
		t.Fatalf("right join sql=%s", sqlStr)
	}

	sqlStr, _ = query.New(db, "sqlite", "books").CrossJoin("authors").ToSQL()
	if !strings.Contains(sqlStr, "CROSS JOIN authors") {
		t.Fatalf("cross join sql=%s", sqlStr)
	}

	rows, err := query.New(db, "sqlite", "books").
		SelectRaw("author_id").
		SelectRaw("SUM(copies) as total").
		GroupBy("author_id").
		Having("total", ">=", 5).
		OrHaving("total", "=", 1).
		OrderByDesc("total").
		Get()
	if err != nil || len(rows) != 2 {
		t.Fatalf("or having: len=%d err=%v", len(rows), err)
	}

	sqlStr, _ = query.New(db, "sqlite", "books").
		OrderBy("title").
		Reorder("id", "desc").
		ToSQL()
	if strings.Contains(sqlStr, "title") || !strings.Contains(sqlStr, "ORDER BY id desc") {
		t.Fatalf("reorder sql=%s", sqlStr)
	}

	rows, err = query.New(db, "sqlite", "books").OrderByDesc("id").Limit(1).Get()
	if err != nil || len(rows) != 1 {
		t.Fatalf("order by desc get: %#v err=%v", rows, err)
	}
	if id, _ := rows[0]["id"].(int64); id != 3 {
		t.Fatalf("expected id 3, got %#v", rows[0]["id"])
	}
}
