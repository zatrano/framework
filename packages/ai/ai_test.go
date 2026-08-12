package ai_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/ai"
)

func TestAIChat(t *testing.T) {
	m := ai.New()
	resp, err := m.Chat(ai.ChatRequest{
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
	resp, err := d.Chat(ai.ChatRequest{
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
			resp, err := d.Chat(ai.ChatRequest{
				Messages: []ai.Message{{Role: "user", Content: "ping"}},
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
}
