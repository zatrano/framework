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

func TestChatJSONFake(t *testing.T) {
	m := ai.New()
	var out struct {
		Text string `json:"text"`
	}
	resp, err := m.ChatJSON(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "hello json"}},
	}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if out.Text == "" || resp == nil {
		t.Fatalf("out=%+v resp=%v", out, resp)
	}
	if !strings.Contains(out.Text, "hello json") {
		t.Fatalf("%q", out.Text)
	}
}

func TestUnmarshalJSONFence(t *testing.T) {
	resp := &ai.ChatResponse{Message: ai.Message{Content: "```json\n{\"a\":1}\n```"}}
	var out map[string]int
	if err := resp.UnmarshalJSON(&out); err != nil {
		t.Fatal(err)
	}
	if out["a"] != 1 {
		t.Fatalf("%v", out)
	}
}

func TestOpenAIResponseFormatJSONObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		rf, _ := payload["response_format"].(map[string]any)
		if rf == nil || rf["type"] != "json_object" {
			t.Fatalf("response_format=%v", payload["response_format"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1","model":"m","choices":[{"message":{"role":"assistant","content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer srv.Close()

	d := &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", APIKey: "k", HTTPClient: srv.Client()}
	resp, err := d.Chat(context.Background(), ai.ChatRequest{
		Messages:       []ai.Message{{Role: "user", Content: "x"}},
		ResponseFormat: ai.JSONObject(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		OK bool `json:"ok"`
	}
	if err := resp.UnmarshalJSON(&out); err != nil || !out.OK {
		t.Fatalf("%v %+v", err, out)
	}
}

func TestOpenAIResponseFormatJSONSchema(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		rf, _ := payload["response_format"].(map[string]any)
		if rf["type"] != "json_schema" {
			t.Fatalf("%v", rf)
		}
		js, _ := rf["json_schema"].(map[string]any)
		if js["name"] != "person" || js["strict"] != true {
			t.Fatalf("%v", js)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1","model":"m","choices":[{"message":{"role":"assistant","content":"{\"name\":\"Ada\"}"}}],"usage":{"total_tokens":1}}`))
	}))
	defer srv.Close()

	d := &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()}
	resp, err := d.Chat(context.Background(), ai.ChatRequest{
		Messages:       []ai.Message{{Role: "user", Content: "x"}},
		ResponseFormat: ai.JSONSchema("person", schema),
	})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Name string `json:"name"`
	}
	if err := resp.UnmarshalJSON(&out); err != nil || out.Name != "Ada" {
		t.Fatalf("%v %+v", err, out)
	}
}
