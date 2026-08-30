package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/support/uuid"
)

// FakeDriver returns deterministic stub replies for tests and local use without API keys.
type FakeDriver struct{}

func (FakeDriver) Name() string { return "fake" }

// Capabilities implements Capabler.
func (FakeDriver) Capabilities() []Capability {
	return []Capability{CapChat, CapEmbed, CapStream, CapTools, CapJSON}
}

// Health implements Healthy (always OK unless context canceled).
func (FakeDriver) Health(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func (FakeDriver) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	prompt := lastUser(req.Messages)
	reply := "ZATRANO AI stub: " + prompt
	if prompt == "" {
		reply = "ZATRANO AI stub: hello"
	}
	if req.ResponseFormat != nil && req.ResponseFormat.wantsJSON() {
		payload, err := json.Marshal(map[string]string{"text": reply})
		if err != nil {
			return nil, err
		}
		reply = string(payload)
	}

	msg := Message{Role: "assistant", Content: reply}
	finish := "stop"
	if len(req.Tools) > 0 && !toolChoiceNone(req.ToolChoice) {
		tool := req.Tools[0]
		name := tool.Function.Name
		if req.ToolChoice != nil && req.ToolChoice.Mode == "function" && req.ToolChoice.Name != "" {
			name = req.ToolChoice.Name
		}
		args, _ := json.Marshal(map[string]string{"query": prompt})
		msg = Message{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				ID:   "call_" + uuid.New()[:8],
				Type: "function",
				Function: ToolCallFunction{
					Name:      name,
					Arguments: string(args),
				},
			}},
		}
		finish = "tool_calls"
	}

	promptTokens := len(strings.Fields(prompt)) + 1
	completionTokens := len(strings.Fields(reply))
	if completionTokens < 1 {
		completionTokens = 1
	}
	return &ChatResponse{
		ID:           "chat_" + uuid.New()[:8],
		Model:        req.Model,
		Message:      msg,
		FinishReason: finish,
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
		Created: time.Now().UTC(),
	}, nil
}

func toolChoiceNone(c *ToolChoice) bool {
	return c != nil && strings.EqualFold(strings.TrimSpace(c.Mode), "none")
}

func (FakeDriver) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([][]float64, len(req.Input))
	for i, s := range req.Input {
		// Deterministic tiny vector from string length.
		n := float64(len(s) + 1)
		out[i] = []float64{n, n / 2, 1}
	}
	return &EmbedResponse{
		Model:      req.Model,
		Embeddings: out,
		Usage:      Usage{PromptTokens: len(req.Input), TotalTokens: len(req.Input)},
	}, nil
}

// ChatStream yields the stub reply in small deltas (implements StreamDriver).
func (FakeDriver) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resp, err := (FakeDriver{}).Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make(chan StreamChunk, 8)
	go func() {
		defer close(out)
		if len(resp.Message.ToolCalls) > 0 {
			for _, tc := range resp.Message.ToolCalls {
				out <- StreamChunk{
					ToolCallDeltas: []StreamToolCallDelta{{
						Index:     0,
						ID:        tc.ID,
						Type:      tc.Type,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					}},
					ID:    resp.ID,
					Model: resp.Model,
				}
			}
			u := resp.Usage
			out <- StreamChunk{
				Done:         true,
				FinishReason: resp.FinishReason,
				ToolCalls:    resp.Message.ToolCalls,
				Usage:        &u,
				ID:           resp.ID,
				Model:        resp.Model,
			}
			return
		}
		content := resp.Message.Content
		parts := strings.Fields(content)
		if len(parts) == 0 {
			parts = []string{content}
		}
		for i, p := range parts {
			if err := ctx.Err(); err != nil {
				out <- StreamChunk{Done: true, Err: err, ID: resp.ID, Model: resp.Model}
				return
			}
			piece := p
			if i > 0 {
				piece = " " + p
			}
			out <- StreamChunk{Delta: piece, ID: resp.ID, Model: resp.Model}
		}
		u := resp.Usage
		out <- StreamChunk{Done: true, FinishReason: resp.FinishReason, Usage: &u, ID: resp.ID, Model: resp.Model}
	}()
	return out, nil
}

func lastUser(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	if len(messages) > 0 {
		return messages[len(messages)-1].Content
	}
	return ""
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
