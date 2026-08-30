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

func TestMessageMultimodalJSON(t *testing.T) {
	m := ai.UserVision("what is this?", "https://example.com/a.png")
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	arr, ok := payload["content"].([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("%s", raw)
	}

	var back ai.Message
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.TextContent() != "what is this?" || !back.HasImages() {
		t.Fatalf("%+v", back)
	}
}

func TestFakeVisionChat(t *testing.T) {
	mgr := ai.New()
	resp, err := mgr.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{ai.UserVision("describe", "https://example.com/x.jpg")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Message.Content, "vision") {
		t.Fatalf("%q", resp.Message.Content)
	}
	if !mgr.Supports(ai.CapVision, "fake") {
		t.Fatal("cap")
	}
}

func TestOpenAIVisionRequestBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		msgs, _ := payload["messages"].([]any)
		if len(msgs) != 1 {
			t.Fatalf("%v", payload["messages"])
		}
		msg, _ := msgs[0].(map[string]any)
		content, _ := msg["content"].([]any)
		if len(content) != 2 {
			t.Fatalf("%v", msg["content"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1","choices":[{"message":{"role":"assistant","content":"cat"}}],"usage":{"total_tokens":1}}`))
	}))
	defer srv.Close()

	d := &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()}
	resp, err := d.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{ai.UserVision("what?", "https://example.com/cat.png")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message.Content != "cat" {
		t.Fatalf("%q", resp.Message.Content)
	}
}
