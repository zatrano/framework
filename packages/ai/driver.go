package ai

import "context"

// Driver generates chat completions (provider-independent).
type Driver interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

// EmbeddingDriver optionally generates vector embeddings.
type EmbeddingDriver interface {
	Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error)
}
