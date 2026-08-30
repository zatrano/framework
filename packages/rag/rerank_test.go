package rag_test

import (
	"context"
	"testing"

	"github.com/zatrano/framework/packages/rag"
)

func TestKeywordRerankAndQueryWith(t *testing.T) {
	store := rag.NewMemoryStore()
	_ = store.Upsert(context.Background(), []rag.Chunk{
		{ID: "1", DocumentID: "d", Text: "cooking pasta with salt", Embedding: []float64{1, 0}},
		{ID: "2", DocumentID: "d", Text: "ZATRANO AI profiles and routing", Embedding: []float64{0.99, 0.01}},
	})
	embed := rag.FuncEmbedder(func(ctx context.Context, texts []string) ([][]float64, error) {
		out := make([][]float64, len(texts))
		for i := range texts {
			out[i] = []float64{1, 0}
		}
		return out, nil
	})
	p := &rag.Pipeline{Embed: embed, Store: store, Chunker: rag.TextChunker{Size: 200, MinChars: 1}}
	hits, err := p.QueryWith(context.Background(), "ZATRANO profiles", rag.QueryOptions{
		TopK:   2,
		Rerank: rag.KeywordReranker{Boost: 1},
	})
	if err != nil || len(hits) == 0 {
		t.Fatalf("%v %+v", err, hits)
	}
	if hits[0].ID != "2" {
		t.Fatalf("expected keyword boost to prefer profiles chunk, got %+v", hits)
	}
}

func TestVectorLiteral(t *testing.T) {
	if got := rag.VectorLiteral([]float64{1, 2.5}); got != "[1,2.5]" {
		t.Fatal(got)
	}
	if got := rag.VectorLiteral(nil); got != "[]" {
		t.Fatal(got)
	}
}

func TestPGVectorStoreDimsGuard(t *testing.T) {
	s := rag.NewPGVectorStore(nil, 0)
	if err := s.Migrate(context.Background()); err == nil {
		t.Fatal("expected dims error")
	}
}
