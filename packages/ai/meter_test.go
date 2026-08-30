package ai_test

import (
	"context"
	"testing"

	"github.com/zatrano/framework/packages/ai"
)

func TestUsageMeter(t *testing.T) {
	m := ai.New()
	prices := ai.PriceTable{
		"zatrano-fake-1": {PromptPer1K: 0.001, CompletionPer1K: 0.002},
	}
	meter := ai.NewUsageMeter(prices)
	m.Observe(meter)

	_, err := m.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "meter me"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Embed(context.Background(), ai.EmbedRequest{Input: []string{"a", "b"}})
	if err != nil {
		t.Fatal(err)
	}

	snap := meter.Snapshot()
	if snap.Calls != 2 || snap.Errors != 0 {
		t.Fatalf("%+v", snap)
	}
	if snap.TotalTokens < 1 || snap.ByOp["chat"].Calls != 1 || snap.ByOp["embed"].Calls != 1 {
		t.Fatalf("%+v", snap)
	}
	if snap.ByProvider["fake"].Calls != 2 {
		t.Fatalf("provider=%+v", snap.ByProvider)
	}
	if snap.EstimatedUSD <= 0 {
		t.Fatalf("expected cost estimate, got %v", snap.EstimatedUSD)
	}

	meter.Reset()
	if meter.Snapshot().Calls != 0 {
		t.Fatal("reset")
	}
}

func TestMultiObserver(t *testing.T) {
	var n int
	inner := ai.FuncObserver{
		Result: func(ctx context.Context, info ai.ResultInfo) { n++ },
	}
	meter := ai.NewUsageMeter(nil, inner)
	m := ai.New()
	m.Observe(meter)
	_, _ = m.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if n != 1 || meter.Snapshot().Calls != 1 {
		t.Fatalf("n=%d calls=%d", n, meter.Snapshot().Calls)
	}
}
