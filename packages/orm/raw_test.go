package orm_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type rawModel struct {
	orm.Model
	Title string `db:"title"`
	Score int    `db:"score"`
	Grp   string `db:"grp"`
}

func (rawModel) TableName() string { return "raw_models" }

func TestOrderByRawAndSelectRaw(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE raw_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		score INTEGER,
		grp TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	orm.Configure(db, "sqlite")

	_, _ = orm.Create[rawModel](map[string]any{"title": "a", "score": 10, "grp": "x"})
	_, _ = orm.Create[rawModel](map[string]any{"title": "b", "score": 30, "grp": "x"})
	_, _ = orm.Create[rawModel](map[string]any{"title": "c", "score": 20, "grp": "y"})

	items, err := orm.Query[rawModel]().OrderByRaw("score DESC").Limit(2).Get()
	if err != nil || len(items) != 2 || items[0].Title != "b" {
		t.Fatalf("order by raw: %+v err=%v", items, err)
	}

	items, err = orm.Query[rawModel]().
		Select("id", "title", "score", "grp").
		OrderByRaw("CASE WHEN title = ? THEN 0 ELSE 1 END", "c").
		Get()
	if err != nil || len(items) == 0 || items[0].Title != "c" {
		t.Fatalf("order bindings: %+v err=%v", items, err)
	}
}
