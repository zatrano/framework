package orm_test

import (
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type pivotParent struct {
	orm.Model
	Name string     `db:"name"`
	Tags []pivotTag `db:"-"`
}

func (pivotParent) TableName() string { return "pivot_parents" }

type pivotTag struct {
	orm.Model
	Name string `db:"name"`
}

func (pivotTag) TableName() string { return "pivot_tags" }

func TestBelongsToManyWithPivot(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, sqlStr := range []string{
		`CREATE TABLE pivot_parents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE pivot_tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE pivot_parent_tag (
			parent_id INTEGER,
			tag_id INTEGER,
			active INTEGER,
			note TEXT
		)`,
	} {
		if _, err := db.Exec(sqlStr); err != nil {
			t.Fatal(err)
		}
	}
	orm.Configure(db, "sqlite")

	parent, _ := orm.Create[pivotParent](map[string]any{"name": "p"})
	tag1, _ := orm.Create[pivotTag](map[string]any{"name": "a"})
	tag2, _ := orm.Create[pivotTag](map[string]any{"name": "b"})
	if err := orm.Attach(parent, "pivot_parent_tag", "parent_id", "tag_id", []any{tag1.ID}, map[string]any{"active": 1, "note": "yes"}); err != nil {
		t.Fatal(err)
	}
	if err := orm.Attach(parent, "pivot_parent_tag", "parent_id", "tag_id", []any{tag2.ID}, map[string]any{"active": 0, "note": "no"}); err != nil {
		t.Fatal(err)
	}

	parents, err := orm.Query[pivotParent]().
		With(orm.EagerBelongsToManyWithPivot[pivotParent, pivotTag](
			"Tags", "pivot_parent_tag", "parent_id", "tag_id", []string{"active", "note"},
		)).
		Get()
	if err != nil || len(parents) != 1 || len(parents[0].Tags) != 2 {
		t.Fatalf("withPivot load=%+v err=%v", parents, err)
	}

	byName := map[string]map[string]any{}
	for i := range parents[0].Tags {
		tag := &parents[0].Tags[i]
		piv := orm.Pivot(tag)
		if piv == nil {
			t.Fatalf("missing pivot for %s", tag.Name)
		}
		byName[tag.Name] = piv
	}
	if fmt.Sprint(byName["a"]["note"]) != "yes" {
		t.Fatalf("pivot note=%v", byName["a"])
	}
	if fmt.Sprint(byName["b"]["active"]) != "0" {
		t.Fatalf("pivot active=%v", byName["b"])
	}
}
