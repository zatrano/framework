package export

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/packages/export/csv"
	"github.com/zatrano/framework/packages/export/xlsx"
)

// ToMaps imports tabular data from a CSV or Excel (.xlsx/.xlsm) payload.
// Format is detected from filename extension.
func ToMaps(filename string, r io.Reader) ([]map[string]string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".xlsx", ".xlsm":
		return xlsx.ToMapsReader(r)
	case ".csv", ".txt", "":
		return csv.ToMapsReader(r)
	default:
		return nil, fmt.Errorf("export: unsupported import format %q (use .csv or .xlsx)", ext)
	}
}

// ToMapsBytes is a convenience wrapper around ToMaps.
func ToMapsBytes(filename string, data []byte) ([]map[string]string, error) {
	return ToMaps(filename, bytes.NewReader(data))
}
