package rag_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/zatrano/framework/packages/ai"
	"github.com/zatrano/framework/packages/rag"
)

func TestCrossEncoderReranker(t *testing.T) {
	hits := []rag.Hit{
		{Chunk: rag.Chunk{ID: "a", Text: "unrelated"}, Score: 0.9},
		{Chunk: rag.Chunk{ID: "b", Text: "zatrano profiles"}, Score: 0.1},
	}
	r := rag.CrossEncoderReranker{
		Score: func(ctx context.Context, query, doc string) (float64, error) {
			if doc == "zatrano profiles" {
				return 10, nil
			}
			return 1, nil
		},
		TopN: 1,
	}
	out, err := r.Rerank(context.Background(), "profiles", hits)
	if err != nil || len(out) != 1 || out[0].ID != "b" {
		t.Fatalf("%v %+v", err, out)
	}
}

type stubRankChat struct{}

func (stubRankChat) ChatJSON(ctx context.Context, req ai.ChatRequest, dest any) (*ai.ChatResponse, error) {
	raw := []byte(`{"order":["b","a"]}`)
	if err := json.Unmarshal(raw, dest); err != nil {
		return nil, err
	}
	return &ai.ChatResponse{Message: ai.Message{Role: "assistant", Content: string(raw)}}, nil
}

func TestLLMReranker(t *testing.T) {
	hits := []rag.Hit{
		{Chunk: rag.Chunk{ID: "a", Text: "aaa"}, Score: 0.9},
		{Chunk: rag.Chunk{ID: "b", Text: "bbb"}, Score: 0.1},
	}
	r := rag.LLMReranker{Chat: stubRankChat{}, TopN: 2}
	out, err := r.Rerank(context.Background(), "q", hits)
	if err != nil || len(out) != 2 || out[0].ID != "b" || out[1].ID != "a" {
		t.Fatalf("%v %+v", err, out)
	}
}

func TestLLMRerankerFallback(t *testing.T) {
	bad := rag.LLMReranker{Chat: badRankChat{}, TopN: 1}
	hits := []rag.Hit{{Chunk: rag.Chunk{ID: "a"}, Score: 1}}
	out, err := bad.Rerank(context.Background(), "q", hits)
	if err != nil || len(out) != 1 || out[0].ID != "a" {
		t.Fatalf("%v %+v", err, out)
	}
}

type badRankChat struct{}

func (badRankChat) ChatJSON(ctx context.Context, req ai.ChatRequest, dest any) (*ai.ChatResponse, error) {
	return nil, context.Canceled
}
