package rag

import "context"

// Document is a source unit to index (article, page, file text, …).
type Document struct {
	ID       string
	Text     string
	Metadata map[string]string
}

// Chunk is an embedded fragment of a document.
type Chunk struct {
	ID         string
	DocumentID string
	Text       string
	Index      int // 0-based position within the document
	Embedding  []float64
	Metadata   map[string]string
}

// Hit is a ranked retrieval result.
type Hit struct {
	Chunk
	Score float64 // cosine similarity in [−1, 1]; higher is better
}

// Chunker splits document text into overlapping pieces.
type Chunker interface {
	Split(doc Document) []Chunk
}

// Embedder turns texts into vectors (typically packages/ai Embed).
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

// VectorStore persists and searches embedded chunks.
type VectorStore interface {
	Upsert(ctx context.Context, chunks []Chunk) error
	Search(ctx context.Context, query []float64, topK int) ([]Hit, error)
	DeleteDocument(ctx context.Context, documentID string) error
	Len() int
}

// FuncEmbedder adapts a function to Embedder.
type FuncEmbedder func(ctx context.Context, texts []string) ([][]float64, error)

// Embed implements Embedder.
func (f FuncEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	return f(ctx, texts)
}
