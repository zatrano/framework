package fuzz_test

import (
	"bytes"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/validation"
)

// FuzzValidateMake fuzzes validation.Make with arbitrary field values.
func FuzzValidateMake(f *testing.F) {
	for _, s := range []string{"", "a@b.c", "not-an-email", "123", strings.Repeat("x", 100)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, value string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("validation panic value=%q: %v", value, r)
			}
		}()
		if len(value) > 4096 {
			return
		}
		v := validation.Make(
			map[string]string{"email": value, "name": value},
			map[string]string{"email": "required|email", "name": "nullable|max:100"},
		)
		_ = v.Passes()
		_, _ = v.Validated()
	})
}

// FuzzHTTPJSONBinding fuzzes Request.JSON / All against arbitrary bodies.
func FuzzHTTPJSONBinding(f *testing.F) {
	seeds := []string{
		`{}`,
		`{"a":1}`,
		`{"nested":{"b":true}}`,
		`[]`,
		`null`,
		`{"a":` + strings.Repeat(`{"b":`, 20) + `1` + strings.Repeat(`}`, 20) + `}`,
		`not-json`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("json binding panic: %v", r)
			}
		}()
		if len(body) > 65536 {
			return
		}
		raw := httptest.NewRequest(stdhttp.MethodPost, "/form", bytes.NewReader(body))
		raw.Header.Set("Content-Type", "application/json")
		req := http.NewRequest(raw)
		var dest map[string]any
		_ = req.JSON(&dest)
		_ = req.All()
	})
}
