package rag_test

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/ai"
	"github.com/zatrano/framework/packages/rag"
)

func TestTextChunker(t *testing.T) {
	c := rag.TextChunker{Size: 20, Overlap: 5, MinChars: 5}
	chunks := c.Split(rag.Document{ID: "d1", Text: strings.Repeat("abcdefghij", 5)})
	if len(chunks) < 2 {
		t.Fatalf("%d", len(chunks))
	}
	if chunks[0].DocumentID != "d1" || chunks[0].ID != "d1#0" {
		t.Fatalf("%+v", chunks[0])
	}
}

func TestMemoryStoreRoundTrip(t *testing.T) {
	store := rag.NewMemoryStore()
	err := store.Upsert(context.Background(), []rag.Chunk{
		{ID: "a", DocumentID: "d", Text: "cats", Embedding: []float64{1, 0, 0}},
		{ID: "b", DocumentID: "d", Text: "dogs", Embedding: []float64{0, 1, 0}},
		{ID: "c", DocumentID: "e", Text: "birds", Embedding: []float64{0, 0, 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	hits, err := store.Search(context.Background(), []float64{0.9, 0.1, 0}, 2)
	if err != nil || len(hits) != 2 || hits[0].ID != "a" {
		t.Fatalf("%v %+v", err, hits)
	}
	if err := store.DeleteDocument(context.Background(), "d"); err != nil {
		t.Fatal(err)
	}
	if store.Len() != 1 {
		t.Fatalf("%d", store.Len())
	}
}

func TestCosineSimilarity(t *testing.T) {
	if s := rag.CosineSimilarity([]float64{1, 0}, []float64{1, 0}); math.Abs(s-1) > 1e-9 {
		t.Fatal(s)
	}
	if s := rag.CosineSimilarity([]float64{1, 0}, []float64{0, 1}); math.Abs(s) > 1e-9 {
		t.Fatal(s)
	}
}

func TestPipelineWithFakeAI(t *testing.T) {
	mgr := ai.New()
	p := &rag.Pipeline{
		Chunker: rag.TextChunker{Size: 100, Overlap: 0, MinChars: 1},
		Embed:   rag.FromAI(mgr, ""),
		Store:   rag.NewMemoryStore(),
	}
	err := p.Index(context.Background(),
		rag.Document{ID: "guide", Text: "ZATRANO is a Go web framework for artisans."},
		rag.Document{ID: "other", Text: "Cooking pasta requires boiling water."},
	)
	if err != nil {
		t.Fatal(err)
	}
	hits, err := p.Query(context.Background(), "What is ZATRANO?", 2)
	if err != nil || len(hits) == 0 {
		t.Fatalf("%v %+v", err, hits)
	}
	ctx := rag.FormatContext(hits, 2000)
	if !strings.Contains(ctx, "score=") {
		t.Fatalf("%q", ctx)
	}
}

func TestHashEmbedderDeterministic(t *testing.T) {
	// Fake AI embeddings are deterministic enough for smoke; prefer pipeline test above.
	embed := rag.FuncEmbedder(func(ctx context.Context, texts []string) ([][]float64, error) {
		out := make([][]float64, len(texts))
		for i, s := range texts {
			v := make([]float64, 8)
			for _, r := range s {
				v[int(r)%8] += 1
			}
			out[i] = v
		}
		return out, nil
	})
	p := &rag.Pipeline{Embed: embed, Store: rag.NewMemoryStore(), Chunker: rag.TextChunker{Size: 50, MinChars: 1}}
	if err := p.Index(context.Background(), rag.Document{ID: "1", Text: "alpha beta gamma"}); err != nil {
		t.Fatal(err)
	}
	hits, err := p.Query(context.Background(), "alpha", 1)
	if err != nil || len(hits) != 1 {
		t.Fatalf("%v %+v", err, hits)
	}
}
