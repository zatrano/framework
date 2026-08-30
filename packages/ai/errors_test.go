package ai_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/ai"
)

func TestKindFromStatus(t *testing.T) {
	tests := []struct {
		status int
		want   ai.Kind
	}{
		{401, ai.KindAuth},
		{403, ai.KindAuth},
		{400, ai.KindInvalid},
		{422, ai.KindInvalid},
		{429, ai.KindRateLimit},
		{500, ai.KindUnavailable},
		{502, ai.KindUnavailable},
		{408, ai.KindUnavailable},
		{200, ai.KindUnknown},
	}
	for _, tt := range tests {
		if got := ai.KindFromStatus(tt.status); got != tt.want {
			t.Fatalf("status %d: got %v want %v", tt.status, got, tt.want)
		}
	}
}

func TestClassify(t *testing.T) {
	if ai.Classify(nil) != ai.KindUnknown {
		t.Fatal("nil")
	}
	if ai.Classify(&ai.Error{Kind: ai.KindRateLimit}) != ai.KindRateLimit {
		t.Fatal("typed")
	}
	if ai.Classify(context.Canceled) != ai.KindContext {
		t.Fatal("canceled")
	}
	if ai.Classify(context.DeadlineExceeded) != ai.KindContext {
		t.Fatal("deadline")
	}
	if ai.Classify(fmt.Errorf("wrap: %w", &ai.Error{Kind: ai.KindAuth})) != ai.KindAuth {
		t.Fatal("wrapped")
	}
	var ne net.Error = timeoutNetError{}
	if ai.Classify(ne) != ai.KindUnavailable {
		t.Fatal("net")
	}
	if ai.Classify(io.EOF) != ai.KindUnavailable {
		t.Fatal("eof")
	}
	if ai.Classify(errors.New("boom")) != ai.KindUnknown {
		t.Fatal("unknown")
	}
}

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "i/o timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

func TestRetryableFallbackable(t *testing.T) {
	auth := &ai.Error{Kind: ai.KindAuth}
	invalid := &ai.Error{Kind: ai.KindInvalid}
	rate := &ai.Error{Kind: ai.KindRateLimit}
	unavail := &ai.Error{Kind: ai.KindUnavailable}
	unknown := errors.New("boom")

	if ai.Retryable(auth) || ai.Retryable(invalid) || ai.Retryable(unknown) {
		t.Fatal("should not retry")
	}
	if !ai.Retryable(rate) || !ai.Retryable(unavail) {
		t.Fatal("should retry")
	}
	if ai.Fallbackable(auth, true) || ai.Fallbackable(invalid, true) {
		t.Fatal("auth/invalid no fallback")
	}
	if !ai.Fallbackable(rate, true) || !ai.Fallbackable(unavail, true) || !ai.Fallbackable(unknown, true) {
		t.Fatal("should fallback")
	}
	if ai.Fallbackable(context.Canceled, true) {
		t.Fatal("cancel no fallback")
	}
	if !ai.Fallbackable(context.DeadlineExceeded, true) {
		t.Fatal("deadline fallback when enabled")
	}
	if ai.Fallbackable(context.DeadlineExceeded, false) {
		t.Fatal("deadline no fallback when disabled")
	}
}

type countingDriver struct {
	name    string
	fails   int
	err     error
	calls   int
	succeed *ai.ChatResponse
}

func (d *countingDriver) Name() string { return d.name }
func (d *countingDriver) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	d.calls++
	if d.calls <= d.fails {
		return nil, d.err
	}
	if d.succeed != nil {
		return d.succeed, nil
	}
	return &ai.ChatResponse{
		Model:   req.Model,
		Message: ai.Message{Role: "assistant", Content: "ok:" + req.Messages[0].Content},
	}, nil
}

