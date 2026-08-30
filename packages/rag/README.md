# rag

Retrieval-augmented generation helpers: chunk → embed → store → query.

Import-only library (no `package:enable`). Pair with `packages/ai` for embeddings and chat.

```go
import (
    "context"

    "github.com/zatrano/framework/packages/ai"
    "github.com/zatrano/framework/packages/rag"
)

mgr := ai.New() // or ai.From(app)
p := &rag.Pipeline{
    Chunker: rag.TextChunker{Size: 800, Overlap: 100},
    Embed:   rag.FromAI(mgr, "text-embedding-3-small"),
    Store:   rag.NewJSONFileStore("storage/rag/chunks.json"), // or NewMemoryStore()
}

_ = p.Index(ctx, rag.Document{ID: "docs", Text: longMarkdown})
hits, _ := p.Query(ctx, "How do profiles work?", 5)
contextBlock := rag.FormatContext(hits, 6000)

resp, _ := mgr.Profile("support").Chat(ctx, ai.ChatRequest{
    Messages: []ai.Message{{
        Role: "user",
        Content: "Use only this context:\n\n" + contextBlock + "\n\nQuestion: How do profiles work?",
    }},
})
_ = resp
```

## Pieces

| Type | Role |
|------|------|
| `TextChunker` | Rune-sized overlapping splits |
| `Embedder` / `FromAI` | Vectors via `ai.Manager.Embed` |
| `MemoryStore` | In-process cosine top-K |
| `JSONFileStore` | Durable JSON file store |
| `SQLStore` | Durable `database/sql` store (JSON embeddings) |
| `PGVectorStore` | Postgres + pgvector ANN |
| `Reranker` / `KeywordReranker` / `QueryWith` | Post-retrieval rerank |
| `Pipeline` | `Index` + `Query` |
| `FormatContext` | Prompt-ready hit dump |

Implement `VectorStore` for Postgres/pgvector, Redis, etc. without changing the pipeline.
