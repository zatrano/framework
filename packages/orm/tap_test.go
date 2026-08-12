package orm_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type tapModel struct {
	orm.Model
	Title string `db:"title"`
}

func (tapModel) TableName() string { return "tap_models" }

func TestTapGetBindingsToSQL(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE tap_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	orm.Configure(db, "sqlite")

	tapped := false
	q := orm.Query[tapModel]().
		Where("title", "demo").
		Tap(func(query *orm.Querier[tapModel]) {
			tapped = true
			query.Limit(5)
		})
	if !tapped {
		t.Fatal("tap not called")
	}
	sqlStr, args := q.ToSQL()
	if !strings.Contains(sqlStr, "LIMIT 5") || len(args) < 1 || args[0] != "demo" {
		t.Fatalf("sql=%s args=%v", sqlStr, args)
	}
	bindings := q.GetBindings()
	if len(bindings) != len(args) || bindings[0] != "demo" {
		t.Fatalf("bindings=%v", bindings)
	}
}
