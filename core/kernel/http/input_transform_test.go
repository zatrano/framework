package http

import (
	"encoding/json"
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
	body := []byte(`{"name":"  Ada  ","note":""}`)
	req := testRequest("POST", "/", body)
	req.Raw().Header.Set("Content-Type", "application/json")
	req.TransformInputs(trimSpaces)
	req.TransformInputs(dropEmpty)

	var dest map[string]string
	if err := req.JSON(&dest); err != nil {
		t.Fatal(err)
	}
	if dest["name"] != "  Ada  " {
		t.Fatalf("JSON dest must stay raw, got %#v", dest)
	}
}

func TestTransformInputsAffectsInputHelpers(t *testing.T) {
	body := []byte(`{"name":"  Ada  ","note":""}`)
	req := testRequest("POST", "/", body)
	req.Raw().Header.Set("Content-Type", "application/json")
	req.TransformInputs(trimSpaces)
	req.TransformInputs(dropEmpty)
	if got := req.Input("name"); got != "Ada" {
		t.Fatalf("Input name=%q", got)
	}
}

func TestTransformInputsIdempotent(t *testing.T) {
	body := []byte(`{"name":"  Ada  "}`)
	req := testRequest("POST", "/", body)
	req.Raw().Header.Set("Content-Type", "application/json")
	req.TransformInputs(trimSpaces)
	_ = req.Input("name")
	req.TransformInputs(trimSpaces)
	if got := req.Input("name"); got != "Ada" {
		t.Fatalf("got %q", got)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	payload := map[string]any{"ok": true}
	raw, _ := json.Marshal(payload)
	req := testRequest("POST", "/", raw)
	req.Raw().Header.Set("Content-Type", "application/json")
	var dest map[string]any
	if err := req.JSON(&dest); err != nil {
		t.Fatal(err)
	}
	if dest["ok"] != true {
		t.Fatalf("%#v", dest)
	}
}
