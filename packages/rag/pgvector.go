package rag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// PGVectorStore is a Postgres + pgvector VectorStore using ANN distance search.
// Requires the pgvector extension (`CREATE EXTENSION vector`).
type PGVectorStore struct {
	DB     *sql.DB
	Table  string // default rag_chunks_pg
	Dims   int    // required embedding dimensions (e.g. 1536)
	Metric string // cosine (default) | l2 | ip
}

// NewPGVectorStore wraps a Postgres *sql.DB. Call Migrate before use.
func NewPGVectorStore(db *sql.DB, dims int) *PGVectorStore {
	return &PGVectorStore{DB: db, Table: "rag_chunks_pg", Dims: dims, Metric: "cosine"}
}

func (s *PGVectorStore) table() string {
	if s != nil && strings.TrimSpace(s.Table) != "" {
		t := strings.TrimSpace(s.Table)
		for _, c := range t {
			if !(c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return "rag_chunks_pg"
			}
		}
		return t
	}
	return "rag_chunks_pg"
}

func (s *PGVectorStore) dims() (int, error) {
	if s == nil || s.Dims <= 0 {
		return 0, fmt.Errorf("rag: PGVectorStore Dims must be > 0")
	}
	return s.Dims, nil
}

func (s *PGVectorStore) ops() (string, error) {
	m := strings.ToLower(strings.TrimSpace(s.Metric))
	if m == "" {
		m = "cosine"
	}
	switch m {
	case "cosine":
		return "<=>", nil
	case "l2", "euclidean":
		return "<->", nil
	case "ip", "inner_product":
		return "<#>", nil
	default:
		return "", fmt.Errorf("rag: unknown pgvector metric %q", s.Metric)
	}
}

// VectorLiteral formats a float slice as a pgvector input literal: [1,2,3].
func VectorLiteral(v []float64) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(x, 'f', -1, 64))
	}
	b.WriteByte(']')
	return b.String()
}

// Migrate enables pgvector (best-effort) and creates the table + optional IVFFlat index hint via sequential scan-friendly btree on document_id.
func (s *PGVectorStore) Migrate(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("rag: PGVectorStore requires DB")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	dims, err := s.dims()
	if err != nil {
		return err
	}
	t := s.table()
	if _, err := s.DB.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS vector`); err != nil {
		// Extension may require superuser; table create will fail clearly if missing.
		_ = err
	}
	_, err = s.DB.ExecContext(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL,
  chunk_index INTEGER NOT NULL,
  text TEXT NOT NULL,
  embedding vector(%d) NOT NULL,
  metadata TEXT
)`, t, dims))
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s_doc_idx ON %s(document_id)`, t, t))
	return err
}

// EnsureIndex creates an IVFFlat cosine index (optional; needs enough rows for lists).
func (s *PGVectorStore) EnsureIndex(ctx context.Context, lists int) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("rag: PGVectorStore requires DB")
	}
	if lists <= 0 {
		lists = 100
	}
	ops, err := s.ops()
	if err != nil {
		return err
	}
	class := "vector_cosine_ops"
	switch ops {
	case "<->":
		class = "vector_l2_ops"
	case "<#>":
		class = "vector_ip_ops"
	}
	t := s.table()
	_, err = s.DB.ExecContext(ctx, fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s_embedding_idx ON %s USING ivfflat (embedding %s) WITH (lists = %d)`,
		t, t, class, lists))
	return err
}

// Upsert implements VectorStore.
func (s *PGVectorStore) Upsert(ctx context.Context, chunks []Chunk) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.DB == nil {
		return fmt.Errorf("rag: PGVectorStore requires DB")
	}
	dims, err := s.dims()
	if err != nil {
		return err
	}
	t := s.table()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, c := range chunks {
		id := strings.TrimSpace(c.ID)
		if id == "" {
			return fmt.Errorf("rag: chunk id is required")
		}
		if len(c.Embedding) == 0 {
			return fmt.Errorf("rag: chunk %s missing embedding", id)
		}
		if len(c.Embedding) != dims {
			return fmt.Errorf("rag: chunk %s embedding dims %d != %d", id, len(c.Embedding), dims)
		}
		var meta any
		if len(c.Metadata) > 0 {
			b, err := json.Marshal(c.Metadata)
			if err != nil {
				return err
			}
			meta = string(b)
		}
		lit := VectorLiteral(c.Embedding)
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, t), id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %s (id, document_id, chunk_index, text, embedding, metadata)
VALUES ($1, $2, $3, $4, $5::vector, $6)`, t),
			id, firstNonEmpty(c.DocumentID, "doc"), c.Index, c.Text, lit, meta)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Search implements VectorStore using pgvector distance operators.
func (s *PGVectorStore) Search(ctx context.Context, query []float64, topK int) ([]Hit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("rag: PGVectorStore requires DB")
	}
	dims, err := s.dims()
	if err != nil {
		return nil, err
	}
	if len(query) != dims {
		return nil, fmt.Errorf("rag: query dims %d != %d", len(query), dims)
	}
	if topK <= 0 {
		topK = 5
	}
	ops, err := s.ops()
	if err != nil {
		return nil, err
	}
	t := s.table()
	lit := VectorLiteral(query)
	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(`
SELECT id, document_id, chunk_index, text, metadata, (embedding %s $1::vector) AS dist
FROM %s
ORDER BY embedding %s $1::vector
LIMIT $2`, ops, t, ops), lit, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []Hit
	for rows.Next() {
		var c Chunk
		var meta sql.NullString
		var dist float64
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.Index, &c.Text, &meta, &dist); err != nil {
			return nil, err
		}
		if meta.Valid && meta.String != "" {
			_ = json.Unmarshal([]byte(meta.String), &c.Metadata)
		}
		// Convert distance to a higher-is-better score.
		score := 1 / (1 + dist)
		if ops == "<#>" {
			// inner product distance is negative IP in pgvector; higher IP → lower dist magnitude
			score = -dist
		}
		hits = append(hits, Hit{Chunk: c, Score: score})
	}
	return hits, rows.Err()
}

// DeleteDocument implements VectorStore.
func (s *PGVectorStore) DeleteDocument(ctx context.Context, documentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return fmt.Errorf("rag: document id is required")
	}
	if s == nil || s.DB == nil {
		return fmt.Errorf("rag: PGVectorStore requires DB")
	}
	_, err := s.DB.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE document_id = $1`, s.table()), documentID)
	return err
}

// Len implements VectorStore.
func (s *PGVectorStore) Len() int {
	if s == nil || s.DB == nil {
		return 0
	}
	var n int
	_ = s.DB.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s`, s.table())).Scan(&n)
	return n
}
