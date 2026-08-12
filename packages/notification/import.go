package notification

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/packages/export/csv"
	"github.com/zatrano/framework/packages/export/xlsx"
)

// ImportRecipients parses CSV or Excel (.xlsx) content into recipients.
func ImportRecipients(filename string, r io.Reader) ([]Recipient, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	var rows []map[string]string
	var err error
	switch ext {
	case ".xlsx", ".xlsm":
		rows, err = xlsx.ToMapsReader(r)
	case ".csv", ".txt", "":
		rows, err = csv.ToMapsReader(r)
	default:
		return nil, fmt.Errorf("notification: unsupported import format %q (use .csv or .xlsx)", ext)
	}
	if err != nil {
		return nil, err
	}
	out := make([]Recipient, 0, len(rows))
	for i, row := range rows {
		rec, err := RecipientFromMap(row)
		if err != nil {
			return nil, fmt.Errorf("notification: row %d: %w", i+2, err)
		}
		out = append(out, rec)
	}
	return out, nil
}

// ImportRecipientsBytes is a convenience wrapper around ImportRecipients.
func ImportRecipientsBytes(filename string, data []byte) ([]Recipient, error) {
	return ImportRecipients(filename, bytes.NewReader(data))
}
