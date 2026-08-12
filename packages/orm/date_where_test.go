package orm_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type dateWhereModel struct {
	orm.Model
	Title string `db:"title"`
	Score int    `db:"score"`
}

func (dateWhereModel) TableName() string { return "date_where_models" }

func TestOrWhereInBetweenAndDateParts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE date_where_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		score INTEGER,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	orm.Configure(db, "sqlite")

	_, _ = db.Exec(`INSERT INTO date_where_models (title, score, created_at, updated_at) VALUES
		('a', 1, '2026-08-01 10:00:00', '2026-08-01 10:00:00'),
		('b', 5, '2026-08-03 12:00:00', '2026-08-03 12:00:00'),
		('c', 9, '2025-01-15 08:00:00', '2025-01-15 08:00:00')`)

	items, err := orm.Query[dateWhereModel]().
		Where("title", "nope").
		OrWhereIn("title", []any{"b"}).
		Get()
	if err != nil || len(items) != 1 || items[0].Title != "b" {
		t.Fatalf("or where in: %+v err=%v", items, err)
	}

	items, err = orm.Query[dateWhereModel]().
		Where("score", 0).
		OrWhereBetween("score", 8, 10).
		Get()
	if err != nil || len(items) != 1 || items[0].Title != "c" {
		t.Fatalf("or between: %+v err=%v", items, err)
	}

	items, err = orm.Query[dateWhereModel]().
		WhereYear("created_at", 2026).
		WhereMonth("created_at", 8).
		OrderBy("id").
		Get()
	if err != nil || len(items) != 2 {
		t.Fatalf("month/year: len=%d err=%v", len(items), err)
	}

	items, err = orm.Query[dateWhereModel]().WhereDate("created_at", "2026-08-03").Get()
	if err != nil || len(items) != 1 || items[0].Title != "b" {
		t.Fatalf("where date: %+v err=%v", items, err)
	}

	items, err = orm.Query[dateWhereModel]().WhereDay("created_at", 3).Get()
	if err != nil || len(items) != 1 || items[0].Title != "b" {
		t.Fatalf("where day: %+v err=%v", items, err)
	}
	items, err = orm.Query[dateWhereModel]().WhereHour("created_at", 12).Get()
	if err != nil || len(items) != 1 || items[0].Title != "b" {
		t.Fatalf("where hour: %+v err=%v", items, err)
	}
}
