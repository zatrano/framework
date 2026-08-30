package ai_test

import (
	"testing"

	"github.com/zatrano/framework/packages/ai"
)

func TestPreferCheapest(t *testing.T) {
	m := ai.New()
	m.Extend("cheap", &ai.OpenAIDriver{APIKey: "k", Model: "cheap-model"})
	m.Extend("pricey", &ai.OpenAIDriver{APIKey: "k", Model: "pricey-model"})
	m.SetPrices(ai.PriceTable{
		"cheap-model":  {PromptPer1K: 0.01, CompletionPer1K: 0.02},
		"pricey-model": {PromptPer1K: 1.0, CompletionPer1K: 2.0},
	})
	got := m.PreferCheapest("pricey", "cheap")
	if len(got) != 2 || got[0] != "cheap" || got[1] != "pricey" {
		t.Fatalf("%v", got)
	}
}

func TestPreferSmartest(t *testing.T) {
	m := ai.New()
	m.Extend("basic", &ai.OpenAIDriver{APIKey: "k", Model: "small"})
	m.Extend("rich", &ai.OpenAIDriver{APIKey: "k", Model: "big"})
	m.SetModels("basic", ai.ModelInfo{ID: "small", ContextWindow: 4096})
	m.SetModels("rich", ai.ModelInfo{ID: "big", ContextWindow: 128000, Caps: []ai.Capability{ai.CapChat, ai.CapTools, ai.CapVision}})
	got := m.PreferSmartest("basic", "rich")
	if len(got) != 2 || got[0] != "rich" {
		t.Fatalf("%v", got)
	}
}
