package ai_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/ai"
)

func TestWatchHealthHealthyFirst(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer down.Close()

	m := ai.New()
	m.Extend("down", &ai.OpenAIDriver{BaseURL: down.URL + "/v1", HTTPClient: down.Client()})
	m.Extend("up", ai.FakeDriver{})
	m.SetProfile("chat", ai.Profile{Providers: []string{"down", "up"}})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updated := make(chan []string, 1)
	go func() {
		_ = m.WatchHealth(ctx, ai.HealthWatchConfig{
			Interval: time.Hour,
			OnUpdate: func(_ string, providers []string) {
				updated <- append([]string{}, providers...)
				cancel()
			},
		})
	}()

	select {
	case providers := <-updated:
		if len(providers) != 2 || providers[0] != "up" || providers[1] != "down" {
			t.Fatalf("%v", providers)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for health update")
	}
}

func TestWatchHealthOnlyHealthy(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer down.Close()

	m := ai.New()
	m.Extend("down", &ai.OpenAIDriver{BaseURL: down.URL + "/v1", HTTPClient: down.Client()})
	m.Extend("up", ai.FakeDriver{})
	m.SetProfile("chat", ai.Profile{Providers: []string{"down", "up"}})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updated := make(chan []string, 1)
	go func() {
		_ = m.WatchHealth(ctx, ai.HealthWatchConfig{
			Interval:    time.Hour,
			OnlyHealthy: true,
			OnUpdate: func(_ string, providers []string) {
				updated <- append([]string{}, providers...)
				cancel()
			},
		})
	}()
	select {
	case providers := <-updated:
		if len(providers) != 1 || providers[0] != "up" {
			t.Fatalf("%v", providers)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestWatchHealthNoChangeNoUpdate(t *testing.T) {
	m := ai.New()
	m.SetProfile("chat", ai.Profile{Providers: []string{"fake", "log"}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var called atomic.Bool
	go func() {
		_ = m.WatchHealth(ctx, ai.HealthWatchConfig{
			Interval: 20 * time.Millisecond,
			OnUpdate: func(string, []string) { called.Store(true) },
		})
	}()
	time.Sleep(80 * time.Millisecond)
	cancel()
	if called.Load() {
		t.Fatal("expected no OnUpdate when order unchanged")
	}
}
