package agent_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/agent"
	"github.com/zatrano/framework/packages/ai"
	"github.com/zatrano/framework/packages/rag"
)

func TestAgentToolLoop(t *testing.T) {
	mgr := ai.New()
	reg := agent.NewRegistry()
	params := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)
	err := reg.Register(ai.FunctionTool("lookup", "Lookup", params), func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args struct {
			Query string `json:"query"`
		}
		_ = call.UnmarshalArguments(&args)
		return `{"ok":true,"q":"` + args.Query + `"}`, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	mem := &agent.BufferMemory{}
	a := &agent.Agent{
		Chat:     agent.FromManager(mgr),
		Tools:    reg,
		Memory:   mem,
		System:   "You are a helpful assistant.",
		MaxSteps: 3,
	}
	res, err := a.Run(context.Background(), "find widgets")
	if err != nil {
		t.Fatal(err)
	}
	if res.Steps < 2 {
		t.Fatalf("steps=%d", res.Steps)
	}
	if res.Response.HasToolCalls() {
		t.Fatal("expected final text")
	}
	if !strings.Contains(res.Response.Message.Content, "stub") {
		t.Fatalf("%q", res.Response.Message.Content)
	}
	// transcript includes tool result
	foundTool := false
	for _, m := range res.Messages {
		if m.Role == "tool" {
			foundTool = true
		}
	}
	if !foundTool {
		t.Fatalf("%+v", res.Messages)
	}
}

func TestAgentWithRetriever(t *testing.T) {
	mgr := ai.New()
	a := &agent.Agent{
		Chat: agent.FromManager(mgr),
		Retrieve: agent.FuncRetriever(func(ctx context.Context, q string) (string, error) {
			return "[1] ZATRANO uses profiles for AI routing.", nil
		}),
		MaxSteps: 1,
	}
	res, err := a.Run(context.Background(), "profiles?")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Messages[0].Content, "Context:") {
		t.Fatalf("%+v", res.Messages[0])
	}
}

func TestRAGRetrieve(t *testing.T) {
	mgr := ai.New()
	p := &rag.Pipeline{
		Chunker: rag.TextChunker{Size: 200, MinChars: 1},
		Embed:   rag.FromAI(mgr, ""),
		Store:   rag.NewMemoryStore(),
	}
	_ = p.Index(context.Background(), rag.Document{ID: "d", Text: "Agents call tools in a loop."})
	r := agent.RAGRetrieve{Pipeline: p, TopK: 1}
	s, err := r.Retrieve(context.Background(), "tools")
	if err != nil || !strings.Contains(s, "score=") {
		t.Fatalf("%v %q", err, s)
	}
}

func TestBufferMemoryTrim(t *testing.T) {
	m := &agent.BufferMemory{MaxKeep: 3}
	m.Append(ai.Message{Role: "system", Content: "sys"})
	m.Append(ai.Message{Role: "user", Content: "1"})
	m.Append(ai.Message{Role: "assistant", Content: "2"})
	m.Append(ai.Message{Role: "user", Content: "3"})
	m.Append(ai.Message{Role: "assistant", Content: "4"})
	msgs := m.Messages()
	if len(msgs) != 3 || msgs[0].Role != "system" {
		t.Fatalf("%+v", msgs)
	}
}
