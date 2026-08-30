package ai_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/ai"
)

func TestCheckHealthFake(t *testing.T) {
	m := ai.New()
	st := m.CheckHealth(context.Background(), "fake")
	if len(st) != 1 || !st[0].OK || st[0].Err != nil || st[0].Skipped {
		t.Fatalf("%+v", st)
	}
	all := m.CheckHealthAll(context.Background())
	if len(all) < 2 {
		t.Fatalf("%d", len(all))
	}
	names := ai.HealthyProviders(all)
	if len(names) < 2 {
		t.Fatalf("%v", names)
	}
}

func TestCheckHealthOpenAI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Fatalf("auth")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	m := ai.New()
	m.Extend("oa", &ai.OpenAIDriver{
		BaseURL:    srv.URL + "/v1",
		APIKey:     "k",
		HTTPClient: srv.Client(),
	})
	st := m.CheckHealth(context.Background(), "oa")
	if len(st) != 1 || !st[0].OK || st[0].Latency < 0 {
		t.Fatalf("%+v", st)
	}
}

func TestCheckHealthOpenAIFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	m := ai.New()
	m.Extend("oa", &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()})
	st := m.CheckHealth(context.Background(), "oa")
	if st[0].OK || st[0].Err == nil {
		t.Fatalf("%+v", st[0])
	}
	if ai.Classify(st[0].Err) != ai.KindUnavailable {
		t.Fatalf("kind=%v", ai.Classify(st[0].Err))
	}
}

func TestCheckHealthCanceled(t *testing.T) {
	m := ai.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	st := m.CheckHealth(ctx, "fake")
	if st[0].OK || st[0].Err == nil {
		t.Fatalf("%+v", st[0])
	}
	_ = time.Second
}
