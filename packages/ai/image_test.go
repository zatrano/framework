package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/ai"
)

func TestFakeGenerateImage(t *testing.T) {
	m := ai.New()
	resp, err := m.GenerateImage(context.Background(), ai.ImageRequest{Prompt: "a red cube"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || !strings.Contains(resp.Data[0].URL, "red") {
		t.Fatalf("%+v", resp)
	}
	if !m.Supports(ai.CapImage, "fake") {
		t.Fatal("cap")
	}
}

func TestOpenAIGenerateImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Fatal("auth")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["prompt"] != "cat" {
			t.Fatalf("%v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"created": 1700000000,
			"data":    []map[string]string{{"url": "https://example.com/cat.png"}},
		})
	}))
	defer srv.Close()

	m := ai.New()
	m.Extend("oa", &ai.OpenAIDriver{
		BaseURL:    srv.URL + "/v1",
		APIKey:     "k",
		HTTPClient: srv.Client(),
	})
	resp, err := m.Using("oa").GenerateImage(context.Background(), ai.ImageRequest{Prompt: "cat", Size: "1024x1024"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "https://example.com/cat.png" {
		t.Fatalf("%+v", resp)
	}
}

func TestGenerateImageRequiresPrompt(t *testing.T) {
	_, err := ai.New().GenerateImage(context.Background(), ai.ImageRequest{})
	if err == nil || ai.Classify(err) != ai.KindInvalid {
		t.Fatalf("%v", err)
	}
}
