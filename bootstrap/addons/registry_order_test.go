package addons_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/bootstrap/addons"
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

func TestOrderMetasOrderTieBreakIndependent(t *testing.T) {
	got, err := addons.OrderMetas([]addons.Meta{
		{Name: "alpha", Order: 50},
		{Name: "zeta", Order: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "zeta" || got[1].Name != "alpha" {
		t.Fatalf("Order must beat Name: got %s %s", got[0].Name, got[1].Name)
	}
}

func namesOfMetas(metas []addons.Meta) []string {
	out := make([]string, len(metas))
	for i, m := range metas {
		out[i] = m.Name
	}
	return out
}

func equalNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestOrderMetasRepeatedIdentical(t *testing.T) {
	in := []addons.Meta{
		{Name: "auth", Order: 50, Requires: []string{"database", "session"}},
		{Name: "session", Order: 130},
		{Name: "zeta", Order: 1},
		{Name: "alpha", Order: 1},
		{Name: "database", Order: 10},
	}
	first, err := addons.OrderMetas(in)
	if err != nil {
		t.Fatal(err)
	}
	want := namesOfMetas(first)
	for i := 0; i < 25; i++ {
		got, err := addons.OrderMetas(in)
		if err != nil {
			t.Fatal(err)
		}
		if !equalNames(want, namesOfMetas(got)) {
			t.Fatalf("run %d: want %v got %v", i, want, namesOfMetas(got))
		}
	}
}

func TestOrderMetasSameSetSameOrderAcrossShuffledInput(t *testing.T) {
	base := []addons.Meta{
		{Name: "auth", Order: 50, Requires: []string{"hashing", "session"}},
		{Name: "session", Order: 130, Optional: []string{"redisx"}},
		{Name: "hashing", Order: 15},
		{Name: "database", Order: 10},
		{Name: "redisx", Order: 20},
		{Name: "zeta", Order: 1},
		{Name: "alpha", Order: 1},
	}
	first, err := addons.OrderMetas(append([]addons.Meta(nil), base...))
	if err != nil {
		t.Fatal(err)
	}
	want := namesOfMetas(first)
	pos := map[string]int{}
	for i, name := range want {
		pos[name] = i
	}
	if pos["auth"] < pos["hashing"] || pos["auth"] < pos["session"] {
		t.Fatalf("dependency must precede dependent: %v", want)
	}
	if pos["session"] < pos["redisx"] {
		t.Fatalf("optional present must precede dependent: %v", want)
	}
	if pos["alpha"] > pos["zeta"] {
		t.Fatalf("Name tie-break: %v", want)
	}
	if pos["database"] > pos["hashing"] {
		t.Fatalf("Order tie-break: %v", want)
	}

	perms := [][]int{
		{6, 5, 4, 3, 2, 1, 0},
		{0, 2, 4, 6, 1, 3, 5},
		{3, 1, 5, 0, 2, 4, 6},
		{1, 0, 3, 2, 5, 4, 6},
		{4, 0, 6, 2, 5, 1, 3},
	}
	for i, perm := range perms {
		shuffled := make([]addons.Meta, len(base))
		for j, idx := range perm {
			shuffled[j] = base[idx]
		}
		got, err := addons.OrderMetas(shuffled)
		if err != nil {
			t.Fatal(err)
		}
		if !equalNames(want, namesOfMetas(got)) {
			t.Fatalf("shuffle %d: want %v got %v", i, want, namesOfMetas(got))
		}
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
