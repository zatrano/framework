package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/zatrano/framework/packages/ai"
)

// PairScorer scores how well a document answers a query (cross-encoder style).
// Higher is better.
type PairScorer func(ctx context.Context, query, document string) (float64, error)

// CrossEncoderReranker scores each hit with PairScorer and reorders by that score.
type CrossEncoderReranker struct {
	Score PairScorer
	TopN  int // 0 = keep all
}

// Rerank implements Reranker.
func (r CrossEncoderReranker) Rerank(ctx context.Context, query string, hits []Hit) ([]Hit, error) {
	if r.Score == nil {
		return nil, fmt.Errorf("rag: CrossEncoderReranker requires Score")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := append([]Hit(nil), hits...)
	for i := range out {
		sc, err := r.Score(ctx, query, out[i].Text)
		if err != nil {
			return nil, err
		}
		out[i].Score = sc
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].ID < out[j].ID
		}
		return out[i].Score > out[j].Score
	})
	if r.TopN > 0 && len(out) > r.TopN {
		out = out[:r.TopN]
	}
	return out, nil
}

// ChatJSON is satisfied by *ai.Client (and adapters around *ai.Manager).
type ChatJSON interface {
	ChatJSON(ctx context.Context, req ai.ChatRequest, dest any) (*ai.ChatResponse, error)
}

// LLMReranker asks a chat model to return hit IDs in best-first order (JSON).
type LLMReranker struct {
	Chat ChatJSON
	TopN int // 0 = keep model order length (capped to input)
}

// FromAIChat adapts *ai.Manager to ChatJSON for LLMReranker.
func FromAIChat(m *ai.Manager) ChatJSON {
	return chatJSONFunc(func(ctx context.Context, req ai.ChatRequest, dest any) (*ai.ChatResponse, error) {
		return m.ChatJSON(ctx, req, dest)
	})
}

type chatJSONFunc func(ctx context.Context, req ai.ChatRequest, dest any) (*ai.ChatResponse, error)

func (f chatJSONFunc) ChatJSON(ctx context.Context, req ai.ChatRequest, dest any) (*ai.ChatResponse, error) {
	return f(ctx, req, dest)
}

type llmRankPayload struct {
	Order []string `json:"order"`
}

// Rerank implements Reranker.
func (r LLMReranker) Rerank(ctx context.Context, query string, hits []Hit) ([]Hit, error) {
	if r.Chat == nil {
		return nil, fmt.Errorf("rag: LLMReranker requires Chat")
	}
	if len(hits) == 0 {
		return hits, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("Rank these passages for the query. Return JSON {\"order\":[\"id\",…]} best-first.\n")
	b.WriteString("Query: ")
	b.WriteString(query)
	b.WriteString("\n\nPassages:\n")
	byID := make(map[string]Hit, len(hits))
	for _, h := range hits {
		byID[h.ID] = h
		fmt.Fprintf(&b, "- id=%s score=%.4f text=%s\n", h.ID, h.Score, truncateRunes(h.Text, 400))
	}
	var parsed llmRankPayload
	_, err := r.Chat.ChatJSON(ctx, ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: b.String()}},
	}, &parsed)
	if err != nil || len(parsed.Order) == 0 {
		// Fallback: keep retrieval order when the model output is unusable.
		out := append([]Hit(nil), hits...)
		if r.TopN > 0 && len(out) > r.TopN {
			out = out[:r.TopN]
		}
		return out, nil
	}
	seen := map[string]bool{}
	out := make([]Hit, 0, len(hits))
	n := len(parsed.Order)
	for i, id := range parsed.Order {
		h, ok := byID[id]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		h.Score = float64(n - i)
		out = append(out, h)
	}
	for _, h := range hits {
		if seen[h.ID] {
			continue
		}
		out = append(out, h)
	}
	if r.TopN > 0 && len(out) > r.TopN {
		out = out[:r.TopN]
	}
	return out, nil
}

func truncateRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if max <= 0 || len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "…"
}
