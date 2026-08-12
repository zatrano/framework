package pdf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/zatrano/framework/packages/http"
)

// Document is a minimal single-page PDF.
type Document struct {
	Title string
	Lines []string
}

// New creates a document.
func New(title string, lines ...string) *Document {
	return &Document{Title: title, Lines: lines}
}

// Bytes renders a minimal PDF 1.4 payload.
func (d *Document) Bytes() []byte {
	lines := d.Lines
	if len(lines) == 0 {
		lines = []string{d.Title}
	}
	var content strings.Builder
	content.WriteString("BT\n/F1 18 Tf\n50 750 Td\n14 TL\n")
	for i, line := range lines {
		if i > 0 {
			content.WriteString("T*\n")
		}
		content.WriteString("(" + pdfEscape(line) + ") Tj\n")
	}
	content.WriteString("ET")
	stream := content.String()

	objects := []string{
		"1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj\n",
		"2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj\n",
		"3 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>endobj\n",
		fmt.Sprintf("4 0 obj<< /Length %d >>stream\n%s\nendstream endobj\n", len(stream), stream),
		"5 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj\n",
	}

	var body bytes.Buffer
	body.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = body.Len()
		body.WriteString(obj)
	}
	xrefPos := body.Len()
	body.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objects)+1))
	body.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		body.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	body.WriteString(fmt.Sprintf("trailer<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefPos))
	return body.Bytes()
}

// Response builds an application/pdf download response.
func Response(filename string, title string, lines ...string) *http.Response {
	if filename == "" {
		filename = "document.pdf"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		filename += ".pdf"
	}
	doc := New(title, lines...)
	resp := http.Text("")
	resp.SetContent(doc.Bytes(), "application/pdf")
	resp.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return resp
}

func pdfEscape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	return r.Replace(s)
}
