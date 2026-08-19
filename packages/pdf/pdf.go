package pdf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/zatrano/framework/packages/http"
)

const (
	pageWidth    = 612.0 // US Letter
	pageHeight   = 792.0
	marginX      = 50.0
	marginTop    = 750.0
	fontSize     = 12.0
	lineHeight   = 14.0
	linesPerPage = 48
)

// Document is a multi-page text PDF.
type Document struct {
	Title string
	Lines []string
}

// New creates a document.
func New(title string, lines ...string) *Document {
	return &Document{Title: title, Lines: lines}
}

// FromMaps builds a simple table document (headers + rows as tab-separated lines).
func FromMaps(title string, rows []map[string]any, headers ...string) *Document {
	if len(headers) == 0 && len(rows) > 0 {
		seen := map[string]struct{}{}
		for _, row := range rows {
			for k := range row {
				seen[k] = struct{}{}
			}
		}
		for k := range seen {
			headers = append(headers, k)
		}
		// stable-ish order: sort by walking twice is fine for small sets — use insertion order from first row keys then extras
		headers = orderedHeaders(rows, headers)
	}
	lines := make([]string, 0, len(rows)+2)
	if title != "" {
		lines = append(lines, title, "")
	}
	if len(headers) > 0 {
		lines = append(lines, strings.Join(headers, " | "))
		lines = append(lines, strings.Repeat("-", min(80, len(strings.Join(headers, " | ")))))
	}
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i, h := range headers {
			cells[i] = stringify(row[h])
		}
		lines = append(lines, strings.Join(cells, " | "))
	}
	return &Document{Title: title, Lines: lines}
}

func orderedHeaders(rows []map[string]any, candidates []string) []string {
	if len(rows) == 0 {
		return candidates
	}
	out := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for k := range rows[0] {
		out = append(out, k)
		seen[k] = struct{}{}
	}
	for _, k := range candidates {
		if _, ok := seen[k]; !ok {
			out = append(out, k)
			seen[k] = struct{}{}
		}
	}
	return out
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprint(v)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Bytes renders a PDF 1.4 payload (multiple pages when needed).
func (d *Document) Bytes() []byte {
	lines := d.Lines
	if len(lines) == 0 {
		lines = []string{d.Title}
	}

	pages := chunkLines(lines, linesPerPage)
	if len(pages) == 0 {
		pages = [][]string{{d.Title}}
	}

	// Object layout:
	// 1 Catalog, 2 Pages, 3..N+2 Page, N+3..2N+2 Contents, 2N+3 Font
	n := len(pages)
	fontObj := 3 + 2*n
	pageObjs := make([]int, n)
	contentObjs := make([]int, n)
	for i := 0; i < n; i++ {
		pageObjs[i] = 3 + i
		contentObjs[i] = 3 + n + i
	}

	objects := make([]string, fontObj)
	// Catalog
	objects[0] = "1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj\n"
	// Pages kids
	var kids strings.Builder
	kids.WriteString("[")
	for i, id := range pageObjs {
		if i > 0 {
			kids.WriteByte(' ')
		}
		kids.WriteString(fmt.Sprintf("%d 0 R", id))
	}
	kids.WriteString("]")
	objects[1] = fmt.Sprintf("2 0 obj<< /Type /Pages /Kids %s /Count %d >>endobj\n", kids.String(), n)

	for i := 0; i < n; i++ {
		objects[pageObjs[i]-1] = fmt.Sprintf(
			"%d 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.0f %.0f] /Contents %d 0 R /Resources << /Font << /F1 %d 0 R >> >> >>endobj\n",
			pageObjs[i], pageWidth, pageHeight, contentObjs[i], fontObj,
		)
		stream := pageContent(pages[i])
		objects[contentObjs[i]-1] = fmt.Sprintf(
			"%d 0 obj<< /Length %d >>stream\n%s\nendstream endobj\n",
			contentObjs[i], len(stream), stream,
		)
	}
	objects[fontObj-1] = fmt.Sprintf("%d 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj\n", fontObj)

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

func chunkLines(lines []string, perPage int) [][]string {
	if perPage < 1 {
		perPage = linesPerPage
	}
	var pages [][]string
	for i := 0; i < len(lines); i += perPage {
		end := i + perPage
		if end > len(lines) {
			end = len(lines)
		}
		pages = append(pages, lines[i:end])
	}
	return pages
}

func pageContent(lines []string) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("BT\n/F1 %.0f Tf\n%.0f %.0f Td\n%.0f TL\n", fontSize, marginX, marginTop, lineHeight))
	for i, line := range lines {
		if i > 0 {
			content.WriteString("T*\n")
		}
		content.WriteString("(" + pdfEscape(line) + ") Tj\n")
	}
	content.WriteString("ET")
	return content.String()
}

// Response builds an application/pdf download (attachment).
func Response(filename string, title string, lines ...string) *http.Response {
	return Attachment(filename, New(title, lines...))
}

// Attachment returns a downloadable PDF response.
func Attachment(filename string, doc *Document) *http.Response {
	return pdfResponse(filename, doc, "attachment")
}

// Inline returns a PDF response meant for browser viewing (Content-Disposition: inline).
func Inline(filename string, doc *Document) *http.Response {
	return pdfResponse(filename, doc, "inline")
}

func pdfResponse(filename string, doc *Document, disposition string) *http.Response {
	if doc == nil {
		doc = New("document")
	}
	if filename == "" {
		filename = "document.pdf"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		filename += ".pdf"
	}
	resp := http.Text("")
	resp.SetContent(doc.Bytes(), "application/pdf")
	resp.Header("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, filename))
	return resp
}

func pdfEscape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	return r.Replace(s)
}
