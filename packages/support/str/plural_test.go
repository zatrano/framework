package str_test

import (
	"testing"

	"github.com/zatrano/framework/packages/support/str"
)

func TestPluralize(t *testing.T) {
	if str.Plural("child") != "children" {
		t.Fatal(str.Plural("child"))
	}
	if str.Plural("cat", 1) != "cat" {
		t.Fatal(str.Plural("cat", 1))
	}
	if str.Plural("bus") != "buses" {
		t.Fatal(str.Plural("bus"))
	}
	if str.Singular("children") != "child" {
		t.Fatal(str.Singular("children"))
	}
	if str.Singular("cities") != "city" {
		t.Fatal(str.Singular("cities"))
	}
}
