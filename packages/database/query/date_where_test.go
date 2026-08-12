package query_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/database/query"
)

func TestOrWhereInBetweenAndDateParts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE dated_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		score INTEGER,
		created_at TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`INSERT INTO dated_items (title, score, created_at) VALUES
		('a', 1, '2026-08-01 10:00:00'),
		('b', 5, '2026-08-03 12:00:00'),
		('c', 9, '2025-01-15 08:00:00')`)

	sqlStr, _ := query.New(db, "sqlite", "dated_items").
		Where("title", "missing").
		OrWhereIn("title", []any{"a", "b"}).
		ToSQL()
	if !strings.Contains(strings.ToUpper(sqlStr), "OR") || !strings.Contains(sqlStr, "IN") {
		t.Fatalf("or where in sql=%s", sqlStr)
	}

	rows, err := query.New(db, "sqlite", "dated_items").
		Where("score", 0).
		OrWhereBetween("score", 4, 6).
		Get()
	if err != nil || len(rows) != 1 || rows[0]["title"] != "b" {
		t.Fatalf("or between: %+v err=%v", rows, err)
	}

	rows, err = query.New(db, "sqlite", "dated_items").
		WhereDate("created_at", "2026-08-03").
		Get()
	if err != nil || len(rows) != 1 || rows[0]["title"] != "b" {
		t.Fatalf("where date: %+v err=%v", rows, err)
	}

	rows, err = query.New(db, "sqlite", "dated_items").
		WhereMonth("created_at", 8).
		WhereYear("created_at", 2026).
		OrderBy("id").
		Get()
	if err != nil || len(rows) != 2 {
		t.Fatalf("where month/year: len=%d err=%v", len(rows), err)
	}
}

func TestWhereDayTimeHourAndWeekday(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE clock_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		created_at TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}
	// 2026-08-03 is a Monday (weekday 1 with Sunday=0)
	_, _ = db.Exec(`INSERT INTO clock_items (title, created_at) VALUES
		('a', '2026-08-03 12:30:00'),
		('b', '2026-08-01 08:00:00'),
		('c', '2026-07-15 12:30:00')`)

	rows, err := query.New(db, "sqlite", "clock_items").WhereDay("created_at", 3).Get()
	if err != nil || len(rows) != 1 || rows[0]["title"] != "a" {
		t.Fatalf("where day: %+v err=%v", rows, err)
	}

	rows, err = query.New(db, "sqlite", "clock_items").WhereTime("created_at", "12:30:00").OrderBy("id").Get()
	if err != nil || len(rows) != 2 {
		t.Fatalf("where time: len=%d err=%v", len(rows), err)
	}

	rows, err = query.New(db, "sqlite", "clock_items").WhereHour("created_at", 12).OrderBy("id").Get()
	if err != nil || len(rows) != 2 {
		t.Fatalf("where hour: len=%d err=%v", len(rows), err)
	}

	rows, err = query.New(db, "sqlite", "clock_items").WhereDayOfWeek("created_at", 1).Get() // Monday
	if err != nil || len(rows) != 1 || rows[0]["title"] != "a" {
		t.Fatalf("where dow: %+v err=%v", rows, err)
	}
}
