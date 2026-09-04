package addons_test

import (
	"testing"

	"github.com/zatrano/framework/bootstrap/addons"
)

func TestOrderMetasRequiresBeforeDependents(t *testing.T) {
	metas := []addons.Meta{
		{Name: "auth", Order: 50, Requires: []string{"database", "session", "hashing"}},
		{Name: "session", Order: 130},
		{Name: "database", Order: 10},
		{Name: "hashing", Order: 15},
	}
	got, err := addons.OrderMetas(metas)
	if err != nil {
		t.Fatal(err)
	}
	pos := map[string]int{}
	var names []string
	for i, m := range got {
		pos[m.Name] = i
		names = append(names, m.Name)
	}
	if pos["auth"] < pos["database"] || pos["auth"] < pos["session"] || pos["auth"] < pos["hashing"] {
		t.Fatalf("auth booted too early: %v", names)
	}
	if pos["database"] > pos["hashing"] {
		// database Order 10, hashing 15, both independent — database first
	}
}

func TestOrderMetasSkipsMissingRequires(t *testing.T) {
	metas := []addons.Meta{
		{Name: "auth", Order: 50, Requires: []string{"database", "session"}},
		{Name: "hashing", Order: 15},
	}
	got, err := addons.OrderMetas(metas)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "hashing" || got[1].Name != "auth" {
		t.Fatalf("got %#v", got)
	}
}

func TestOrderMetasCycle(t *testing.T) {
	_, err := addons.OrderMetas([]addons.Meta{
		{Name: "a", Requires: []string{"b"}},
		{Name: "b", Requires: []string{"a"}},
	})
	if err == nil {
		t.Fatal("expected cycle")
	}
}

func TestOrderMetasTieBreak(t *testing.T) {
	got, err := addons.OrderMetas([]addons.Meta{
		{Name: "zeta", Order: 1},
		{Name: "alpha", Order: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "alpha" || got[1].Name != "zeta" {
		t.Fatalf("got %s %s", got[0].Name, got[1].Name)
	}
}

func TestNewPlanNilMeansAllImported(t *testing.T) {
	p := addons.NewPlan(nil)
	if len(p.Imported) != len(p.Enabled) {
		t.Fatalf("imported=%d enabled=%d", len(p.Imported), len(p.Enabled))
	}
}

func TestNewPlanEmptyMeansKernelOnly(t *testing.T) {
	p := addons.NewPlan([]string{})
	if p.Enabled == nil {
		t.Fatal("enabled should be empty slice, not nil")
	}
	if len(p.Enabled) != 0 {
		t.Fatalf("enabled=%v", p.Enabled)
	}
}
