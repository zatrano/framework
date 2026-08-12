package query_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/database/query"
)

func TestSelectOrderHavingRaw(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE raw_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		score INTEGER,
		grp TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`INSERT INTO raw_items (title, score, grp) VALUES
		('a', 10, 'x'),
		('b', 20, 'x'),
		('c', 5, 'y'),
		('d', 15, 'y')`)

	sqlStr, args := query.New(db, "sqlite", "raw_items").
		SelectRaw("score * ? as weighted", 2).
		Where("id", ">", 0).
		OrderByRaw("score DESC").
		ToSQL()
	if !strings.Contains(sqlStr, "score * ? as weighted") {
		t.Fatalf("select raw sql=%s", sqlStr)
	}
	if !strings.Contains(sqlStr, "ORDER BY score DESC") {
		t.Fatalf("order raw sql=%s", sqlStr)
	}
	if len(args) < 2 || args[0] != 2 {
		t.Fatalf("select bindings first: %#v", args)
	}

	rows, err := query.New(db, "sqlite", "raw_items").
		SelectRaw("grp").
		SelectRaw("COUNT(*) as total").
		SelectRaw("SUM(score) as points").
		GroupBy("grp").
		HavingRaw("COUNT(*) >= ?", 1).
		OrHavingRaw("SUM(score) > ?", 1000).
		OrderByRaw("total DESC").
		Get()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("grouped rows=%d", len(rows))
	}

	rows, err = query.New(db, "sqlite", "raw_items").
		SelectRaw("grp").
		SelectRaw("COUNT(*) as total").
		GroupBy("grp").
		HavingRaw("COUNT(*) >= ?", 2).
		Get()
	if err != nil || len(rows) != 2 {
		t.Fatalf("having filter: len=%d err=%v", len(rows), err)
	}

	rows, err = query.New(db, "sqlite", "raw_items").
		Select("title", "score").
		OrderByRaw("CASE WHEN title = ? THEN 0 ELSE 1 END, score DESC", "c").
		Get()
	if err != nil || len(rows) == 0 {
		t.Fatalf("order bindings: err=%v", err)
	}
	if title, _ := rows[0]["title"].(string); title != "c" {
		t.Fatalf("expected c first, got %#v", rows[0])
	}
}
