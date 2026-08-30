package ai

import (
	"sort"
	"strings"
)

// SetPrices registers optional USD rates used by PreferCheapest / cost estimates.
func (m *Manager) SetPrices(prices PriceTable) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if prices == nil {
		m.prices = nil
		return
	}
	cp := make(PriceTable, len(prices))
	for k, v := range prices {
		cp[k] = v
	}
	m.prices = cp
}

// Prices returns a copy of registered model prices.
func (m *Manager) Prices() PriceTable {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.prices == nil {
		return nil
	}
	cp := make(PriceTable, len(m.prices))
	for k, v := range m.prices {
		cp[k] = v
	}
	return cp
}

// PreferCheapest returns providers sorted by estimated chat cost of their default model
// (PromptPer1K+CompletionPer1K). Unknown prices sort last; original relative order among equals preserved.
func (m *Manager) PreferCheapest(names ...string) []string {
	return m.rankProviders(names, rankByCost)
}

// PreferSmartest returns providers sorted by capability count then context window
// (from SetModels / InferCapabilities). Higher is better; ties keep input order.
func (m *Manager) PreferSmartest(names ...string) []string {
	return m.rankProviders(names, rankBySmart)
}

type rankMode int

const (
	rankByCost rankMode = iota
	rankBySmart
)

type rankedProvider struct {
	name  string
	idx   int
	cost  float64 // lower better; +Inf if unknown
	score int     // higher better (smart)
}

func (m *Manager) rankProviders(names []string, mode rankMode) []string {
	if m == nil || len(names) == 0 {
		return nil
	}
	chain := normalizeProviderNames(names)
	items := make([]rankedProvider, 0, len(chain))
	for i, n := range chain {
		rp := rankedProvider{name: n, idx: i, cost: 1e18}
		info, err := m.Describe(n)
		if err != nil {
			items = append(items, rp)
			continue
		}
		switch mode {
		case rankByCost:
			rp.cost = m.estimateProviderChatCost(info)
		case rankBySmart:
			rp.score = smartScore(info)
		}
		items = append(items, rp)
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		switch mode {
		case rankByCost:
			if a.cost != b.cost {
				return a.cost < b.cost
			}
		case rankBySmart:
			if a.score != b.score {
				return a.score > b.score
			}
		}
		return a.idx < b.idx
	})
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.name
	}
	return out
}

func (m *Manager) estimateProviderChatCost(info ProviderInfo) float64 {
	model := strings.TrimSpace(info.DefaultModel)
	if model == "" && len(info.Models) > 0 {
		model = info.Models[0].ID
	}
	m.mu.RLock()
	prices := m.prices
	m.mu.RUnlock()
	if prices == nil || model == "" {
		return 1e18
	}
	p, ok := prices[model]
	if !ok {
		return 1e18
	}
	sum := p.PromptPer1K + p.CompletionPer1K
	if sum <= 0 {
		return 1e18
	}
	return sum
}

func smartScore(info ProviderInfo) int {
	score := len(info.Caps) * 10
	maxCtx := 0
	for _, mod := range info.Models {
		if mod.ContextWindow > maxCtx {
			maxCtx = mod.ContextWindow
		}
		score += len(mod.Caps)
	}
	// normalize context into score buckets
	score += maxCtx / 1000
	return score
}
