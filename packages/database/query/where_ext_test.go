package query_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/database/query"
)

func TestWhereColumnRawNullAndWhen(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		body TEXT,
		score INTEGER,
		note TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`INSERT INTO items (title, body, score, note) VALUES
		('same', 'same', 10, NULL),
		('alpha', 'beta', 5, 'x'),
		('gamma', 'gamma', 1, NULL)`)

	b := query.New(db, "sqlite", "items")
	sqlStr, _ := b.WhereColumn("title", "body").ToSQL()
	if !strings.Contains(sqlStr, "title = body") {
		t.Fatalf("where column sql=%s", sqlStr)
	}

	rows, err := query.New(db, "sqlite", "items").WhereColumn("title", "body").Get()
	if err != nil || len(rows) != 2 {
		t.Fatalf("where column: len=%d err=%v", len(rows), err)
	}

	rows, err = query.New(db, "sqlite", "items").
		Where("score", ">", 0).
		OrWhereNull("note").
		Get()
	if err != nil || len(rows) < 2 {
		t.Fatalf("or where null: len=%d err=%v", len(rows), err)
	}

	rows, err = query.New(db, "sqlite", "items").
		WhereRaw("score >= ?", 5).
		OrWhereRaw("title = ?", "gamma").
		Get()
	if err != nil || len(rows) != 3 {
		t.Fatalf("where raw: len=%d err=%v", len(rows), err)
	}

	min := 8
	rows, err = query.New(db, "sqlite", "items").
		When(min > 0, func(q *query.Builder) *query.Builder {
			return q.Where("score", ">=", min)
		}).
		Unless(true, func(q *query.Builder) *query.Builder {
			return q.WhereNotNull("note")
		}).
		Get()
	if err != nil || len(rows) != 1 {
		t.Fatalf("when/unless: len=%d err=%v", len(rows), err)
	}
}
