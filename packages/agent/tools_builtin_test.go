package agent_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/agent"
	"github.com/zatrano/framework/packages/ai"
)

func TestRegisterWebFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world page"))
	}))
	defer srv.Close()

	reg := agent.NewRegistry()
	err := agent.RegisterWebFetch(reg, agent.WebFetchOptions{
		AllowHTTP:  true,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	tools := reg.Tools()
	if len(tools) != 1 || tools[0].Function.Name != "web_fetch" {
		t.Fatalf("%+v", tools)
	}
	out, err := reg.Execute(context.Background(), ai.ToolCall{
		ID: "1", Type: "function",
		Function: ai.ToolCallFunction{Name: "web_fetch", Arguments: `{"url":"` + srv.URL + `"}`},
	})
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(parsed["body"].(string), "hello") {
		t.Fatalf("%v", parsed)
	}
}

func TestRegisterFileSearch(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "guide.md"), []byte("ZATRANO profiles route AI calls."), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "other.txt"), []byte("unrelated"), 0o600)

	reg := agent.NewRegistry()
	err := agent.RegisterFileSearch(reg, agent.FileSearchOptions{
		Root:       dir,
		Extensions: []string{".md", ".txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := reg.Execute(context.Background(), ai.ToolCall{
		Function: ai.ToolCallFunction{Name: "file_search", Arguments: `{"query":"profiles"}`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "guide.md") {
		t.Fatalf("%s", out)
	}
}
