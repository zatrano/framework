package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Reranker reorders (and optionally truncates) retrieval hits.
type Reranker interface {
	Rerank(ctx context.Context, query string, hits []Hit) ([]Hit, error)
}

// FuncReranker adapts a function to Reranker.
type FuncReranker func(ctx context.Context, query string, hits []Hit) ([]Hit, error)

// Rerank implements Reranker.
func (f FuncReranker) Rerank(ctx context.Context, query string, hits []Hit) ([]Hit, error) {
	return f(ctx, query, hits)
}

// KeywordReranker boosts hits whose text contains query tokens.
// Final score = original Score + Boost * matchRatio.
type KeywordReranker struct {
	Boost float64 // default 0.2
	TopN  int     // 0 = keep all after rerank
}

// Rerank implements Reranker.
func (r KeywordReranker) Rerank(ctx context.Context, query string, hits []Hit) ([]Hit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	boost := r.Boost
	if boost == 0 {
		boost = 0.2
	}
	tokens := tokenize(query)
	out := append([]Hit(nil), hits...)
	for i := range out {
		if len(tokens) == 0 {
			continue
		}
		lower := strings.ToLower(out[i].Text)
		matched := 0
		for _, tok := range tokens {
			if strings.Contains(lower, tok) {
				matched++
			}
		}
		ratio := float64(matched) / float64(len(tokens))
		out[i].Score = out[i].Score + boost*ratio
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].ID < out[j].ID
		}
		return out[i].Score > out[j].Score
	})
	if r.TopN > 0 && len(out) > r.TopN {
		out = out[:r.TopN]
	}
	return out, nil
}

func tokenize(q string) []string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(q)))
	out := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, f := range fields {
		f = strings.Trim(f, ".,!?:;\"'()[]{}")
		if len(f) < 2 || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

// QueryOptions configures Pipeline.QueryWith.
type QueryOptions struct {
	TopK     int      // candidates from store; default 5
	Rerank   Reranker // optional
	RerankK  int      // fetch TopK*RerankK candidates before rerank; default 3 → 3*TopK
	FinalTop int      // after rerank; 0 = TopK
}

// QueryWith retrieves candidates and optionally reranks them.
func (p *Pipeline) QueryWith(ctx context.Context, question string, opts QueryOptions) ([]Hit, error) {
	topK := opts.TopK
	if topK <= 0 {
		topK = 5
	}
	fetch := topK
	if opts.Rerank != nil {
		mult := opts.RerankK
		if mult <= 0 {
			mult = 3
		}
		fetch = topK * mult
	}
	hits, err := p.Query(ctx, question, fetch)
	if err != nil {
		return nil, err
	}
	if opts.Rerank == nil {
		if len(hits) > topK {
			hits = hits[:topK]
		}
		return hits, nil
	}
	reranked, err := opts.Rerank.Rerank(ctx, question, hits)
	if err != nil {
		return nil, err
	}
	final := opts.FinalTop
	if final <= 0 {
		final = topK
	}
	if len(reranked) > final {
		reranked = reranked[:final]
	}
	return reranked, nil
}

// QueryRerank is QueryWith with a reranker and default candidate multiplier.
func (p *Pipeline) QueryRerank(ctx context.Context, question string, topK int, r Reranker) ([]Hit, error) {
	if r == nil {
		return nil, fmt.Errorf("rag: reranker is required")
	}
	return p.QueryWith(ctx, question, QueryOptions{TopK: topK, Rerank: r})
}
