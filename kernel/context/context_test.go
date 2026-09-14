package context_test

import (
	"testing"

	"github.com/zatrano/framework/v2/kernel/context"
)

func TestStore(t *testing.T) {
	s := context.New()
	if !s.Add("a", 1) || s.Add("a", 2) {
		t.Fatal("add")
	}
	s.Put("b", "x")
	if s.Get("a") != 1 || s.Get("missing", "fb") != "fb" || s.Get("missing") != nil {
		t.Fatal("get")
	}
	if !s.Has("a") || s.Has("no") {
		t.Fatal("has")
	}
	if s.Pull("b") != "x" || s.Has("b") {
		t.Fatal("pull")
	}
	if s.Pull("no", "fb") != "fb" || s.Pull("no") != nil {
		t.Fatal("pull missing")
	}
	s.Put("c", 3)
	s.Put("d", 4)
	s.Forget("c")
	if s.Has("c") || !s.Has("d") {
		t.Fatal("forget")
	}
	all := s.All()
	if all["d"] != 4 {
		t.Fatalf("%v", all)
	}
	only := s.Only("d", "missing")
	if len(only) != 1 || only["d"] != 4 {
		t.Fatalf("%v", only)
	}
	s.Flush()
	if len(s.All()) != 0 {
		t.Fatal("flush")
	}
}
