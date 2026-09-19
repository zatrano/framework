package http

import (
	"bytes"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func trimSpaces(_ string, value string) (string, bool) {
	return strings.TrimSpace(value), true
}

func dropEmpty(_ string, value string) (string, bool) {
	if value == "" {
		return "", false
	}
	return value, true
}

func TestTransformInputsLazyJSONDoesNotParseOverlay(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader(`{"name":"  Ada  ","note":""}`))
	raw.Header.Set("Content-Type", "application/json")
	req := NewRequest(raw)
	req.TransformInputs(trimSpaces)
	req.TransformInputs(dropEmpty)

	var dest map[string]string
	if err := req.JSON(&dest); err != nil {
		t.Fatal(err)
	}
	if dest["name"] != "  Ada  " {
		t.Fatalf("JSON dest must stay raw, got %#v", dest)
	}
	if req.jsonRead {
		t.Fatal("JSON() must not parse the Input overlay")
	}

	all := req.All()
	if all["name"] != "Ada" {
		t.Fatalf("All name=%q", all["name"])
	}
	if _, ok := all["note"]; ok {
		t.Fatalf("note should be dropped, got %#v", all)
	}
	if !req.jsonRead {
		t.Fatal("All() should parse the overlay")
	}
}

func TestTransformInputsRawFormStaysUntransformedUntilInput(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader("name=%20Ada%20"))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := NewRequest(raw)
	req.TransformInputs(trimSpaces)

	if err := req.Raw().ParseForm(); err != nil {
		t.Fatal(err)
	}
	if got := req.Raw().Form.Get("name"); got == "Ada" {
		t.Fatal("Raw().Form must stay untransformed until Input/All")
	}
	if req.Input("name") != "Ada" {
		t.Fatalf("Input name=%q", req.Input("name"))
	}
	if req.Raw().Form.Get("name") != "Ada" {
		t.Fatalf("Raw().Form after Input=%q", req.Raw().Form.Get("name"))
	}
}

func TestTransformInputsApplyBeforeMerge(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader("name=%20Ada%20"))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := NewRequest(raw)
	req.TransformInputs(trimSpaces)
	req.Merge(map[string]string{"role": "  admin  "})
	if req.Input("name") != "Ada" {
		t.Fatalf("name=%q", req.Input("name"))
	}
	if req.Input("role") != "  admin  " {
		t.Fatalf("merged role should not be re-trimmed, got %q", req.Input("role"))
	}
}

func TestTransformInputsImmediateAfterFirstApply(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader("name=Ada"))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := NewRequest(raw)
	req.TransformInputs(trimSpaces)
	if req.Input("name") != "Ada" {
		t.Fatal(req.Input("name"))
	}
	req.TransformInputs(func(key, value string) (string, bool) {
		if key == "name" {
			return strings.ToUpper(value), true
		}
		return value, true
	})
	if req.Input("name") != "ADA" {
		t.Fatalf("late transform name=%q", req.Input("name"))
	}
}

func BenchmarkJSONWithQueuedTransforms(b *testing.B) {
	body := []byte(`{"ok":true,"name":"Ada"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw := httptest.NewRequest(stdhttp.MethodPost, "/json", bytes.NewReader(body))
		raw.Header.Set("Content-Type", "application/json")
		req := NewRequest(raw)
		req.TransformInputs(trimSpaces)
		req.TransformInputs(dropEmpty)
		var dest struct {
			OK bool `json:"ok"`
		}
		if err := req.JSON(&dest); err != nil || !dest.OK {
			b.Fatalf("err=%v dest=%#v", err, dest)
		}
	}
}

func BenchmarkJSONAfterEagerInputOverlay(b *testing.B) {
	body := []byte(`{"ok":true,"name":"Ada"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw := httptest.NewRequest(stdhttp.MethodPost, "/json", bytes.NewReader(body))
		raw.Header.Set("Content-Type", "application/json")
		req := NewRequest(raw)
		req.TransformInputs(trimSpaces)
		req.TransformInputs(dropEmpty)
		_ = req.All()
		var dest struct {
			OK bool `json:"ok"`
		}
		if err := json.Unmarshal(mustBody(b, req), &dest); err != nil || !dest.OK {
			b.Fatalf("err=%v dest=%#v", err, dest)
		}
	}
}

func mustBody(tb testing.TB, req *Request) []byte {
	tb.Helper()
	raw, err := req.Body()
	if err != nil {
		tb.Fatal(err)
	}
	return raw
}
