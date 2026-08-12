package query_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/database/query"
)

func TestTapAndGetBindings(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tapped := false
	b := query.New(db, "sqlite", "items").
		Where("score", ">", 5).
		Tap(func(q *query.Builder) {
			tapped = true
			q.OrderBy("id")
		}).
		Limit(10)

	if !tapped {
		t.Fatal("tap not called")
	}
	sqlStr, args := b.ToSQL()
	if len(args) != 1 || args[0] != 5 {
		t.Fatalf("to sql args=%v sql=%s", args, sqlStr)
	}
	bindings := b.GetBindings()
	if len(bindings) != 1 || bindings[0] != 5 {
		t.Fatalf("bindings=%v", bindings)
	}
}

func TestAddSelect(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sqlStr, _ := query.New(db, "sqlite", "posts").
		Select("id").
		AddSelect("title", "published").
		ToSQL()
	if !strings.Contains(sqlStr, "id") || !strings.Contains(sqlStr, "title") || !strings.Contains(sqlStr, "published") {
		t.Fatalf("add select sql=%s", sqlStr)
	}
}
