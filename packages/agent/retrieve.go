package agent

import (
	"context"
	"strings"

	"github.com/zatrano/framework/packages/rag"
)

// RAGRetrieve adapts a rag.Pipeline to Retriever (topK hits → FormatContext).
type RAGRetrieve struct {
	Pipeline *rag.Pipeline
	TopK     int
	MaxChars int
}

// Retrieve implements Retriever.
func (r RAGRetrieve) Retrieve(ctx context.Context, query string) (string, error) {
	if r.Pipeline == nil {
		return "", nil
	}
	topK := r.TopK
	if topK <= 0 {
		topK = 5
	}
	hits, err := r.Pipeline.Query(ctx, query, topK)
	if err != nil {
		return "", err
	}
	return rag.FormatContext(hits, r.MaxChars), nil
}

// FuncRetriever adapts a function to Retriever.
type FuncRetriever func(ctx context.Context, query string) (string, error)

// Retrieve implements Retriever.
func (f FuncRetriever) Retrieve(ctx context.Context, query string) (string, error) {
	if f == nil {
		return "", nil
	}
	s, err := f(ctx, query)
	return strings.TrimSpace(s), err
}