func TestRetrySameProvider(t *testing.T) {
	m := ai.New()
	m.SetDefaults(ai.Defaults{
		Timeout:           time.Second,
		Retry:             ai.RetryPolicy{MaxRetries: 2, InitialDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, Multiplier: 2},
		FallbackOnTimeout: true,
	})
	d := &countingDriver{
		name:  "flaky",
		fails: 2,
		err:   &ai.Error{Kind: ai.KindRateLimit, Status: 429, Err: errors.New("slow down")},
	}
	m.Extend("flaky", d)
	m.Use("flaky")
	resp, err := m.Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "retry-me"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.calls != 3 {
		t.Fatalf("calls=%d want 3", d.calls)
	}
	if !strings.Contains(resp.Message.Content, "retry-me") {
		t.Fatalf("%v", resp.Message.Content)
	}
}

func TestAuthNoFallback(t *testing.T) {
	m := ai.New()
	m.SetDefaults(ai.Defaults{
		Timeout: time.Second,
		Retry:   ai.RetryPolicy{MaxRetries: 2, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond},
	})
	auth := &countingDriver{
		name:  "auth",
		err:   &ai.Error{Kind: ai.KindAuth, Status: 401, Err: errors.New("bad key")},
		fails: 100,
	}
	ok := &countingDriver{name: "ok"}
	m.Extend("auth", auth)
	m.Extend("ok", ok)
	m.SetProfile("content", ai.Profile{Providers: []string{"auth", "ok"}})
	_, err := m.Profile("content").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected auth error")
	}
	if ai.Classify(err) != ai.KindAuth {
		t.Fatalf("kind=%v", ai.Classify(err))
	}
	if auth.calls != 1 {
		t.Fatalf("auth calls=%d (no retry)", auth.calls)
	}
	if ok.calls != 0 {
		t.Fatalf("ok should not be called, calls=%d", ok.calls)
	}
}

func TestUnavailableThenFallback(t *testing.T) {
	m := ai.New()
	m.SetDefaults(ai.Defaults{
		Timeout: time.Second,
		Retry:   ai.RetryPolicy{MaxRetries: 1, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond},
	})
	broken := &countingDriver{
		name:  "broken",
		fails: 100,
		err:   &ai.Error{Kind: ai.KindUnavailable, Status: 500, Err: errors.New("boom")},
	}
	ok := &countingDriver{name: "ok"}
	m.Extend("broken", broken)
	m.Extend("ok", ok)
	m.SetProfile("content", ai.Profile{Providers: []string{"broken", "ok"}})
	resp, err := m.Profile("content").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "fb"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if broken.calls != 2 { // 1 + 1 retry
		t.Fatalf("broken calls=%d", broken.calls)
	}
	if ok.calls != 1 {
		t.Fatalf("ok calls=%d", ok.calls)
	}
	if !strings.Contains(resp.Message.Content, "fb") {
		t.Fatalf("%v", resp.Message.Content)
	}
}

func TestOpenAIHTTPErrorKind(t *testing.T) {
	err := ai.HTTPError("openai", 429, `{"error":"rate"}`, 2*time.Second)
	if ai.Classify(err) != ai.KindRateLimit {
		t.Fatal(ai.Classify(err))
	}
	if err.RetryAfter != 2*time.Second {
		t.Fatal(err.RetryAfter)
	}
}

func TestInvalidNoRetryNoFallback(t *testing.T) {
	m := ai.New()
	m.SetDefaults(ai.Defaults{
		Timeout: time.Second,
		Retry:   ai.RetryPolicy{MaxRetries: 3, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond},
	})
	bad := &countingDriver{
		name:  "bad",
		fails: 100,
		err:   &ai.Error{Kind: ai.KindInvalid, Status: 400, Err: errors.New("bad req")},
	}
	ok := &countingDriver{name: "ok"}
	m.Extend("bad", bad)
	m.Extend("ok", ok)
	m.SetProfile("p", ai.Profile{Providers: []string{"bad", "ok"}})
	_, err := m.Profile("p").Chat(context.Background(), ai.ChatRequest{
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil || ai.Classify(err) != ai.KindInvalid {
		t.Fatalf("%v", err)
	}
	if bad.calls != 1 || ok.calls != 0 {
		t.Fatalf("bad=%d ok=%d", bad.calls, ok.calls)
	}
}
