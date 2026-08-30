package rag

import (
	"context"
	"fmt"
	"strings"
)

// Pipeline indexes documents and retrieves relevant chunks.
type Pipeline struct {
	Chunker Chunker
	Embed   Embedder
	Store   VectorStore
}

// Index chunk→embed→upsert for the given documents.
func (p *Pipeline) Index(ctx context.Context, docs ...Document) error {
	if p == nil || p.Embed == nil || p.Store == nil {
		return fmt.Errorf("rag: pipeline requires Embed and Store")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	chunker := p.Chunker
	if chunker == nil {
		chunker = TextChunker{}
	}
	var all []Chunk
	for _, doc := range docs {
		parts := chunker.Split(doc)
		all = append(all, parts...)
	}
	if len(all) == 0 {
		return nil
	}
	texts := make([]string, len(all))
	for i, c := range all {
		texts[i] = c.Text
	}
	vecs, err := p.Embed.Embed(ctx, texts)
	if err != nil {
		return err
	}
	if len(vecs) != len(all) {
		return fmt.Errorf("rag: embedder returned %d vectors for %d chunks", len(vecs), len(all))
	}
	for i := range all {
		all[i].Embedding = vecs[i]
	}
	return p.Store.Upsert(ctx, all)
}

// Query embeds the question and returns top-K hits.
func (p *Pipeline) Query(ctx context.Context, question string, topK int) ([]Hit, error) {
	if p == nil || p.Embed == nil || p.Store == nil {
		return nil, fmt.Errorf("rag: pipeline requires Embed and Store")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, fmt.Errorf("rag: question is required")
	}
	vecs, err := p.Embed.Embed(ctx, []string{question})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil, fmt.Errorf("rag: empty query embedding")
	}
	return p.Store.Search(ctx, vecs[0], topK)
}

// FormatContext joins hit texts for injection into a chat prompt.
func FormatContext(hits []Hit, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 6000
	}
	var b strings.Builder
	for i, h := range hits {
		block := strings.TrimSpace(h.Text)
		if block == "" {
			continue
		}
		header := fmt.Sprintf("[%d] (score=%.3f doc=%s)\n", i+1, h.Score, h.DocumentID)
		if b.Len()+len(header)+len(block)+2 > maxChars {
			break
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(header)
		b.WriteString(block)
	}
	return b.String()
}
