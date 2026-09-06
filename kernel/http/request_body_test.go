package http_test

import (
	"bytes"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
)

func TestJSONThenBodyDoesNotClose(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader(`{"name":"Ada"}`))
	raw.Header.Set("Content-Type", "application/json")
	req := http.NewRequest(raw)

	var dest map[string]string
	if err := req.JSON(&dest); err != nil {
		t.Fatal(err)
	}
	if dest["name"] != "Ada" {
		t.Fatalf("dest=%v", dest)
	}
	body, err := req.Body()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, []byte(`{"name":"Ada"}`)) {
		t.Fatalf("body=%s", body)
	}
}

func TestBodyTooLarge(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader(strings.Repeat("x", 64)))
	req := http.NewRequest(raw)
	req.SetMaxBodyBytes(8)
	if _, err := req.Body(); err != http.ErrBodyTooLarge {
		t.Fatalf("err=%v", err)
	}
}

func TestJSONTooLarge(t *testing.T) {
	payload, _ := json.Marshal(map[string]string{"blob": strings.Repeat("a", 64)})
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", bytes.NewReader(payload))
	raw.Header.Set("Content-Type", "application/json")
	req := http.NewRequest(raw)
	req.SetMaxBodyBytes(16)
	var dest map[string]string
	if err := req.JSON(&dest); err != http.ErrBodyTooLarge {
		t.Fatalf("err=%v", err)
	}
}
