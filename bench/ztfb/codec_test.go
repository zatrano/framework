package main

import (
	"encoding/json"
	"testing"

	goccyjson "github.com/goccy/go-json"
)

func TestGoccyMatchesEncodingJSON(t *testing.T) {
	samples := []any{
		map[string]string{"message": "Hello, World!"},
		map[string]string{"html": "<tag>&\"'"},
		map[string]any{"n": 1, "ok": true, "s": "a/b"},
		[]any{"<", ">"},
	}
	for _, sample := range samples {
		want, err := json.Marshal(sample)
		if err != nil {
			t.Fatal(err)
		}
		got, err := goccyjson.Marshal(sample)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("goccy %s encoding/json %s", got, want)
		}
	}
}
