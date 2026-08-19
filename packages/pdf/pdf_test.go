package pdf_test

import (
	"bytes"
	"strings"
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
	if !bytes.Contains(raw, []byte("/BaseFont /Helvetica")) {
		t.Fatal("ASCII docs should keep Helvetica")
	}
	if !bytes.Contains(raw, []byte("%%EOF")) {
		t.Fatal("missing eof")
	}
}

func TestPDFTurkishEmbedsIdentityH(t *testing.T) {
	doc := pdf.New("Düğün", "Katılımcı listesi", "Kişi: Ayşe")
	raw := doc.Bytes()
	if !bytes.Contains(raw, []byte("/Identity-H")) {
		t.Fatal("expected Identity-H encoding for non-ASCII text")
	}
	if !bytes.Contains(raw, []byte("/CIDFontType2")) {
		t.Fatal("expected CIDFontType2")
	}
	// Must not put UTF-8 "Düğün" into a Helvetica literal string.
	if bytes.Contains(raw, []byte("(Düğün)")) {
		t.Fatal("Turkish title must not be Helvetica PDF string")
	}
	if bytes.Contains(raw, []byte("/BaseFont /Helvetica")) && !bytes.Contains(raw, []byte("/Identity-H")) {
		t.Fatal("fell back to Helvetica without Identity-H")
	}
}

func TestPDFMultipage(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	raw := pdf.New("many", lines...).Bytes()
	// 100 lines / 48 per page => 3 pages
	if !bytes.Contains(raw, []byte("/Count 3")) {
		t.Fatal("expected 3 pages")
	}
}

func TestFromMapsAndInline(t *testing.T) {
	doc := pdf.FromMaps("Users", []map[string]any{
		{"name": "Ada", "role": "admin"},
	}, "name", "role")
	raw := doc.Bytes()
	if !bytes.Contains(raw, []byte("Ada")) {
		t.Fatal("missing cell")
	}
	resp := pdf.Inline("users.pdf", doc)
	if resp == nil {
		t.Fatal("nil response")
	}
	cd := resp.GetHeader("Content-Disposition")
	if !strings.Contains(cd, "inline") {
		t.Fatalf("disposition=%q", cd)
	}
	att := pdf.Attachment("users.pdf", doc)
	if !strings.Contains(att.GetHeader("Content-Disposition"), "attachment") {
		t.Fatalf("attachment disposition=%q", att.GetHeader("Content-Disposition"))
	}
}
