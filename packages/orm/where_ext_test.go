package orm_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type whereExtModel struct {
	orm.Model
	Title string  `db:"title"`
	Body  string  `db:"body"`
	Note  *string `db:"note"`
}

func (whereExtModel) TableName() string { return "where_ext_models" }

func TestWhereColumnRawNullAndWhen(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE where_ext_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		body TEXT,
		note TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	orm.Configure(db, "sqlite")

	note := "kept"
	_, _ = orm.Create[whereExtModel](map[string]any{"title": "same", "body": "same", "note": nil})
	_, _ = orm.Create[whereExtModel](map[string]any{"title": "a", "body": "b", "note": note})

	items, err := orm.Query[whereExtModel]().WhereColumn("title", "body").Get()
	if err != nil || len(items) != 1 {
		t.Fatalf("where column: len=%d err=%v", len(items), err)
	}

	items, err = orm.Query[whereExtModel]().
		Where("title", "a").
		OrWhereNull("note").
		Get()
	if err != nil || len(items) != 2 {
		t.Fatalf("or null: len=%d err=%v", len(items), err)
	}

	items, err = orm.Query[whereExtModel]().
		WhereRaw("title = ?", "same").
		Get()
	if err != nil || len(items) != 1 {
		t.Fatalf("where raw: len=%d err=%v", len(items), err)
	}

	filter := true
	items, err = orm.Query[whereExtModel]().
		When(filter, func(q *orm.Querier[whereExtModel]) *orm.Querier[whereExtModel] {
			return q.WhereNotNull("note")
		}).
		Get()
	if err != nil || len(items) != 1 || items[0].Title != "a" {
		t.Fatalf("when: %+v err=%v", items, err)
	}
}
