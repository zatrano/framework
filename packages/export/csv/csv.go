package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/zatrano/framework/packages/http"
)

// FromMaps writes CSV from a slice of maps (union of keys, sorted).
func FromMaps(rows []map[string]any) (string, error) {
	if len(rows) == 0 {
		return "", nil
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

// FromMapsWithHeaders writes CSV using explicit column order.
func FromMapsWithHeaders(rows []map[string]any, headers []string) (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(headers); err != nil {
		return "", err
	}
	for _, row := range rows {
		record := make([]string, len(headers))
		for i, h := range headers {
			record[i] = stringify(row[h])
		}
		if err := w.Write(record); err != nil {
			return "", err
		}
	}
	w.Flush()
	return buf.String(), w.Error()
}

// ToMaps parses CSV text into row maps keyed by header.
func ToMaps(data string) ([]map[string]string, error) {
	return ToMapsReader(strings.NewReader(data))
}

// ToMapsReader parses CSV from a reader.
func ToMapsReader(r io.Reader) ([]map[string]string, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	headers := records[0]
	out := make([]map[string]string, 0, len(records)-1)
	for _, row := range records[1:] {
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
		out = append(out, item)
	}
	return out, nil
}

// Response builds a downloadable CSV HTTP response.
func Response(filename string, rows []map[string]any) *http.Response {
	body, err := FromMaps(rows)
	if err != nil {
		return http.JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	if filename == "" {
		filename = "export.csv"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		filename += ".csv"
	}
	resp := http.Text(body)
	resp.Header("Content-Type", "text/csv; charset=utf-8")
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
