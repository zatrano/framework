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
	if bag.Changed() {
		t.Fatal("untouched anonymous bag should not be changed")
	}
	if bag.Has("n") {
		t.Fatal("expected missing")
	}
	if bag.Increment("n") != 1 || bag.Increment("n", 2) != 3 {
		t.Fatalf("increment=%v", bag.Get("n"))
	}
	if !bag.Changed() {
		t.Fatal("Put/Increment should mark changed")
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
	if !next.Changed() {
		t.Fatal("loaded flash should mark changed so drain persists")
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

func TestChangedAnonymousAndFlash(t *testing.T) {
	m := session.NewManager(t.TempDir(), 120)
	bag, err := m.Start("")
	if err != nil {
		t.Fatal(err)
	}
	if bag.Changed() {
		t.Fatal("Start(\"\") should leave changed=false")
	}
	bag.Put("x", 1)
	if !bag.Changed() {
		t.Fatal("Put should set changed")
	}
	_ = m.Save(bag)

	// No flash on disk → reload should be unchanged until a write.
	clean, err := m.Start(bag.ID())
	if err != nil {
		t.Fatal(err)
	}
	if clean.Changed() {
		t.Fatal("session without pending flash should start unchanged")
	}
}
