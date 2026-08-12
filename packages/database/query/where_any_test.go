package query_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/database/query"
)

func TestWhereAnyAndAll(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		body TEXT,
		score INTEGER
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`INSERT INTO items (title, body, score) VALUES
		('alpha', 'one', 1),
		('beta', 'alpha', 2),
		('gamma', 'gamma', 3)`)

	sqlStr, _ := query.New(db, "sqlite", "items").
		WhereAny([]string{"title", "body"}, "LIKE", "%alpha%").
		ToSQL()
	if !strings.Contains(sqlStr, "(title LIKE ? OR body LIKE ?)") {
		t.Fatalf("where any sql=%s", sqlStr)
	}

	rows, err := query.New(db, "sqlite", "items").
		WhereAny([]string{"title", "body"}, "LIKE", "%alpha%").
		Get()
	if err != nil || len(rows) != 2 {
		t.Fatalf("where any: len=%d err=%v", len(rows), err)
	}

	sqlStr, _ = query.New(db, "sqlite", "items").
		WhereAll([]string{"title", "body"}, "gamma").
		ToSQL()
	if !strings.Contains(sqlStr, "(title = ? AND body = ?)") {
		t.Fatalf("where all sql=%s", sqlStr)
	}

	rows, err = query.New(db, "sqlite", "items").
		WhereAll([]string{"title", "body"}, "gamma").
		Get()
	if err != nil || len(rows) != 1 {
		t.Fatalf("where all: len=%d err=%v", len(rows), err)
	}

	rows, err = query.New(db, "sqlite", "items").
		Where("score", 1).
		OrWhereAny([]string{"title", "body"}, "beta").
		Get()
	if err != nil || len(rows) != 2 {
		t.Fatalf("or where any: len=%d err=%v", len(rows), err)
	}

	rows, err = query.New(db, "sqlite", "items").
		Where("score", 99).
		OrWhereAll([]string{"title", "body"}, "gamma").
		Get()
	if err != nil || len(rows) != 1 {
		t.Fatalf("or where all: len=%d err=%v", len(rows), err)
	}
}
