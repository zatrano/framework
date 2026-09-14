package consolecore

import "testing"

func TestHasFlagAndFormat(t *testing.T) {
	if !HasFlag([]string{"--json", "x"}, "--json") {
		t.Fatal("HasFlag")
	}
	got, err := FormatFromArgs([]string{"--format", "json"})
	if err != nil || got != "json" {
		t.Fatalf("FormatFromArgs=%q %v", got, err)
	}
	if ToSnake("MakeUser") != "make_user" {
		t.Fatal(ToSnake("MakeUser"))
	}
	if ToExported("name") != "Name" {
		t.Fatal(ToExported("name"))
	}
}

func TestParseEnabledAddons(t *testing.T) {
	src := "var EnabledAddons = []string{\n\t\"view\",\n\t\"validation\",\n}\n"
	got := ParseEnabledAddons(src)
	if len(got) != 2 || got[0] != "view" || got[1] != "validation" {
		t.Fatalf("%v", got)
	}
}
