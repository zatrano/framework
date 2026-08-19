package xlsx_test

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/zatrano/framework/packages/export/xlsx"
)

func TestToMapsReadsSharedStrings(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0"?><Types></Types>`,
		"xl/sharedStrings.xml": `<?xml version="1.0"?>
<sst><si><t>email</t></si><si><t>phone</t></si><si><t>a@x.com</t></si><si><t>+1</t></si></sst>`,
		"xl/worksheets/sheet1.xml": `<?xml version="1.0"?>
<worksheet><sheetData>
  <row><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>
  <row><c r="A2" t="s"><v>2</v></c><c r="B2" t="s"><v>3</v></c></row>
</sheetData></worksheet>`,
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	rows, err := xlsx.ToMaps(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["email"] != "a@x.com" || rows[0]["phone"] != "+1" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}

func TestFromMapsRoundTrip(t *testing.T) {
	in := []map[string]any{
		{"email": "a@x.com", "name": "Ada"},
		{"email": "b@y.com", "name": "Bob"},
	}
	raw, err := xlsx.FromMaps(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(raw, []byte("PK")) {
		t.Fatal("expected zip/xlsx magic")
	}
	out, err := xlsx.ToMaps(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("rows=%d %#v", len(out), out)
	}
	byEmail := map[string]string{}
	for _, row := range out {
		byEmail[row["email"]] = row["name"]
	}
	if byEmail["a@x.com"] != "Ada" || byEmail["b@y.com"] != "Bob" {
		t.Fatalf("unexpected %#v", out)
	}
}

func TestFromMapsWithHeadersOrder(t *testing.T) {
	raw, err := xlsx.FromMapsWithHeaders([]map[string]any{
		{"b": "2", "a": "1"},
	}, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := xlsx.ToMaps(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0]["a"] != "1" || out[0]["b"] != "2" {
		t.Fatalf("%#v", out)
	}
}
