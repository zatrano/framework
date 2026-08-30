package agent

import (
	"context"

	"github.com/zatrano/framework/packages/ai"
)

// Chatter is satisfied by *ai.Client (Using / Profile).
// Use FromManager to adapt *ai.Manager (whose Chat has a variadic provider arg).
type Chatter interface {
	Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error)
}

// FromManager adapts *ai.Manager to Chatter (default / profile-less Chat).
func FromManager(m *ai.Manager) Chatter {
	return chatterFunc(func(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
		return m.Chat(ctx, req)
	})
}

type chatterFunc func(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error)

func (f chatterFunc) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	return f(ctx, req)
}

// Memory stores conversation turns for multi-step agent runs.
type Memory interface {
	Messages() []ai.Message
	Append(msgs ...ai.Message)
	Clear()
}

// Retriever optionally injects RAG context before the user turn.
type Retriever interface {
	Retrieve(ctx context.Context, query string) (string, error)
}

// Handler executes a single tool call and returns a string result for the model.
type Handler func(ctx context.Context, call ai.ToolCall) (string, error)

// Result is the outcome of Agent.Run.
type Result struct {
	Response *ai.ChatResponse
	Steps    int
	Messages []ai.Message // full transcript including tools
}
