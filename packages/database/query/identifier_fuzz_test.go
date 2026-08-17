package query

import "testing"

func FuzzSanitizeIdentifier(f *testing.F) {
	for _, s := range []string{"id", "users.name", "a.b.c", "id; drop", "1=1", "../x", "col--", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = sanitizeIdentifier(in)
	})
}
