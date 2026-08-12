package csv_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/export/csv"
)

func TestFromMaps(t *testing.T) {
	out, err := csv.FromMaps([]map[string]any{
		{"id": 1, "name": "Ada"},
		{"id": 2, "name": "Grace"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "id,name") && !strings.Contains(out, "name,id") {
		t.Fatalf("headers missing: %q", out)
	}
	if !strings.Contains(out, "Ada") || !strings.Contains(out, "Grace") {
		t.Fatalf("rows missing: %q", out)
	}
}

func TestToMaps(t *testing.T) {
	rows, err := csv.ToMaps("name,email\nAda,ada@zatrano.test\nBob,bob@zatrano.test\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0]["name"] != "Ada" || rows[1]["email"] != "bob@zatrano.test" {
		t.Fatalf("unexpected %#v", rows)
	}
}
