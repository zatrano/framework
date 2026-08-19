package export_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/export"
	"github.com/zatrano/framework/packages/export/xlsx"
)

func TestToMapsCSV(t *testing.T) {
	rows, err := export.ToMaps("people.csv", strings.NewReader("name,email\nAda,a@x.com\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["name"] != "Ada" {
		t.Fatalf("%#v", rows)
	}
}

func TestToMapsXLSX(t *testing.T) {
	raw, err := xlsx.FromMaps([]map[string]any{{"name": "Ada"}})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := export.ToMapsBytes("people.xlsx", raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["name"] != "Ada" {
		t.Fatalf("%#v", rows)
	}
}
