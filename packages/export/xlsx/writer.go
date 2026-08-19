package xlsx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/zatrano/framework/packages/http"
)

// FromMaps writes an .xlsx workbook from row maps (union of keys, sorted).
func FromMaps(rows []map[string]any) ([]byte, error) {
	if len(rows) == 0 {
		return buildXLSX([]string{}, nil)
	}
	keysSet := map[string]struct{}{}
	for _, row := range rows {
		for k := range row {
			keysSet[k] = struct{}{}
		}
	}
	headers := make([]string, 0, len(keysSet))
	for k := range keysSet {
		headers = append(headers, k)
	}
	sort.Strings(headers)
	return FromMapsWithHeaders(rows, headers)
}

// FromMapsWithHeaders writes an .xlsx using explicit column order.
func FromMapsWithHeaders(rows []map[string]any, headers []string) ([]byte, error) {
	data := make([][]string, 0, len(rows))
	for _, row := range rows {
		line := make([]string, len(headers))
		for i, h := range headers {
			line[i] = stringify(row[h])
		}
		data = append(data, line)
	}
	return buildXLSX(headers, data)
}

// WriteTo writes an .xlsx from maps to w.
func WriteTo(w io.Writer, rows []map[string]any) error {
	raw, err := FromMaps(rows)
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}

// Response builds a downloadable XLSX HTTP response.
func Response(filename string, rows []map[string]any) *http.Response {
	body, err := FromMaps(rows)
	if err != nil {
		return http.JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	if filename == "" {
		filename = "export.xlsx"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		filename += ".xlsx"
	}
	resp := http.Text("")
	resp.SetContent(body, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	resp.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return resp
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

func buildXLSX(headers []string, rows [][]string) ([]byte, error) {
	// Build shared string table: headers + all cell values as shared strings.
	shared := make([]string, 0, len(headers)+len(rows)*len(headers))
	index := map[string]int{}
	addShared := func(s string) int {
		if i, ok := index[s]; ok {
			return i
		}
		i := len(shared)
		shared = append(shared, s)
		index[s] = i
		return i
	}
	headerIdx := make([]int, len(headers))
	for i, h := range headers {
		headerIdx[i] = addShared(h)
	}
	rowIdx := make([][]int, len(rows))
	for r, row := range rows {
		rowIdx[r] = make([]int, len(headers))
		for c := range headers {
			val := ""
			if c < len(row) {
				val = row[c]
			}
			rowIdx[r][c] = addShared(val)
		}
	}

	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	sheet.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	if len(headers) > 0 {
		writeSheetRow(&sheet, 1, headerIdx)
		for i, idxs := range rowIdx {
			writeSheetRow(&sheet, i+2, idxs)
		}
	}
	sheet.WriteString(`</sheetData></worksheet>`)

	var sst strings.Builder
	sst.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	sst.WriteString(`<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="`)
	sst.WriteString(strconv.Itoa(len(shared)))
	sst.WriteString(`" uniqueCount="`)
	sst.WriteString(strconv.Itoa(len(shared)))
	sst.WriteString(`">`)
	for _, s := range shared {
		sst.WriteString(`<si><t>`)
		sst.WriteString(xmlEscape(s))
		sst.WriteString(`</t></si>`)
	}
	sst.WriteString(`</sst>`)

	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
			`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
			`<Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>` +
			`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
			`</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
			`</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
			`<sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
			`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>` +
			`<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
			`</Relationships>`,
		"xl/styles.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
			`<fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>` +
			`<fills count="1"><fill><patternFill patternType="none"/></fill></fills>` +
			`<borders count="1"><border/></borders>` +
			`<cellStyleXfs count="1"><xf/></cellStyleXfs>` +
			`<cellXfs count="1"><xf xfId="0"/></cellXfs>` +
			`</styleSheet>`,
		"xl/sharedStrings.xml":     sst.String(),
		"xl/worksheets/sheet1.xml": sheet.String(),
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(w, body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeSheetRow(b *strings.Builder, rowNum int, idxs []int) {
	b.WriteString(`<row r="`)
	b.WriteString(strconv.Itoa(rowNum))
	b.WriteString(`">`)
	for c, idx := range idxs {
		ref := colName(c) + strconv.Itoa(rowNum)
		b.WriteString(`<c r="`)
		b.WriteString(ref)
		b.WriteString(`" t="s"><v>`)
		b.WriteString(strconv.Itoa(idx))
		b.WriteString(`</v></c>`)
	}
	b.WriteString(`</row>`)
}

func colName(index int) string {
	// 0 -> A, 25 -> Z, 26 -> AA
	if index < 0 {
		index = 0
	}
	var s string
	for index >= 0 {
		s = string(rune('A'+index%26)) + s
		index = index/26 - 1
	}
	return s
}

func xmlEscape(s string) string {
	r := strings.NewReplacer(
		`&`, `&amp;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
		`"`, `&quot;`,
	)
	return r.Replace(s)
}
