package ai_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/ai"
)

func TestDriversProfilesSorted(t *testing.T) {
	m := ai.New()
	m.Extend("zeta", ai.FakeDriver{})
	m.Extend("alpha", ai.FakeDriver{})
	m.SetProfile("z", ai.Profile{Providers: []string{"fake"}})
	m.SetProfile("a", ai.Profile{Providers: []string{"fake"}})

	drivers := m.Drivers()
	for i := 1; i < len(drivers); i++ {
		if drivers[i-1] > drivers[i] {
			t.Fatalf("unsorted drivers: %v", drivers)
		}
	}
	profiles := m.Profiles()
	if len(profiles) < 2 || profiles[0] != "a" || profiles[len(profiles)-1] != "z" {
		t.Fatalf("profiles=%v", profiles)
	}
}

func TestObserverChat(t *testing.T) {
	m := ai.New()
	var reqs []ai.RequestInfo
	var results []ai.ResultInfo
	m.Observe(ai.FuncObserver{
		Request: func(ctx context.Context, info ai.RequestInfo) { reqs = append(reqs, info) },
		Result:  func(ctx context.Context, info ai.ResultInfo) { results = append(results, info) },
	})
	resp, err := m.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "obs"}},
	})
	if err != nil || resp == nil {
		t.Fatalf("%v %v", err, resp)
	}
	if len(reqs) != 1 || reqs[0].Op != "chat" {
		t.Fatalf("%+v", reqs)
	}
	if len(results) != 1 || results[0].Err != nil || results[0].Attempts < 1 {
		t.Fatalf("%+v", results)
	}
	if results[0].Provider != "fake" || results[0].Usage.TotalTokens < 1 {
		t.Fatalf("%+v", results[0])
	}
	if results[0].Latency < 0 {
		t.Fatal("latency")
	}
}

func TestObserverFallbackCounts(t *testing.T) {
	m := ai.New()
	m.SetDefaults(ai.Defaults{
		Timeout: time.Second,
		Retry:   ai.RetryPolicy{MaxRetries: 0, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond},
	})
	m.Extend("broken", failDriver{err: errors.New("boom")})
	m.Extend("ok", ai.FakeDriver{})
	m.SetProfile("p", ai.Profile{Providers: []string{"broken", "ok"}})

	var result ai.ResultInfo
	m.Observe(ai.FuncObserver{
		Result: func(ctx context.Context, info ai.ResultInfo) { result = info },
	})
	_, err := m.Profile("p").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Fallbacks != 1 || result.Provider != "ok" || result.Profile != "p" {
		t.Fatalf("%+v", result)
	}
}

func TestObserverStream(t *testing.T) {
	m := ai.New()
	var result ai.ResultInfo
	done := make(chan struct{})
	m.Observe(ai.FuncObserver{
		Result: func(ctx context.Context, info ai.ResultInfo) {
			result = info
			close(done)
		},
	})
	ch, err := m.ChatStream(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "stream-obs"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for range ch {
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for OnResult")
	}
	if result.Op != "stream" || result.Err != nil || result.Usage.TotalTokens < 1 {
		t.Fatalf("%+v", result)
	}
}
