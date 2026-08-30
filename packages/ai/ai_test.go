package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/ai"
	zhttp "github.com/zatrano/framework/packages/http"
)

func TestAIChat(t *testing.T) {
	m := ai.New()
	resp, err := m.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "ping"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Message.Content, "ping") {
		t.Fatalf("%v", resp.Message.Content)
	}
	if resp.Usage.TotalTokens < 1 {
		t.Fatal("usage")
	}
}

func TestFakeDriver(t *testing.T) {
	d := ai.FakeDriver{}
	if d.Name() != "fake" {
		t.Fatal(d.Name())
	}
	resp, err := d.Chat(context.Background(), ai.ChatRequest{
		Model:    "zatrano-fake-1",
		Messages: []ai.Message{{Role: "user", Content: "hello world"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Message.Content, "hello world") {
		t.Fatalf("%v", resp.Message.Content)
	}
	if resp.Message.Role != "assistant" {
		t.Fatal(resp.Message.Role)
	}
}

func TestChatContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ai.New().Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected canceled context error")
	}
}

func TestDefaultsApplied(t *testing.T) {
	m := ai.New()
	temp := 0.2
	m.SetDefaults(ai.Defaults{
		Model:       "cfg-model",
		Temperature: &temp,
		MaxTokens:   64,
		Timeout:     time.Second,
	})
	resp, err := m.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Model != "cfg-model" {
		t.Fatalf("model=%q", resp.Model)
	}
}

func TestLogDriver(t *testing.T) {
	var logged string
	d := ai.LogDriver{
		Log: func(format string, args ...any) {
			logged = strings.TrimSpace(strings.ReplaceAll(format, "%q", "%s"))
			_ = args
			logged = "ok"
		},
	}
	resp, err := d.Chat(context.Background(), ai.ChatRequest{
		Model:    "m1",
		Messages: []ai.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || logged != "ok" {
		t.Fatalf("resp=%v logged=%q", resp, logged)
	}
}

func TestUsingNamedProvider(t *testing.T) {
	m := ai.New()
	m.Extend("primary", ai.FakeDriver{})
	resp, err := m.Using("primary").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "via-using"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Message.Content, "via-using") {
		t.Fatalf("%v", resp.Message.Content)
	}
}

type failDriver struct{ err error }

func (failDriver) Name() string { return "fail" }
func (d failDriver) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	return nil, d.err
}

func TestProfileFallback(t *testing.T) {
	m := ai.New()
	m.Extend("broken", failDriver{err: errors.New("boom")})
	m.Extend("ok", ai.FakeDriver{})
	m.SetProfile("content", ai.Profile{
		Providers: []string{"broken", "ok"},
		Model:     "profile-model",
	})
	resp, err := m.Profile("content").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "fallback"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Model != "profile-model" {
		t.Fatalf("model=%q", resp.Model)
	}
	if !strings.Contains(resp.Message.Content, "fallback") {
		t.Fatalf("%v", resp.Message.Content)
	}
}

func TestBootConfigProvidersAndProfiles(t *testing.T) {
	m := ai.New()
	err := m.BootConfig(map[string]any{
		"default": "local",
		"model":   "base-model",
		"timeout": 15,
		"providers": map[string]any{
			"local": map[string]any{
				"driver": "fake",
			},
		},
		"profiles": map[string]any{
			"support": map[string]any{
				"providers": []any{"local"},
				"model":     "support-model",
			},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := m.Profile("support").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Model != "support-model" {
		t.Fatalf("model=%q", resp.Model)
	}
	resp2, err := m.Using("local").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil || !strings.Contains(resp2.Message.Content, "x") {
		t.Fatalf("using local: %v %v", err, resp2)
	}
}

func TestFakeEmbed(t *testing.T) {
	m := ai.New()
	resp, err := m.Embed(context.Background(), ai.EmbedRequest{Input: []string{"abc"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Embeddings) != 1 || len(resp.Embeddings[0]) != 3 {
		t.Fatalf("%+v", resp)
	}
}

func TestOpenAIDriverParse(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
		content string
		prompt  int
		comp    int
	}{
		{
			name:    "success",
			status:  http.StatusOK,
			body:    `{"id":"chatcmpl-1","model":"gpt-test","created":1700000000,"choices":[{"message":{"role":"assistant","content":"hi there"}}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`,
			content: "hi there",
			prompt:  3,
			comp:    2,
		},
		{
			name:    "missing choices",
			status:  http.StatusOK,
			body:    `{"id":"chatcmpl-2","choices":[]}`,
			wantErr: true,
		},
		{
			name:    "http error",
			status:  http.StatusUnauthorized,
			body:    `{"error":{"message":"bad key"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" {
					t.Fatalf("path %s", r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
					t.Fatalf("auth %q", got)
				}
				var payload map[string]any
				_ = json.NewDecoder(r.Body).Decode(&payload)
				if payload["temperature"] == nil || payload["max_tokens"] == nil {
					t.Fatalf("expected temperature/max_tokens in body: %#v", payload)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			d := &ai.OpenAIDriver{
				BaseURL:    srv.URL + "/v1",
				APIKey:     "test-key",
				Model:      "gpt-test",
				HTTPClient: srv.Client(),
			}
			temp := 0.5
			resp, err := d.Chat(context.Background(), ai.ChatRequest{
				Messages:    []ai.Message{{Role: "user", Content: "ping"}},
				Temperature: &temp,
				MaxTokens:   32,
			})
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if resp.Message.Content != tt.content {
				t.Fatalf("content=%q", resp.Message.Content)
			}
			if resp.Usage.PromptTokens != tt.prompt || resp.Usage.CompletionTokens != tt.comp {
				t.Fatalf("usage=%+v", resp.Usage)
			}
			if resp.ID == "" || resp.Model == "" {
				t.Fatalf("%+v", resp)
			}
		})
	}
}

func TestOpenAIHelper(t *testing.T) {
	d := ai.OpenAI("sk-test")
	od, ok := d.(*ai.OpenAIDriver)
	if !ok {
		t.Fatalf("%T", d)
	}
	if od.APIKey != "sk-test" || od.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("%+v", od)
	}
	compat := ai.OpenAICompatible("http://localhost:11434/v1", "", "llama3")
	if compat.Name() != "openai_compatible" {
		t.Fatal(compat.Name())
	}
}

func TestDemoChatHandler(t *testing.T) {
	mgr := ai.New()
	handler := ai.DemoChatHandler(mgr)

	raw := httptest.NewRequest(http.MethodPost, "/demo/ai/chat", bytes.NewBufferString(`{"message":"ping"}`))
	raw.Header.Set("Content-Type", "application/json")
	req := zhttp.NewRequest(raw)
	resp := handler(req)
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
	body := string(resp.Content())
	if !strings.Contains(body, "ping") {
		t.Fatalf("%s", body)
	}

	raw2 := httptest.NewRequest(http.MethodPost, "/demo/ai/chat", bytes.NewBufferString(`{}`))
	raw2.Header.Set("Content-Type", "application/json")
	resp2 := handler(zhttp.NewRequest(raw2))
	if resp2.StatusCode() != 422 {
		t.Fatalf("status=%d", resp2.StatusCode())
	}
}
