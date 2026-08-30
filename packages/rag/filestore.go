package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// JSONFileStore is a durable VectorStore backed by a single JSON file.
// Suitable for small/medium corpora; brute-force cosine search like MemoryStore.
type JSONFileStore struct {
	Path string

	mu     sync.RWMutex
	chunks map[string]Chunk
	loaded bool
}

// NewJSONFileStore returns a store that persists to path (created on first Upsert/Save).
func NewJSONFileStore(path string) *JSONFileStore {
	return &JSONFileStore{Path: path, chunks: make(map[string]Chunk)}
}

func (s *JSONFileStore) ensureLoadedLocked() error {
	if s.loaded {
		return nil
	}
	if s.chunks == nil {
		s.chunks = make(map[string]Chunk)
	}
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			s.loaded = true
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		s.loaded = true
		return nil
	}
	var list []Chunk
	if err := json.Unmarshal(raw, &list); err != nil {
		return fmt.Errorf("rag: load %s: %w", s.Path, err)
	}
	for _, c := range list {
		id := strings.TrimSpace(c.ID)
		if id == "" {
			continue
		}
		s.chunks[id] = c
	}
	s.loaded = true
	return nil
}

func (s *JSONFileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ensureLoadedLocked()
}

func (s *JSONFileStore) persistLocked() error {
	if strings.TrimSpace(s.Path) == "" {
		return fmt.Errorf("rag: JSONFileStore path is required")
	}
	dir := filepath.Dir(s.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	list := make([]Chunk, 0, len(s.chunks))
	for _, c := range s.chunks {
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}

// Upsert implements VectorStore and writes the file.
func (s *JSONFileStore) Upsert(ctx context.Context, chunks []Chunk) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("rag: store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return err
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
	return s.persistLocked()
}

// Search implements VectorStore.
func (s *JSONFileStore) Search(ctx context.Context, query []float64, topK int) ([]Hit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("rag: store is nil")
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	if topK <= 0 {
		topK = 5
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	hits := make([]Hit, 0, len(s.chunks))
	for _, c := range s.chunks {
		hits = append(hits, Hit{Chunk: c, Score: CosineSimilarity(query, c.Embedding)})
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

// DeleteDocument implements VectorStore and writes the file.
func (s *JSONFileStore) DeleteDocument(ctx context.Context, documentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return fmt.Errorf("rag: document id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return err
	}
	for id, c := range s.chunks {
		if c.DocumentID == documentID {
			delete(s.chunks, id)
		}
	}
	return s.persistLocked()
}

// Len implements VectorStore.
func (s *JSONFileStore) Len() int {
	if s == nil {
		return 0
	}
	_ = s.load()
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.chunks)
}
