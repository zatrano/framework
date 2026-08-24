package orm_test

import (
	"testing"

	"github.com/zatrano/framework/packages/orm"
)

func TestCursorAndCollectionAndWithOnly(t *testing.T) {
	db := setupEagerDB(t)
	defer db.Close()

	for i := 0; i < 5; i++ {
		_, _ = orm.Create[eagerParent](map[string]any{"name": "p"})
	}

	var n int
	err := orm.Query[eagerParent]().OrderBy("id").Cursor(func(p *eagerParent) error {
		n++
		return nil
	})
	if err != nil || n != 5 {
		t.Fatalf("cursor n=%d err=%v", n, err)
	}

	parents, err := orm.Query[eagerParent]().OrderBy("id").Get()
	if err != nil {
		t.Fatal(err)
	}
	col := orm.Collect(parents)
	ids, err := col.Pluck("id")
	if err != nil || len(ids) != 5 {
		t.Fatalf("pluck=%v err=%v", ids, err)
	}

	_, _ = orm.Create[eagerChildLoad](map[string]any{"parent_id": parents[0].ID, "name": "c"})
	if err := orm.Load(&parents, orm.EagerHasMany[eagerParent, eagerChildLoad]("Children", "parent_id")); err != nil {
		t.Fatal(err)
	}
	if len(parents[0].Children) != 1 {
		t.Fatalf("load fluent children=%d", len(parents[0].Children))
	}

	q := orm.Query[eagerParent]().With(orm.EagerHasMany[eagerParent, eagerChildLoad]("Children", "parent_id"))
	q.Without()
	q.WithOnly(orm.EagerCount[eagerParent, eagerChildLoad]("ChildrenCount", "parent_id"))
	got, err := q.OrderBy("id").Limit(1).Get()
	if err != nil || got[0].ChildrenCount != 1 {
		t.Fatalf("withOnly count=%d err=%v children=%d", got[0].ChildrenCount, err, len(got[0].Children))
	}
}
