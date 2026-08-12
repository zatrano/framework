package search

import (
	"sort"
	"strings"
	"sync"
)

// Document is an indexable record.
type Document struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Title   string         `json:"title"`
	Body    string         `json:"body"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Hit is a search result with a simple score.
type Hit struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
}

// Engine is a searchable document index.
type Engine interface {
	Index(doc Document) error
	Delete(id string) error
	Search(query string, limit int) ([]Hit, error)
	Flush() error
	Count() int
}

// MemoryEngine is an in-memory full-text-ish search engine.
type MemoryEngine struct {
	mu   sync.RWMutex
	docs map[string]Document
}

// NewMemoryEngine creates an empty in-memory engine.
func NewMemoryEngine() *MemoryEngine {
	return &MemoryEngine{docs: make(map[string]Document)}
}

// Index upserts a document.
func (e *MemoryEngine) Index(doc Document) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if doc.ID == "" {
		doc.ID = doc.Type + ":" + doc.Title
	}
	e.docs[doc.ID] = doc
	return nil
}

// Delete removes a document by id.
func (e *MemoryEngine) Delete(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.docs, id)
	return nil
}

// Search ranks documents by token overlap.
func (e *MemoryEngine) Search(query string, limit int) ([]Hit, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	tokens := tokenize(query)
	if len(tokens) == 0 {
		return []Hit{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	hits := make([]Hit, 0)
	for _, doc := range e.docs {
		hay := strings.ToLower(doc.Title + " " + doc.Body + " " + doc.Type)
		score := 0.0
		for _, tok := range tokens {
			if strings.Contains(hay, tok) {
				score += 1
				if strings.Contains(strings.ToLower(doc.Title), tok) {
					score += 2
				}
			}
		}
		if score > 0 {
			hits = append(hits, Hit{Document: doc, Score: score})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Document.Title < hits[j].Document.Title
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

// Flush clears the index.
func (e *MemoryEngine) Flush() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.docs = make(map[string]Document)
	return nil
}

// Count returns indexed document count.
func (e *MemoryEngine) Count() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.docs)
}

// Manager wraps an Engine with helpers.
type Manager struct {
	engine Engine
}

// New creates a search manager.
func New(engine Engine) *Manager {
	return &Manager{engine: engine}
}

// Engine returns the underlying engine.
func (m *Manager) Engine() Engine { return m.engine }

// Index proxies to the engine.
func (m *Manager) Index(doc Document) error { return m.engine.Index(doc) }

// Search proxies to the engine.
func (m *Manager) Search(query string, limit int) ([]Hit, error) {
	return m.engine.Search(query, limit)
}

// Delete proxies to the engine.
func (m *Manager) Delete(id string) error { return m.engine.Delete(id) }

// Flush proxies to the engine.
func (m *Manager) Flush() error { return m.engine.Flush() }

// Count proxies to the engine.
func (m *Manager) Count() int { return m.engine.Count() }

func tokenize(query string) []string {
	parts := strings.Fields(strings.ToLower(query))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, ".,!?;:\"'()[]{}")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
