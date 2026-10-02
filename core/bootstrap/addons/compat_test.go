package addons

import "testing"

func TestMeetsFrameworkMin(t *testing.T) {
	if !MeetsFrameworkMin("2.0.1", "") {
		t.Fatal("empty min must pass")
	}
	if !MeetsFrameworkMin("2.0.1", "2.0.1") {
		t.Fatal("equal must pass")
	}
	if !MeetsFrameworkMin("v2.1.0", "2.0.1") {
		t.Fatal("newer must pass")
	}
	if MeetsFrameworkMin("2.0.0", "2.0.1") {
		t.Fatal("older must fail")
	}
	if MeetsFrameworkMin("", "2.0.1") {
		t.Fatal("empty have must fail when min is set")
	}
}
