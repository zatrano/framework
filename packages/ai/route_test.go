package ai_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/ai"
)

type downDriver struct{}

func (downDriver) Name() string { return "down" }
func (downDriver) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	return nil, errors.New("down")
}
func (downDriver) Health(ctx context.Context) error {
	return ai.HealthError("down", errors.New("unreachable"))
}

func TestProfileLiveSkipsUnhealthy(t *testing.T) {
	m := ai.New()
	m.Extend("down", downDriver{})
	m.Extend("ok", ai.FakeDriver{})
	m.SetProfile("content", ai.Profile{Providers: []string{"down", "ok"}})

	client, err := m.ProfileLive(context.Background(), "content")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "live"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("nil resp")
	}
}

func TestProfileLiveFailIfNone(t *testing.T) {
	m := ai.New()
	m.Extend("down", downDriver{})
	m.SetProfile("p", ai.Profile{Providers: []string{"down"}})
	_, err := m.ProfileLive(context.Background(), "p", ai.FailIfNone())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFilterCapableAndUsingLive(t *testing.T) {
	m := ai.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	m.Extend("oa", &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()})

	got := m.FilterCapable([]string{"fake", "oa"}, ai.CapEmbed)
	if len(got) != 2 {
		t.Fatalf("%v", got)
	}
	client, err := m.UsingLive(context.Background(), []string{"fake", "oa"}, ai.RequireCaps(ai.CapChat))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilterHealthyOrder(t *testing.T) {
	m := ai.New()
	m.Extend("down", downDriver{})
	live := m.FilterHealthy(context.Background(), "down", "fake", "log")
	if len(live) < 1 || live[0] == "down" {
		t.Fatalf("%v", live)
	}
}
