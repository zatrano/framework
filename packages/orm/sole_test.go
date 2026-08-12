package orm_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type soleModel struct {
	orm.Model
	Title string `db:"title"`
	Tag   string `db:"tag"`
}

func (soleModel) TableName() string { return "sole_models" }

func TestSoleAndOrWhereLike(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE sole_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		tag TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	orm.Configure(db, "sqlite")

	_, _ = orm.Create[soleModel](map[string]any{"title": "Alpha", "tag": "a"})
	_, _ = orm.Create[soleModel](map[string]any{"title": "Beta", "tag": "b"})
	_, _ = orm.Create[soleModel](map[string]any{"title": "Gamma", "tag": "a"})

	one, err := orm.Query[soleModel]().Where("title", "Beta").Sole()
	if err != nil || one.Title != "Beta" {
		t.Fatalf("sole=%+v err=%v", one, err)
	}
	if _, err := orm.Query[soleModel]().Where("tag", "a").Sole(); err == nil {
		t.Fatal("expected multiple records error")
	}
	if _, err := orm.Query[soleModel]().Where("title", "Nope").Sole(); err != sql.ErrNoRows {
		t.Fatalf("expected no rows, got %v", err)
	}

	items, err := orm.Query[soleModel]().WhereLike("title", "A%").OrWhereLike("title", "G%").OrderBy("id").Get()
	if err != nil || len(items) != 2 {
		t.Fatalf("or where like: len=%d err=%v", len(items), err)
	}
}
