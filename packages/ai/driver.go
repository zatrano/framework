package ai

import (
	"context"
	"sort"
)

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

// StreamToolCallDelta is an incremental tool-call fragment from a stream frame.
type StreamToolCallDelta struct {
	Index     int    // provider tool_calls[].index
	ID        string // set on first fragment
	Type      string // "function"
	Name      string // function name (usually complete on first fragment)
	Arguments string // JSON argument fragment to append
}

// StreamChunk is one piece of a streamed chat response.
// Mid-stream failures set Err; Done marks the terminal chunk (success or after Err).
// ToolCallDeltas carry incremental tool call pieces; ToolCalls on Done holds the assembled calls.
type StreamChunk struct {
	Delta          string
	ToolCallDeltas []StreamToolCallDelta
	ToolCalls      []ToolCall // assembled on Done when finish_reason is tool_calls
	FinishReason   string
	Done           bool
	Usage          *Usage
	ID             string
	Model          string
	Err            error
}

// HasToolCallDeltas reports whether this chunk includes tool-call fragments.
func (c StreamChunk) HasToolCallDeltas() bool {
	return len(c.ToolCallDeltas) > 0
}

type toolCallAssembler struct {
	byIndex map[int]*ToolCall
}

func (a *toolCallAssembler) apply(deltas []StreamToolCallDelta) {
	if a.byIndex == nil {
		a.byIndex = make(map[int]*ToolCall)
	}
	for _, d := range deltas {
		tc, ok := a.byIndex[d.Index]
		if !ok {
			tc = &ToolCall{Type: "function"}
			a.byIndex[d.Index] = tc
		}
		if d.ID != "" {
			tc.ID = d.ID
		}
		if d.Type != "" {
			tc.Type = d.Type
		}
		if d.Name != "" {
			tc.Function.Name = d.Name
		}
		if d.Arguments != "" {
			tc.Function.Arguments += d.Arguments
		}
	}
}

func (a *toolCallAssembler) result() []ToolCall {
	if a == nil || len(a.byIndex) == 0 {
		return nil
	}
	idxs := make([]int, 0, len(a.byIndex))
	for i := range a.byIndex {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	out := make([]ToolCall, 0, len(idxs))
	for _, i := range idxs {
		out = append(out, *a.byIndex[i])
	}
	return out
}

// CollectStream drains a ChatStream channel into text, assembled tool calls, and usage.
func CollectStream(ch <-chan StreamChunk) (text string, calls []ToolCall, usage *Usage, finish string, err error) {
	var asm toolCallAssembler
	for chunk := range ch {
		if chunk.Err != nil {
			err = chunk.Err
		}
		text += chunk.Delta
		if len(chunk.ToolCallDeltas) > 0 {
			asm.apply(chunk.ToolCallDeltas)
		}
		if len(chunk.ToolCalls) > 0 {
			calls = chunk.ToolCalls
		}
		if chunk.Usage != nil {
			u := *chunk.Usage
			usage = &u
		}
		if chunk.FinishReason != "" {
			finish = chunk.FinishReason
		}
	}
	if len(calls) == 0 {
		calls = asm.result()
	}
	return text, calls, usage, finish, err
}
