package rag_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/zatrano/framework/packages/rag"
	_ "modernc.org/sqlite"
)

func TestSQLStore(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := rag.NewSQLStore(db)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	err = s.Upsert(context.Background(), []rag.Chunk{
		{ID: "a", DocumentID: "d", Text: "cats", Embedding: []float64{1, 0, 0}, Metadata: map[string]string{"k": "v"}},
		{ID: "b", DocumentID: "d", Text: "dogs", Embedding: []float64{0, 1, 0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Len() != 2 {
		t.Fatalf("%d", s.Len())
	}
	hits, err := s.Search(context.Background(), []float64{0.9, 0.1, 0}, 1)
	if err != nil || len(hits) != 1 || hits[0].ID != "a" {
		t.Fatalf("%v %+v", err, hits)
	}
	if hits[0].Metadata["k"] != "v" {
		t.Fatalf("%+v", hits[0].Metadata)
	}
	// update
	err = s.Upsert(context.Background(), []rag.Chunk{
		{ID: "a", DocumentID: "d", Text: "cats!", Embedding: []float64{1, 0, 0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteDocument(context.Background(), "d"); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 0 {
		t.Fatalf("%d", s.Len())
	}
}
