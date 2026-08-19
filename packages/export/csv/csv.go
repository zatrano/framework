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

// Options controls CSV encode/decode behaviour.
type Options struct {
	// Comma is the field delimiter (default ',').
	Comma rune
	// UseBOM writes a UTF-8 BOM on export (Excel-friendly).
	UseBOM bool
}

func (o Options) comma() rune {
	if o.Comma == 0 {
		return ','
	}
	return o.Comma
}

// FromMaps writes CSV from a slice of maps (union of keys, sorted).
func FromMaps(rows []map[string]any) (string, error) {
	return FromMapsWithOptions(rows, Options{})
}

// FromMapsWithOptions writes CSV with delimiter / BOM options.
func FromMapsWithOptions(rows []map[string]any, opt Options) (string, error) {
	if len(rows) == 0 {
		if opt.UseBOM {
			return "\uFEFF", nil
		}
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
	return FromMapsWithHeadersOptions(rows, headers, opt)
}

// FromMapsWithHeaders writes CSV using explicit column order.
func FromMapsWithHeaders(rows []map[string]any, headers []string) (string, error) {
	return FromMapsWithHeadersOptions(rows, headers, Options{})
}

// FromMapsWithHeadersOptions writes CSV with explicit headers and options.
func FromMapsWithHeadersOptions(rows []map[string]any, headers []string, opt Options) (string, error) {
	var buf bytes.Buffer
	if opt.UseBOM {
		buf.WriteString("\uFEFF")
	}
	w := csv.NewWriter(&buf)
	w.Comma = opt.comma()
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

// WriteTo writes CSV rows to w.
func WriteTo(w io.Writer, rows []map[string]any, opt Options) error {
	body, err := FromMapsWithOptions(rows, opt)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, body)
	return err
}

// ToMaps parses CSV text into row maps keyed by header.
func ToMaps(data string) ([]map[string]string, error) {
	return ToMapsWithOptions(data, Options{})
}

// ToMapsWithOptions parses CSV text with delimiter options (BOM is stripped).
func ToMapsWithOptions(data string, opt Options) ([]map[string]string, error) {
	data = strings.TrimPrefix(data, "\uFEFF")
	return ToMapsReaderWithOptions(strings.NewReader(data), opt)
}

// ToMapsReader parses CSV from a reader.
func ToMapsReader(r io.Reader) ([]map[string]string, error) {
	return ToMapsReaderWithOptions(r, Options{})
}

// ToMapsReaderWithOptions parses CSV from a reader with options.
func ToMapsReaderWithOptions(r io.Reader, opt Options) ([]map[string]string, error) {
	// Strip UTF-8 BOM if present at the start of the stream.
	br := newBOMReader(r)
	cr := csv.NewReader(br)
	cr.Comma = opt.comma()
	cr.TrimLeadingSpace = true
	cr.ReuseRecord = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	headers := append([]string(nil), records[0]...)
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
	return ResponseWithOptions(filename, rows, Options{UseBOM: true})
}

// ResponseWithOptions builds a CSV download with options.
func ResponseWithOptions(filename string, rows []map[string]any, opt Options) *http.Response {
	body, err := FromMapsWithOptions(rows, opt)
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

// bomReader strips a leading UTF-8 BOM from r.
type bomReader struct {
	r   io.Reader
	buf []byte
	n   int
}

func newBOMReader(r io.Reader) io.Reader {
	br := &bomReader{r: r, buf: make([]byte, 3)}
	n, err := io.ReadFull(r, br.buf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		br.n = n
		return br
	}
	if err != nil {
		br.n = 0
		br.buf = nil
		return r
	}
	if n == 3 && br.buf[0] == 0xEF && br.buf[1] == 0xBB && br.buf[2] == 0xBF {
		br.n = 0
		br.buf = nil
		return br
	}
	br.n = n
	return br
}

func (b *bomReader) Read(p []byte) (int, error) {
	if b.n > 0 {
		n := copy(p, b.buf[:b.n])
		b.buf = b.buf[n:]
		b.n -= n
		return n, nil
	}
	return b.r.Read(p)
}
