package fuzz_test

import (
	"testing"

	"github.com/zatrano/framework/packages/database/query"
)

// FuzzQueryWhereToSQL fuzzes Where column/operator/value → ToSQL.
// Invalid identifiers must not panic; only crashes are failures.
func FuzzQueryWhereToSQL(f *testing.F) {
	seeds := []struct {
		col, op, val string
	}{
		{"id", "=", "1"},
		{"users.name", "LIKE", "%x%"},
		{"id; drop", "=", "1"},
		{"col--", "OR", "1"},
		{"", "=", ""},
		{"a.b.c", "IN", "1"},
	}
	for _, s := range seeds {
		f.Add(s.col, s.op, s.val)
	}
	f.Fuzz(func(t *testing.T, col, op, val string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("query Where/ToSQL panic col=%q op=%q: %v", col, op, r)
			}
		}()
		if len(col) > 256 || len(op) > 64 || len(val) > 1024 {
			return
		}
		b := query.New(nil, "sqlite", "users")
		b.Where(col, op, val)
		_, _ = b.ToSQL()
	})
}

// FuzzQueryWhereRaw fuzzes WhereRaw input for panics only (raw is intentional).
func FuzzQueryWhereRaw(f *testing.F) {
	for _, s := range []string{"1=1", "id = ?", "x); drop--", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("WhereRaw panic: %v", r)
			}
		}()
		if len(raw) > 2048 {
			return
		}
		b := query.New(nil, "sqlite", "users")
		b.WhereRaw(raw, 1)
		_, _ = b.ToSQL()
	})
}
