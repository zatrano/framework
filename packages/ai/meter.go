package ai

import (
	"context"
	"sync"
	"time"
)

// ModelPrice is optional USD pricing per 1K tokens for cost estimates.
type ModelPrice struct {
	PromptPer1K     float64 // chat/completion prompt
	CompletionPer1K float64 // chat completion
	EmbedPer1K      float64 // embeddings (falls back to PromptPer1K if 0)
}

// PriceTable maps model id → per-1K USD rates (empty = tokens only, no cost).
type PriceTable map[string]ModelPrice

// MeterBucket aggregates calls for one dimension (provider / model / op).
type MeterBucket struct {
	Calls            int
	Errors           int
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	EstimatedUSD     float64
	Latency          time.Duration
}

// MeterSnapshot is a point-in-time view of UsageMeter totals.
type MeterSnapshot struct {
	MeterBucket
	ByProvider map[string]MeterBucket
	ByModel    map[string]MeterBucket
	ByOp       map[string]MeterBucket
}

// UsageMeter implements Observer and aggregates token/latency/cost stats.
type UsageMeter struct {
	mu      sync.Mutex
	prices  PriceTable
	next    Observer
	total   MeterBucket
	byProv  map[string]MeterBucket
	byModel map[string]MeterBucket
	byOp    map[string]MeterBucket
}

// NewUsageMeter creates a meter. Optional next Observer is chained after recording.
func NewUsageMeter(prices PriceTable, next ...Observer) *UsageMeter {
	m := &UsageMeter{
		prices:  prices,
		byProv:  make(map[string]MeterBucket),
		byModel: make(map[string]MeterBucket),
		byOp:    make(map[string]MeterBucket),
	}
	if len(next) > 0 {
		m.next = next[0]
	}
	return m
}

// OnRequest implements Observer (forwards to next if set).
func (m *UsageMeter) OnRequest(ctx context.Context, info RequestInfo) {
	if m != nil && m.next != nil {
		m.next.OnRequest(ctx, info)
	}
}

// OnResult implements Observer.
func (m *UsageMeter) OnResult(ctx context.Context, info ResultInfo) {
	if m == nil {
		return
	}
	m.record(info)
	if m.next != nil {
		m.next.OnResult(ctx, info)
	}
}

func (m *UsageMeter) record(info ResultInfo) {
	cost := estimateCost(info.Model, info.Op, info.Usage, m.prices)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.total = addBucket(m.total, info, cost)
	prov := info.Provider
	if prov == "" {
		prov = info.RequestInfo.Provider
	}
	if prov == "" {
		prov = "default"
	}
	m.byProv[prov] = addBucket(m.byProv[prov], info, cost)
	model := info.Model
	if model == "" {
		model = "unknown"
	}
	m.byModel[model] = addBucket(m.byModel[model], info, cost)
	op := info.Op
	if op == "" {
		op = "unknown"
	}
	m.byOp[op] = addBucket(m.byOp[op], info, cost)
}

func addBucket(b MeterBucket, info ResultInfo, cost float64) MeterBucket {
	b.Calls++
	if info.Err != nil {
		b.Errors++
	}
	b.PromptTokens += info.Usage.PromptTokens
	b.CompletionTokens += info.Usage.CompletionTokens
	b.TotalTokens += info.Usage.TotalTokens
	b.EstimatedUSD += cost
	b.Latency += info.Latency
	return b
}

func estimateCost(model, op string, u Usage, prices PriceTable) float64 {
	if len(prices) == 0 || model == "" {
		return 0
	}
	p, ok := prices[model]
	if !ok {
		return 0
	}
	if op == "embed" {
		rate := p.EmbedPer1K
		if rate == 0 {
			rate = p.PromptPer1K
		}
		tokens := u.TotalTokens
		if tokens == 0 {
			tokens = u.PromptTokens
		}
		return float64(tokens) / 1000.0 * rate
	}
	return float64(u.PromptTokens)/1000.0*p.PromptPer1K +
		float64(u.CompletionTokens)/1000.0*p.CompletionPer1K
}

// Snapshot returns a copy of current aggregates.
func (m *UsageMeter) Snapshot() MeterSnapshot {
	if m == nil {
		return MeterSnapshot{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return MeterSnapshot{
		MeterBucket: m.total,
		ByProvider:  copyBuckets(m.byProv),
		ByModel:     copyBuckets(m.byModel),
		ByOp:        copyBuckets(m.byOp),
	}
}

// Reset clears all aggregates.
func (m *UsageMeter) Reset() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.total = MeterBucket{}
	m.byProv = make(map[string]MeterBucket)
	m.byModel = make(map[string]MeterBucket)
	m.byOp = make(map[string]MeterBucket)
}

func copyBuckets(in map[string]MeterBucket) map[string]MeterBucket {
	out := make(map[string]MeterBucket, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// MultiObserver fans out to multiple observers (nil entries skipped).
type MultiObserver []Observer

func (m MultiObserver) OnRequest(ctx context.Context, info RequestInfo) {
	for _, o := range m {
		if o != nil {
			o.OnRequest(ctx, info)
		}
	}
}

func (m MultiObserver) OnResult(ctx context.Context, info ResultInfo) {
	for _, o := range m {
		if o != nil {
			o.OnResult(ctx, info)
		}
	}
}
