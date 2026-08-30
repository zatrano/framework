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

// StreamDriver optionally streams chat completions (SSE / chunked).
type StreamDriver interface {
	ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}

// StreamChunk is one piece of a streamed chat response.
// Mid-stream failures set Err; Done marks the terminal chunk (success or after Err).
type StreamChunk struct {
	Delta string
	Done  bool
	Usage *Usage
	ID    string
	Model string
	Err   error
}
