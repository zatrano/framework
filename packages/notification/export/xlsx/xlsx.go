package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ToMapsReader reads the first worksheet of an .xlsx into row maps keyed by header.
func ToMapsReader(r io.Reader) ([]map[string]string, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ToMaps(raw)
}

// ToMaps parses .xlsx bytes into maps.
func ToMaps(data []byte) ([]map[string]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("xlsx: %w", err)
	}
	shared, err := readSharedStrings(zr)
	if err != nil {
		return nil, err
	}
	sheet, err := findFirstSheet(zr)
	if err != nil {
		return nil, err
	}
	rows, err := readSheetRows(sheet, shared)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	headers := rows[0]
	out := make([]map[string]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		item := make(map[string]string, len(headers))
		for i, h := range headers {
			h = strings.TrimSpace(h)
			if h == "" {
				continue
			}
			if i < len(row) {
				item[h] = row[i]
			} else {
				item[h] = ""
			}
		}
		empty := true
		for _, v := range item {
			if strings.TrimSpace(v) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func findFirstSheet(zr *zip.Reader) (io.ReadCloser, error) {
	candidates := []string{
		"xl/worksheets/sheet1.xml",
		"xl/worksheets/sheet.xml",
	}
	for _, name := range candidates {
		for _, f := range zr.File {
			if f.Name == name {
				return f.Open()
			}
		}
	}
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("xlsx: worksheet not found")
}

func readSharedStrings(zr *zip.Reader) ([]string, error) {
	var file *zip.File
	for _, f := range zr.File {
		if f.Name == "xl/sharedStrings.xml" {
			file = f
			break
		}
	}
	if file == nil {
		return nil, nil
	}
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	type tNode struct {
		Text string `xml:"t"`
	}
	type siNode struct {
		T      string  `xml:"t"`
		Richer []tNode `xml:"r"`
	}
	type sst struct {
		SI []siNode `xml:"si"`
	}
	var doc sst
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(doc.SI))
	for _, si := range doc.SI {
		if si.T != "" {
			out = append(out, si.T)
			continue
		}
		var b strings.Builder
		for _, r := range si.Richer {
			b.WriteString(r.Text)
		}
		out = append(out, b.String())
	}
	return out, nil
}

type sheetXML struct {
	Rows []struct {
		Cells []struct {
			Ref  string `xml:"r,attr"`
			Type string `xml:"t,attr"`
			V    string `xml:"v"`
			IS   struct {
				T string `xml:"t"`
			} `xml:"is"`
		} `xml:"c"`
	} `xml:"sheetData>row"`
}

func readSheetRows(rc io.ReadCloser, shared []string) ([][]string, error) {
	defer rc.Close()
	var doc sheetXML
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return nil, err
	}
	out := make([][]string, 0, len(doc.Rows))
	for _, row := range doc.Rows {
		maxCol := 0
		values := map[int]string{}
		for _, cell := range row.Cells {
			col := columnIndex(cell.Ref)
			if col > maxCol {
				maxCol = col
			}
			val := cell.V
			switch cell.Type {
			case "s":
				if idx, err := strconv.Atoi(strings.TrimSpace(cell.V)); err == nil && idx >= 0 && idx < len(shared) {
					val = shared[idx]
				}
			case "inlineStr":
				val = cell.IS.T
			}
			values[col] = val
		}
		line := make([]string, maxCol+1)
		for i := 0; i <= maxCol; i++ {
			line[i] = values[i]
		}
		out = append(out, line)
	}
	return out, nil
}

func columnIndex(ref string) int {
	col := 0
	for _, ch := range ref {
		if ch < 'A' || ch > 'Z' {
			break
		}
		col = col*26 + int(ch-'A') + 1
	}
	if col == 0 {
		return 0
	}
	return col - 1
}
