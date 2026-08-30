package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/ai"
)

func TestFakeToolCalls(t *testing.T) {
	m := ai.New()
	params := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)
	resp, err := m.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "weather Istanbul"}},
		Tools:    []ai.Tool{ai.FunctionTool("get_weather", "Weather lookup", params)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasToolCalls() || resp.FinishReason != "tool_calls" {
		t.Fatalf("%+v", resp)
	}
	call := resp.Message.ToolCalls[0]
	if call.Function.Name != "get_weather" {
		t.Fatal(call.Function.Name)
	}
	var args struct {
		Query string `json:"query"`
	}
	if err := call.UnmarshalArguments(&args); err != nil {
		t.Fatal(err)
	}
	if args.Query != "weather Istanbul" {
		t.Fatalf("%q", args.Query)
	}

	// Round-trip: feed tool result back
	msgs := []ai.Message{
		{Role: "user", Content: "weather Istanbul"},
		ai.AssistantToolCalls(call),
		ai.ToolResultMessage(call.ID, `{"temp":22}`),
	}
	resp2, err := m.Chat(context.Background(), ai.ChatRequest{
		Messages:   msgs,
		Tools:      []ai.Tool{ai.FunctionTool("get_weather", "", params)},
		ToolChoice: ai.ToolChoiceNone(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp2.HasToolCalls() {
		t.Fatal("expected text reply with ToolChoiceNone")
	}
}

func TestOpenAIToolsRequestBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		tools, _ := payload["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("tools=%v", payload["tools"])
		}
		if payload["tool_choice"] != "auto" {
			t.Fatalf("tool_choice=%v", payload["tool_choice"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-t1","model":"gpt-test","created":1700000000,
			"choices":[{
				"finish_reason":"tool_calls",
				"message":{
					"role":"assistant",
					"tool_calls":[{
						"id":"call_1","type":"function",
						"function":{"name":"lookup","arguments":"{\"q\":\"x\"}"}
					}]
				}
			}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer srv.Close()

	d := &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", APIKey: "k", HTTPClient: srv.Client()}
	resp, err := d.Chat(context.Background(), ai.ChatRequest{
		Messages:   []ai.Message{{Role: "user", Content: "x"}},
		Tools:      []ai.Tool{ai.FunctionTool("lookup", "Lookup", nil)},
		ToolChoice: ai.ToolChoiceAuto(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.FinishReason != "tool_calls" || !resp.HasToolCalls() {
		t.Fatalf("%+v", resp)
	}
	if resp.Message.ToolCalls[0].Function.Name != "lookup" {
		t.Fatal(resp.Message.ToolCalls[0])
	}
}

func TestToolChoiceFunctionAPI(t *testing.T) {
	c := ai.ToolChoiceFunction("get_weather")
	api := c // exercise via OpenAI body through apply — use Chat with mock
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		tc, _ := payload["tool_choice"].(map[string]any)
		fn, _ := tc["function"].(map[string]any)
		if tc["type"] != "function" || fn["name"] != "get_weather" {
			t.Fatalf("%v", payload["tool_choice"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}],"usage":{}}`))
	}))
	defer srv.Close()
	d := &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()}
	_, err := d.Chat(context.Background(), ai.ChatRequest{
		Messages:   []ai.Message{{Role: "user", Content: "x"}},
		Tools:      []ai.Tool{ai.FunctionTool("get_weather", "", nil)},
		ToolChoice: api,
	})
	if err != nil {
		t.Fatal(err)
	}
}
