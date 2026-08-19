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
	all := append([]string{}, lines...)
	if d.Title != "" {
		all = append(all, d.Title)
	}

	pages := chunkLines(lines, linesPerPage)
	if len(pages) == 0 {
		pages = [][]string{{d.Title}}
	}

	if needsUnicode(all) {
		if font, err := loadSystemTTF(); err == nil {
			return d.bytesUnicode(pages, font)
		}
	}
	return d.bytesHelvetica(pages)
}

func (d *Document) bytesHelvetica(pages [][]string) []byte {
	n := len(pages)
	fontObj := 3 + 2*n
	pageObjs := make([]int, n)
	contentObjs := make([]int, n)
	for i := 0; i < n; i++ {
		pageObjs[i] = 3 + i
		contentObjs[i] = 3 + n + i
	}

	objects := make([]string, fontObj)
	objects[0] = "1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj\n"
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
		stream := pageContentHelvetica(pages[i])
		objects[contentObjs[i]-1] = fmt.Sprintf(
			"%d 0 obj<< /Length %d >>stream\n%s\nendstream endobj\n",
			contentObjs[i], len(stream), stream,
		)
	}
	objects[fontObj-1] = fmt.Sprintf("%d 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj\n", fontObj)

	return assemblePDF(objects)
}

func (d *Document) bytesUnicode(pages [][]string, font *ttfFont) []byte {
	n := len(pages)
	// Objects: 1 Catalog, 2 Pages, 3..n+2 Page, n+3..2n+2 Contents,
	// 2n+3 Type0, 2n+4 CIDFont, 2n+5 FontDescriptor, 2n+6 FontFile2, 2n+7 ToUnicode
	type0Obj := 3 + 2*n
	cidObj := type0Obj + 1
	descObj := type0Obj + 2
	fileObj := type0Obj + 3
	toUniObj := type0Obj + 4
	numObjs := toUniObj

	pageObjs := make([]int, n)
	contentObjs := make([]int, n)
	for i := 0; i < n; i++ {
		pageObjs[i] = 3 + i
		contentObjs[i] = 3 + n + i
	}

	used := map[uint16]struct{}{}
	contents := make([]string, n)
	for i := 0; i < n; i++ {
		stream, pageUsed := pageContentUnicode(pages[i], font)
		contents[i] = stream
		for g := range pageUsed {
			used[g] = struct{}{}
		}
	}

	baseName := "ZAT+" + font.postscript
	compressed, err := flateBytes(font.data)
	if err != nil {
		return d.bytesHelvetica(pages)
	}
	toUni := buildToUnicode(font, used)
	widths := buildWidthArray(font, used)

	objects := make([][]byte, numObjs)

	objects[0] = []byte("1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj\n")
	var kids strings.Builder
	kids.WriteString("[")
	for i, id := range pageObjs {
		if i > 0 {
			kids.WriteByte(' ')
		}
		kids.WriteString(fmt.Sprintf("%d 0 R", id))
	}
	kids.WriteString("]")
	objects[1] = []byte(fmt.Sprintf("2 0 obj<< /Type /Pages /Kids %s /Count %d >>endobj\n", kids.String(), n))

	for i := 0; i < n; i++ {
		objects[pageObjs[i]-1] = []byte(fmt.Sprintf(
			"%d 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.0f %.0f] /Contents %d 0 R /Resources << /Font << /F1 %d 0 R >> >> >>endobj\n",
			pageObjs[i], pageWidth, pageHeight, contentObjs[i], type0Obj,
		))
		stream := contents[i]
		objects[contentObjs[i]-1] = []byte(fmt.Sprintf(
			"%d 0 obj<< /Length %d >>stream\n%s\nendstream endobj\n",
			contentObjs[i], len(stream), stream,
		))
	}

	objects[type0Obj-1] = []byte(fmt.Sprintf(
		"%d 0 obj<< /Type /Font /Subtype /Type0 /BaseFont /%s /Encoding /Identity-H /DescendantFonts [%d 0 R] /ToUnicode %d 0 R >>endobj\n",
		type0Obj, baseName, cidObj, toUniObj,
	))
	objects[cidObj-1] = []byte(fmt.Sprintf(
		"%d 0 obj<< /Type /Font /Subtype /CIDFontType2 /BaseFont /%s /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /FontDescriptor %d 0 R /DW 1000 /W %s /CIDToGIDMap /Identity >>endobj\n",
		cidObj, baseName, descObj, widths,
	))
	objects[descObj-1] = []byte(fmt.Sprintf(
		"%d 0 obj<< /Type /FontDescriptor /FontName /%s /Flags 32 /FontBBox [%d %d %d %d] /ItalicAngle 0 /Ascent %d /Descent %d /CapHeight %d /StemV 80 /FontFile2 %d 0 R >>endobj\n",
		descObj, baseName,
		font.bbox[0], font.bbox[1], font.bbox[2], font.bbox[3],
		font.ascent, font.descent, font.capHeight, fileObj,
	))

	var fileObjBuf bytes.Buffer
	fmt.Fprintf(&fileObjBuf, "%d 0 obj<< /Length %d /Length1 %d /Filter /FlateDecode >>stream\n", fileObj, len(compressed), len(font.data))
	fileObjBuf.Write(compressed)
	fileObjBuf.WriteString("\nendstream endobj\n")
	objects[fileObj-1] = fileObjBuf.Bytes()

	objects[toUniObj-1] = []byte(fmt.Sprintf(
		"%d 0 obj<< /Length %d >>stream\n%s\nendstream endobj\n",
		toUniObj, len(toUni), toUni,
	))

	return assemblePDFBytes(objects)
}

func assemblePDF(objects []string) []byte {
	objs := make([][]byte, len(objects))
	for i, o := range objects {
		objs[i] = []byte(o)
	}
	return assemblePDFBytes(objs)
}

func assemblePDFBytes(objects [][]byte) []byte {
	var body bytes.Buffer
	body.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = body.Len()
		body.Write(obj)
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

func pageContentHelvetica(lines []string) string {
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

func pageContentUnicode(lines []string, font *ttfFont) (string, map[uint16]struct{}) {
	used := map[uint16]struct{}{}
	var content strings.Builder
	content.WriteString(fmt.Sprintf("BT\n/F1 %.0f Tf\n%.0f %.0f Td\n%.0f TL\n", fontSize, marginX, marginTop, lineHeight))
	for i, line := range lines {
		if i > 0 {
			content.WriteString("T*\n")
		}
		hex, lineUsed := font.encodeLineHex(line)
		for g := range lineUsed {
			used[g] = struct{}{}
		}
		content.WriteString(hex + " Tj\n")
	}
	content.WriteString("ET")
	return content.String(), used
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
