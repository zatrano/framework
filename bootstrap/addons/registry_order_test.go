package addons_test

import (
	"strings"
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

func TestOrderMetasMissingRequiresErrors(t *testing.T) {
	_, err := addons.OrderMetas([]addons.Meta{
		{Name: "auth", Order: 50, Requires: []string{"database", "session"}},
		{Name: "hashing", Order: 15},
	})
	if err == nil {
		t.Fatal("expected missing Requires to error")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Fatalf("err=%v", err)
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

func TestExpandPullsRequiresAndSkipsMissingOptional(t *testing.T) {
	catalog := map[string]addons.Meta{
		"auth":     {Name: "auth", Requires: []string{"database"}, Optional: []string{"redisx", "missing"}},
		"database": {Name: "database"},
		"redisx":   {Name: "redisx"},
	}
	lookup := func(name string) (addons.Meta, bool) {
		m, ok := catalog[name]
		return m, ok
	}
	got, err := addons.Expand([]string{"auth"}, lookup)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, m := range got {
		names[m.Name] = true
	}
	if !names["auth"] || !names["database"] || !names["redisx"] {
		t.Fatalf("expanded=%v", names)
	}
	if names["missing"] {
		t.Fatal("missing optional should not be included")
	}
}

func TestExpandMissingRequiresErrors(t *testing.T) {
	lookup := func(name string) (addons.Meta, bool) {
		if name == "auth" {
			return addons.Meta{Name: "auth", Requires: []string{"database"}}, true
		}
		return addons.Meta{}, false
	}
	_, err := addons.Expand([]string{"auth"}, lookup)
	if err == nil || !strings.Contains(err.Error(), "requires") {
		t.Fatalf("err=%v", err)
	}
}

func TestOrderMetasOptionalPresentBootsFirst(t *testing.T) {
	got, err := addons.OrderMetas([]addons.Meta{
		{Name: "auth", Order: 50, Optional: []string{"session"}},
		{Name: "session", Order: 130},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "session" || got[1].Name != "auth" {
		t.Fatalf("got %s %s", got[0].Name, got[1].Name)
	}
}

func TestOrderMetasOptionalMissingIsSkipped(t *testing.T) {
	got, err := addons.OrderMetas([]addons.Meta{
		{Name: "auth", Order: 50, Optional: []string{"session"}},
		{Name: "hashing", Order: 15},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "hashing" || got[1].Name != "auth" {
		t.Fatalf("got %#v", got)
	}
}

func TestBootableDropsUnsatisfiedRequiresAndDependents(t *testing.T) {
	got := addons.Bootable([]addons.Meta{
		{Name: "flash", Requires: []string{"session"}},
		{Name: "apitoken", Requires: []string{"auth"}},
		{Name: "auth", Requires: []string{"session"}},
		{Name: "health"},
		{Name: "orm", Requires: []string{"database"}},
		{Name: "database"},
	})
	names := map[string]bool{}
	for _, m := range got {
		names[m.Name] = true
	}
	if !names["health"] || !names["database"] || !names["orm"] {
		t.Fatalf("kept=%v", names)
	}
	if names["flash"] || names["session"] || names["auth"] || names["apitoken"] {
		t.Fatalf("should drop flash/auth chain without session, got %v", names)
	}
}

func TestBootableKeepsSatisfiedRequires(t *testing.T) {
	got := addons.Bootable([]addons.Meta{
		{Name: "flash", Requires: []string{"session"}},
		{Name: "session"},
	})
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
}
