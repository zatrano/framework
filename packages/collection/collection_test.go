package collection_test

import (
	"testing"

	"github.com/zatrano/framework/packages/collection"
)

func TestCollectionFilterAndTake(t *testing.T) {
	c := collection.Make(1, 2, 3, 4, 5)
	even := c.Filter(func(n int) bool { return n%2 == 0 })
	if even.Count() != 2 {
		t.Fatalf("expected 2 even numbers, got %d", even.Count())
	}
	taken := c.Take(3).All()
	if len(taken) != 3 || taken[0] != 1 {
		t.Fatalf("unexpected take result: %#v", taken)
	}
}

func TestGroupByKeyByPartition(t *testing.T) {
	type row struct {
		Role string
		Name string
	}
	c := collection.Make(
		row{Role: "admin", Name: "Ada"},
		row{Role: "user", Name: "Bob"},
		row{Role: "admin", Name: "Eve"},
	)
	grouped := collection.GroupBy(c, func(r row) string { return r.Role })
	if len(grouped["admin"]) != 2 {
		t.Fatalf("grouped=%#v", grouped)
	}
	keyed := collection.KeyBy(c, func(r row) string { return r.Name })
	if keyed["Bob"].Role != "user" {
		t.Fatalf("keyed=%#v", keyed)
	}
	pass, fail := c.Partition(func(r row) bool { return r.Role == "admin" })
	if pass.Count() != 2 || fail.Count() != 1 {
		t.Fatalf("partition pass=%d fail=%d", pass.Count(), fail.Count())
	}
}

func TestFirstWhereFlatMapSliding(t *testing.T) {
	c := collection.Make(1, 2, 3, 4, 5)
	got, ok := c.FirstWhere(func(n int) bool { return n > 3 })
	if !ok || got != 4 {
		t.Fatalf("firstWhere=%v ok=%v", got, ok)
	}
	last, ok := c.LastWhere(func(n int) bool { return n%2 == 0 })
	if !ok || last != 4 {
		t.Fatalf("lastWhere=%v", last)
	}
	flat := collection.FlatMap(c, func(n int) []int { return []int{n, n * 10} })
	if flat.Count() != 10 {
		t.Fatalf("flatMap count=%d", flat.Count())
	}
	nested := collection.Make([]int{1, 2}, []int{3})
	if collection.Flatten(nested).Count() != 3 {
		t.Fatal("flatten")
	}
	windows := c.Sliding(3, 2)
	if len(windows) != 2 || windows[0].Count() != 3 || windows[1].All()[0] != 3 {
		t.Fatalf("sliding=%v", windows)
	}
}
