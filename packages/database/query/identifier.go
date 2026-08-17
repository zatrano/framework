package query

import (
	"fmt"
	"strings"
	"unicode"
)

// sanitizeIdentifier allows table/column references used in ORDER BY / GROUP BY.
// Accepts: col, table.col, schema.table.col — letters, digits, underscore, and dots.
// Rejects SQL metacharacters to prevent injection via user-controlled sort fields.
func sanitizeIdentifier(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty identifier")
	}
	if len(name) > 128 {
		return "", fmt.Errorf("identifier too long")
	}
	parts := strings.Split(name, ".")
	if len(parts) > 3 {
		return "", fmt.Errorf("invalid identifier")
	}
	for _, part := range parts {
		if part == "" || !isSimpleIdent(part) {
			return "", fmt.Errorf("invalid identifier")
		}
	}
	return name, nil
}

func isSimpleIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

func sanitizeOrderDirection(dir string) string {
	switch strings.ToLower(strings.TrimSpace(dir)) {
	case "desc", "descending":
		return "desc"
	default:
		return "asc"
	}
}

// sanitizeOperator allows a small set of comparison operators for Having/Where helpers.
func sanitizeOperator(op string) (string, error) {
	op = strings.TrimSpace(op)
	switch strings.ToLower(op) {
	case "=", "!=", "<>", "<", ">", "<=", ">=", "like", "ilike", "is", "is not", "in", "not in":
		return op, nil
	default:
		return "", fmt.Errorf("invalid operator")
	}
}
