package rag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SQLStore is a durable VectorStore using database/sql.
// Embeddings are stored as JSON arrays; search is brute-force cosine in Go
// (portable across SQLite/MySQL/Postgres without pgvector).
type SQLStore struct {
	DB    *sql.DB
	Table string // default rag_chunks
}

// NewSQLStore wraps db. Call Migrate once before use.
func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{DB: db, Table: "rag_chunks"}
}

func (s *SQLStore) table() string {
	if s != nil && strings.TrimSpace(s.Table) != "" {
		t := strings.TrimSpace(s.Table)
		for _, c := range t {
			if !(c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return "rag_chunks"
			}
		}
		return t
	}
	return "rag_chunks"
}

// Migrate creates the chunks table and document index if needed.
func (s *SQLStore) Migrate(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("rag: SQLStore requires DB")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	t := s.table()
	_, err := s.DB.ExecContext(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL,
  chunk_index INTEGER NOT NULL,
  text TEXT NOT NULL,
  embedding TEXT NOT NULL,
  metadata TEXT
)`, t))
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s_doc_idx ON %s(document_id)`, t, t))
	return err
}

// Upsert implements VectorStore.
func (s *SQLStore) Upsert(ctx context.Context, chunks []Chunk) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.DB == nil {
		return fmt.Errorf("rag: SQLStore requires DB")
	}
	t := s.table()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	del, err := tx.PrepareContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, t))
	if err != nil {
		return err
	}
	defer del.Close()
	ins, err := tx.PrepareContext(ctx, fmt.Sprintf(`
INSERT INTO %s (id, document_id, chunk_index, text, embedding, metadata)
VALUES (?, ?, ?, ?, ?, ?)`, t))
	if err != nil {
		return err
	}
	defer ins.Close()
	for _, c := range chunks {
		id := strings.TrimSpace(c.ID)
		if id == "" {
			return fmt.Errorf("rag: chunk id is required")
		}
		if _, err := del.ExecContext(ctx, id); err != nil {
			return err
		}
		if err := execChunk(ctx, ins, c); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func execChunk(ctx context.Context, stmt *sql.Stmt, c Chunk) error {
	id := strings.TrimSpace(c.ID)
	if id == "" {
		return fmt.Errorf("rag: chunk id is required")
	}
	if len(c.Embedding) == 0 {
		return fmt.Errorf("rag: chunk %s missing embedding", id)
	}
	emb, err := json.Marshal(c.Embedding)
	if err != nil {
		return err
	}
	var meta []byte
	if len(c.Metadata) > 0 {
		meta, err = json.Marshal(c.Metadata)
		if err != nil {
			return err
		}
	}
	_, err = stmt.ExecContext(ctx, id, firstNonEmpty(c.DocumentID, "doc"), c.Index, c.Text, string(emb), nullableJSON(meta))
	return err
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

// Search implements VectorStore.
func (s *SQLStore) Search(ctx context.Context, query []float64, topK int) ([]Hit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("rag: SQLStore requires DB")
	}
	if topK <= 0 {
		topK = 5
	}
	t := s.table()
	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(
		`SELECT id, document_id, chunk_index, text, embedding, metadata FROM %s`, t))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []Hit
	for rows.Next() {
		var c Chunk
		var embRaw string
		var meta sql.NullString
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.Index, &c.Text, &embRaw, &meta); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(embRaw), &c.Embedding); err != nil {
			return nil, err
		}
		if meta.Valid && meta.String != "" {
			_ = json.Unmarshal([]byte(meta.String), &c.Metadata)
		}
		hits = append(hits, Hit{Chunk: c, Score: CosineSimilarity(query, c.Embedding)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
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
func (s *SQLStore) DeleteDocument(ctx context.Context, documentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return fmt.Errorf("rag: document id is required")
	}
	if s == nil || s.DB == nil {
		return fmt.Errorf("rag: SQLStore requires DB")
	}
	_, err := s.DB.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE document_id = ?`, s.table()), documentID)
	return err
}

// Len implements VectorStore.
func (s *SQLStore) Len() int {
	if s == nil || s.DB == nil {
		return 0
	}
	var n int
	_ = s.DB.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s`, s.table())).Scan(&n)
	return n
}
