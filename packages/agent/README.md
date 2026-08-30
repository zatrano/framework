# agent

Multi-step AI agents: chat ↔ tool loop, conversation memory, optional RAG retrieval.

Import-only library (no `package:enable`). Uses `packages/ai` for chat/tools and optionally `packages/rag`.

```go
import (
    "context"
    "encoding/json"

    "github.com/zatrano/framework/packages/agent"
    "github.com/zatrano/framework/packages/ai"
)

mgr := ai.New()
reg := agent.NewRegistry()
_ = reg.Register(ai.FunctionTool("lookup", "Search docs", json.RawMessage(`{
    "type":"object","properties":{"query":{"type":"string"}},"required":["query"]
}`)), func(ctx context.Context, call ai.ToolCall) (string, error) {
    var args struct{ Query string `json:"query"` }
    _ = call.UnmarshalArguments(&args)
    return `{"hits":[]}`, nil
})

a := &agent.Agent{
    Chat:   mgr.Profile("support"), // *ai.Client implements Chatter; or agent.FromManager(mgr)
    Tools:  reg,
    Memory: &agent.BufferMemory{},
    System: "You help with ZATRANO docs.",
}
res, err := a.Run(ctx, "How do profiles work?")
_ = res.Response.Message.Content
```

## Built-in tools

```go
_ = agent.RegisterWebFetch(reg, agent.WebFetchOptions{})
_ = agent.RegisterFileSearch(reg, agent.FileSearchOptions{Root: "./docs", Extensions: []string{".md"}})
```

## Queue

```go
cat := agent.NewCatalog()
_ = cat.Register("support", a)
runner := &agent.Runner{Catalog: cat}
_ = runner.RegisterQueue(queue.From(app))
_ = runner.PushRun(queue.From(app), agent.RunJob{Agent: "support", Message: "hi", ID: "1"})
```

## RAG

```go
a.Retrieve = agent.RAGRetrieve{Pipeline: ragPipeline, TopK: 5}
```

## Pieces

| Type | Role |
|------|------|
| `Agent` | `Run` loop (`MaxSteps`, last-step `ToolChoiceNone` by default) |
| `Registry` | Tool defs + handlers |
| `BufferMemory` | Conversation buffer (`MaxKeep`) |
| `Retriever` / `RAGRetrieve` | Optional context injection |
| `RegisterWebFetch` / `RegisterFileSearch` | HTTPS fetch + sandboxed file search |
| `Catalog` / `Runner` / `PushRun` | Queue-backed `agent.run` jobs |
| `Result` | Final response + step count + transcript |
