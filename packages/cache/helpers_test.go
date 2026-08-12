package cache_test

import (
	"testing"
	"time"

	"github.com/zatrano/framework/packages/cache"
)

func TestCacheHelpers(t *testing.T) {
	store, err := cache.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !cache.Add(store, "a", "1", time.Minute) {
		t.Fatal("add should succeed")
	}
	if cache.Add(store, "a", "2", time.Minute) {
		t.Fatal("add should fail when exists")
	}
	if err := cache.PutMany(store, map[string]any{"b": 2, "c": 3}, time.Minute); err != nil {
		t.Fatal(err)
	}
	many := cache.Many(store, "a", "b", "missing")
	if many["a"] != "1" || many["b"] == nil || many["missing"] != nil {
		t.Fatalf("many=%v", many)
	}
	n, err := cache.Increment(store, "counter")
	if err != nil || n != 1 {
		t.Fatalf("inc=%d err=%v", n, err)
	}
	n, err = cache.Decrement(store, "counter")
	if err != nil || n != 0 {
		t.Fatalf("dec=%d err=%v", n, err)
	}
	value, err := cache.RememberForever(store, "forever", func() (any, error) {
		return "ok", nil
	})
	if err != nil || value != "ok" {
		t.Fatalf("remember=%v err=%v", value, err)
	}
}
