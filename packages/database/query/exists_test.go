package query_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/database/query"
)

func TestWhereExistsHelpers(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE books (id INTEGER PRIMARY KEY, author_id INTEGER, title TEXT);
		INSERT INTO authors (id, name) VALUES (1, 'Ada'), (2, 'Grace');
		INSERT INTO books (id, author_id, title) VALUES (1, 1, 'A'), (2, 1, 'B');
	`)
	if err != nil {
		t.Fatal(err)
	}

	sub := query.New(db, "sqlite", "books").
		WhereRaw("books.author_id = authors.id")
	sqlStr, _ := query.New(db, "sqlite", "authors").WhereExists(sub).ToSQL()
	if !strings.Contains(sqlStr, "EXISTS (") || !strings.Contains(sqlStr, "SELECT 1 FROM books") {
		t.Fatalf("exists sql=%s", sqlStr)
	}

	rows, err := query.New(db, "sqlite", "authors").
		WhereExists(query.New(db, "sqlite", "books").WhereRaw("books.author_id = authors.id")).
		Get()
	if err != nil || len(rows) != 1 {
		t.Fatalf("where exists: len=%d err=%v", len(rows), err)
	}

	rows, err = query.New(db, "sqlite", "authors").
		WhereNotExists(query.New(db, "sqlite", "books").WhereRaw("books.author_id = authors.id")).
		Get()
	if err != nil || len(rows) != 1 {
		t.Fatalf("where not exists: len=%d err=%v rows=%v", len(rows), err, rows)
	}
	if name, _ := rows[0]["name"].(string); name != "Grace" {
		t.Fatalf("expected Grace, got %#v", rows[0])
	}

	rows, err = query.New(db, "sqlite", "authors").
		Where("id", 0).
		OrWhereExists(query.New(db, "sqlite", "books").WhereRaw("books.author_id = authors.id")).
		Get()
	if err != nil || len(rows) != 1 {
		t.Fatalf("or where exists: len=%d err=%v", len(rows), err)
	}

	rows, err = query.New(db, "sqlite", "authors").
		Where("id", 1).
		OrWhereNotExists(query.New(db, "sqlite", "books").WhereRaw("books.author_id = authors.id")).
		Get()
	if err != nil || len(rows) != 2 {
		t.Fatalf("or where not exists: len=%d err=%v", len(rows), err)
	}
}
