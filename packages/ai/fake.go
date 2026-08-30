package ai

import (
	"context"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/support/uuid"
)

// FakeDriver returns deterministic stub replies for tests and local use without API keys.
type FakeDriver struct{}

func (FakeDriver) Name() string { return "fake" }

func (FakeDriver) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	prompt := lastUser(req.Messages)
	reply := "ZATRANO AI stub: " + prompt
	if prompt == "" {
		reply = "ZATRANO AI stub: hello"
	}
	promptTokens := len(strings.Fields(prompt)) + 1
	completionTokens := len(strings.Fields(reply))
	return &ChatResponse{
		ID:      "chat_" + uuid.New()[:8],
		Model:   req.Model,
		Message: Message{Role: "assistant", Content: reply},
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
		Created: time.Now().UTC(),
	}, nil
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
		out <- StreamChunk{Done: true, Usage: &u, ID: resp.ID, Model: resp.Model}
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
