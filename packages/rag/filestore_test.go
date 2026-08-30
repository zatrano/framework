package rag_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/rag"
)

func TestJSONFileStorePersist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chunks.json")
	s := rag.NewJSONFileStore(path)
	err := s.Upsert(context.Background(), []rag.Chunk{
		{ID: "a", DocumentID: "d", Text: "alpha", Embedding: []float64{1, 0}},
		{ID: "b", DocumentID: "d", Text: "beta", Embedding: []float64{0, 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s2 := rag.NewJSONFileStore(path)
	hits, err := s2.Search(context.Background(), []float64{1, 0}, 1)
	if err != nil || len(hits) != 1 || hits[0].ID != "a" {
		t.Fatalf("%v %+v", err, hits)
	}
	if s2.Len() != 2 {
		t.Fatalf("%d", s2.Len())
	}
	if err := s2.DeleteDocument(context.Background(), "d"); err != nil {
		t.Fatal(err)
	}
	if s2.Len() != 0 {
		t.Fatalf("%d", s2.Len())
	}
}
