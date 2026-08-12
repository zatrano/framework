package session_test

import (
	"testing"

	"github.com/zatrano/framework/packages/session"
)

func TestBagLifecycleHelpers(t *testing.T) {
	m := session.NewManager(t.TempDir(), 120)
	bag, err := m.Start("")
	if err != nil {
		t.Fatal(err)
	}
	if bag.Has("n") {
		t.Fatal("expected missing")
	}
	if bag.Increment("n") != 1 || bag.Increment("n", 2) != 3 {
		t.Fatalf("increment=%v", bag.Get("n"))
	}
	if bag.Decrement("n") != 2 {
		t.Fatalf("decrement=%v", bag.Get("n"))
	}
	bag.Flash("msg", "hi")
	_ = m.Save(bag)

	next, err := m.Start(bag.ID())
	if err != nil {
		t.Fatal(err)
	}
	if !next.Has("n") || next.Get("msg") != "hi" {
		t.Fatalf("reload failed n=%v msg=%v", next.Get("n"), next.Get("msg"))
	}
	next.Keep("msg")
	all := next.All()
	if all["n"] != float64(2) && all["n"] != 2 {
		// JSON roundtrip may make numbers float64
		if n, ok := all["n"].(float64); !ok || n != 2 {
			if n, ok := all["n"].(int); !ok || n != 2 {
				t.Fatalf("all=%v", all)
			}
		}
	}
	if err := next.Invalidate(); err != nil {
		t.Fatal(err)
	}
	if next.Has("n") {
		t.Fatal("expected flush")
	}
}
