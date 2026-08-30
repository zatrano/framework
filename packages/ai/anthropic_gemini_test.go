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

func TestAnthropicChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "ak" {
			t.Fatal("api key")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Fatal("version")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["system"] != "be brief" {
			t.Fatalf("system=%v", body["system"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "msg_1",
			"model":       "claude-test",
			"stop_reason": "end_turn",
			"content":     []map[string]string{{"type": "text", "text": "hello from claude"}},
			"usage":       map[string]int{"input_tokens": 3, "output_tokens": 5},
		})
	}))
	defer srv.Close()

	d := &ai.AnthropicDriver{
		BaseURL:    srv.URL,
		APIKey:     "ak",
		Model:      "claude-test",
		HTTPClient: srv.Client(),
	}
	m := ai.New()
	m.Extend("anthropic", d)
	resp, err := m.Using("anthropic").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: "be brief"},
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Message.Content, "claude") {
		t.Fatalf("%v", resp.Message.Content)
	}
	if resp.FinishReason != "stop" || resp.Usage.TotalTokens != 8 {
		t.Fatalf("%+v", resp)
	}
	if !m.Supports(ai.CapVision, "anthropic") || !m.Supports(ai.CapTools, "anthropic") || m.Supports(ai.CapImage, "anthropic") {
		t.Fatal("caps")
	}
}

func TestAnthropicTools(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["tools"]; !ok {
			t.Fatal("missing tools")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_t", "model": "claude-test", "stop_reason": "tool_use",
			"content": []map[string]any{{
				"type": "tool_use", "id": "call_1", "name": "lookup",
				"input": map[string]string{"q": "x"},
			}},
			"usage": map[string]int{"input_tokens": 1, "output_tokens": 2},
		})
	}))
	defer srv.Close()
	m := ai.New()
	m.Extend("a", &ai.AnthropicDriver{BaseURL: srv.URL, APIKey: "k", HTTPClient: srv.Client()})
	resp, err := m.Using("a").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "use tool"}},
		Tools:    []ai.Tool{ai.FunctionTool("lookup", "look", nil)},
	})
	if err != nil || !resp.HasToolCalls() || resp.Message.ToolCalls[0].Function.Name != "lookup" {
		t.Fatalf("%v %+v", err, resp)
	}
}

func TestAnthropicHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("%s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	m := ai.New()
	m.Extend("a", &ai.AnthropicDriver{BaseURL: srv.URL, APIKey: "k", HTTPClient: srv.Client()})
	st := m.CheckHealth(context.Background(), "a")
	if !st[0].OK {
		t.Fatalf("%+v", st[0])
	}
}

func TestBuildDriverAnthropic(t *testing.T) {
	d, err := ai.BuildDriver(ai.ProviderOptions{Driver: "claude", APIKey: "k", Model: "claude-x"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != "anthropic" {
		t.Fatal(d.Name())
	}
}

func TestGeminiChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "gk" {
			t.Fatal("key")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content":      map[string]any{"parts": []map[string]string{{"text": "hello gemini"}}},
				"finishReason": "STOP",
			}},
			"usageMetadata": map[string]int{
				"promptTokenCount": 2, "candidatesTokenCount": 4, "totalTokenCount": 6,
			},
		})
	}))
	defer srv.Close()

	m := ai.New()
	m.Extend("gemini", &ai.GeminiDriver{
		BaseURL:    srv.URL,
		APIKey:     "gk",
		Model:      "gemini-test",
		HTTPClient: srv.Client(),
	})
	resp, err := m.Using("gemini").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Message.Content, "gemini") {
		t.Fatalf("%v", resp.Message.Content)
	}
	if resp.Usage.TotalTokens != 6 {
		t.Fatalf("%+v", resp.Usage)
	}
}

func TestGeminiHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v1beta/models") {
			t.Fatalf("%s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer srv.Close()
	m := ai.New()
	m.Extend("g", &ai.GeminiDriver{BaseURL: srv.URL, APIKey: "k", HTTPClient: srv.Client()})
	st := m.CheckHealth(context.Background(), "g")
	if !st[0].OK {
		t.Fatalf("%+v", st[0])
	}
}

func TestGeminiEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":embedContent") {
			t.Fatalf("%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"embedding": map[string]any{"values": []float64{0.1, 0.2}},
		})
	}))
	defer srv.Close()
	m := ai.New()
	m.Extend("g", &ai.GeminiDriver{BaseURL: srv.URL, APIKey: "k", HTTPClient: srv.Client()})
	resp, err := m.Using("g").Embed(context.Background(), ai.EmbedRequest{Input: []string{"hi"}})
	if err != nil || len(resp.Embeddings) != 1 || len(resp.Embeddings[0]) != 2 {
		t.Fatalf("%v %+v", err, resp)
	}
	if !m.Supports(ai.CapEmbed, "g") || !m.Supports(ai.CapStream, "g") {
		t.Fatal("caps")
	}
}
