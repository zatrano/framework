package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// MemoryStore is an in-process vector store (tests, small corpora).
type MemoryStore struct {
	mu     sync.RWMutex
	chunks map[string]Chunk // chunk ID → chunk
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{chunks: make(map[string]Chunk)}
}

// Upsert implements VectorStore.
func (s *MemoryStore) Upsert(ctx context.Context, chunks []Chunk) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("rag: store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.chunks == nil {
		s.chunks = make(map[string]Chunk)
	}
	for _, c := range chunks {
		id := strings.TrimSpace(c.ID)
		if id == "" {
			return fmt.Errorf("rag: chunk id is required")
		}
		if len(c.Embedding) == 0 {
			return fmt.Errorf("rag: chunk %s missing embedding", id)
		}
		cp := c
		cp.Metadata = cloneMeta(c.Metadata)
		cp.Embedding = append([]float64(nil), c.Embedding...)
		s.chunks[id] = cp
	}
	return nil
}

// Search implements VectorStore (brute-force cosine top-K).
func (s *MemoryStore) Search(ctx context.Context, query []float64, topK int) ([]Hit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("rag: store is nil")
	}
	if topK <= 0 {
		topK = 5
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	hits := make([]Hit, 0, len(s.chunks))
	for _, c := range s.chunks {
		score := CosineSimilarity(query, c.Embedding)
		hits = append(hits, Hit{Chunk: c, Score: score})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].ID < hits[j].ID
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, nil
}

// DeleteDocument implements VectorStore.
func (s *MemoryStore) DeleteDocument(ctx context.Context, documentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return fmt.Errorf("rag: document id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, c := range s.chunks {
		if c.DocumentID == documentID {
			delete(s.chunks, id)
		}
	}
	return nil
}

// Len implements VectorStore.
func (s *MemoryStore) Len() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.chunks)
}
