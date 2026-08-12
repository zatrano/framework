package pdf_test

import (
	"bytes"
	"testing"

	"github.com/zatrano/framework/packages/pdf"
)

func TestPDFBytes(t *testing.T) {
	doc := pdf.New("ZATRANO", "Hello PDF", "Phase 26")
	raw := doc.Bytes()
	if !bytes.HasPrefix(raw, []byte("%PDF-1.4")) {
		t.Fatal("missing header")
	}
	if !bytes.Contains(raw, []byte("Hello PDF")) {
		t.Fatal("missing text")
	}
	if !bytes.Contains(raw, []byte("%%EOF")) {
		t.Fatal("missing eof")
	}
}
