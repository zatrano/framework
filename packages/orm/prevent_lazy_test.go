package orm_test

import (
	"testing"

	"github.com/zatrano/framework/packages/orm"
)

func TestPreventLazyLoading(t *testing.T) {
	db := setupEagerDB(t)
	defer db.Close()

	p, _ := orm.Create[eagerParent](map[string]any{"name": "p"})
	_, _ = orm.Create[eagerChildLoad](map[string]any{"parent_id": p.ID, "name": "c"})

	orm.PreventLazyLoading(true)
	t.Cleanup(func() { orm.PreventLazyLoading(false) })

	if _, err := orm.LazyHasMany[eagerParent, eagerChildLoad](p, "Children", "parent_id"); err == nil {
		t.Fatal("expected lazy loading prevented")
	}

	parents, err := orm.Query[eagerParent]().
		With(orm.EagerHasMany[eagerParent, eagerChildLoad]("Children", "parent_id")).
		Get()
	if err != nil {
		t.Fatal(err)
	}
	// Already loaded — LazyHasMany should allow when RelationLoaded.
	if _, err := orm.LazyHasMany[eagerParent, eagerChildLoad](&parents[0], "Children", "parent_id"); err != nil {
		t.Fatalf("loaded relation should pass: %v", err)
	}
}
