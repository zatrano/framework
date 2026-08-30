package ai_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/ai"
)

func TestFakeChatStream(t *testing.T) {
	m := ai.New()
	ch, err := m.ChatStream(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "hello stream"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var built strings.Builder
	var done bool
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatal(chunk.Err)
		}
		built.WriteString(chunk.Delta)
		if chunk.Done {
			done = true
			if chunk.Usage == nil || chunk.Usage.TotalTokens < 1 {
				t.Fatalf("usage=%v", chunk.Usage)
			}
		}
	}
	if !done {
		t.Fatal("expected Done chunk")
	}
	if !strings.Contains(built.String(), "hello stream") {
		t.Fatalf("%q", built.String())
	}
}

func TestChatStreamFallbackSetup(t *testing.T) {
	m := ai.New()
	m.SetDefaults(ai.Defaults{
		Timeout: time.Second,
		Retry:   ai.RetryPolicy{MaxRetries: 0, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond},
	})
	m.Extend("broken", failStreamDriver{err: &ai.Error{Kind: ai.KindUnavailable, Status: 500, Err: errors.New("down")}})
	m.Extend("ok", ai.FakeDriver{})
	m.SetProfile("s", ai.Profile{Providers: []string{"broken", "ok"}})
	ch, err := m.Profile("s").ChatStream(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "via-fallback"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var built strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatal(chunk.Err)
		}
		built.WriteString(chunk.Delta)
	}
	if !strings.Contains(built.String(), "via-fallback") {
		t.Fatalf("%q", built.String())
	}
}

func TestChatStreamAuthNoFallback(t *testing.T) {
	m := ai.New()
	m.Extend("auth", failStreamDriver{err: &ai.Error{Kind: ai.KindAuth, Status: 401, Err: errors.New("bad")}})
	m.Extend("ok", ai.FakeDriver{})
	m.SetProfile("s", ai.Profile{Providers: []string{"auth", "ok"}})
	_, err := m.Profile("s").ChatStream(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil || ai.Classify(err) != ai.KindAuth {
		t.Fatalf("%v", err)
	}
}

type failStreamDriver struct{ err error }

func (failStreamDriver) Name() string { return "fail-stream" }
func (d failStreamDriver) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	return nil, d.err
}
func (d failStreamDriver) ChatStream(ctx context.Context, req ai.ChatRequest) (<-chan ai.StreamChunk, error) {
	return nil, d.err
}

func TestOpenAIChatStreamSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path %s", r.URL.Path)
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["stream"] != true {
			t.Fatalf("stream=%v", payload["stream"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		frames := []string{
			`data: {"id":"chatcmpl-s1","model":"gpt-stream","choices":[{"delta":{"content":"Hel"}}]}`,
			`data: {"id":"chatcmpl-s1","model":"gpt-stream","choices":[{"delta":{"content":"lo"}}]}`,
			`data: {"id":"chatcmpl-s1","model":"gpt-stream","choices":[{"delta":{}}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`,
			`data: [DONE]`,
		}
		for _, f := range frames {
			_, _ = w.Write([]byte(f + "\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer srv.Close()

	d := &ai.OpenAIDriver{
		BaseURL:    srv.URL + "/v1",
		APIKey:     "k",
		Model:      "gpt-stream",
		HTTPClient: srv.Client(),
	}
	ch, err := d.ChatStream(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var built strings.Builder
	var usage *ai.Usage
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatal(chunk.Err)
		}
		built.WriteString(chunk.Delta)
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}
	if built.String() != "Hello" {
		t.Fatalf("%q", built.String())
	}
	if usage == nil || usage.TotalTokens != 3 {
		t.Fatalf("%+v", usage)
	}
}

func TestOpenAIChatStreamToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		frames := []string{
			`data: {"id":"s2","model":"m","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"lookup","arguments":""}}]}}]}`,
			`data: {"id":"s2","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"q\":"}}]}}]}`,
			`data: {"id":"s2","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"x\"}"}}]}}]}`,
			`data: {"id":"s2","choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
			`data: [DONE]`,
		}
		for _, f := range frames {
			_, _ = w.Write([]byte(f + "\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer srv.Close()

	d := &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()}
	ch, err := d.ChatStream(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "x"}},
		Tools:    []ai.Tool{ai.FunctionTool("lookup", "", nil)},
	})
	if err != nil {
		t.Fatal(err)
	}
	text, calls, _, finish, err := ai.CollectStream(ch)
	if err != nil {
		t.Fatal(err)
	}
	if text != "" || finish != "tool_calls" || len(calls) != 1 {
		t.Fatalf("text=%q finish=%q calls=%+v", text, finish, calls)
	}
	if calls[0].Function.Name != "lookup" || calls[0].Function.Arguments != `{"q":"x"}` {
		t.Fatalf("%+v", calls[0])
	}
}

func TestFakeStreamToolCalls(t *testing.T) {
	m := ai.New()
	ch, err := m.ChatStream(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "run tool"}},
		Tools:    []ai.Tool{ai.FunctionTool("lookup", "", nil)},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, calls, _, finish, err := ai.CollectStream(ch)
	if err != nil || finish != "tool_calls" || len(calls) != 1 {
		t.Fatalf("err=%v finish=%q calls=%v", err, finish, calls)
	}
}

func TestOpenAIChatStreamHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate"}`))
	}))
	defer srv.Close()
	d := &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", APIKey: "k", HTTPClient: srv.Client()}
	_, err := d.ChatStream(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if ai.Classify(err) != ai.KindRateLimit {
		t.Fatalf("%v kind=%v", err, ai.Classify(err))
	}
}
