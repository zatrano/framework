package ai

import (
	"context"
	"fmt"
)

// LogFn is a printf-style logger (compatible with packages/log Infof).
type LogFn func(format string, args ...any)

// LogDriver wraps another driver and logs prompt/reply (or error).
type LogDriver struct {
	Log   LogFn
	Inner Driver
}

func (LogDriver) Name() string { return "log" }

// Capabilities implements Capabler (delegates to Inner).
func (d LogDriver) Capabilities() []Capability {
	inner := d.Inner
	if inner == nil {
		inner = FakeDriver{}
	}
	return InferCapabilities(inner)
}

// Health implements Healthy when Inner does.
func (d LogDriver) Health(ctx context.Context) error {
	inner := d.Inner
	if inner == nil {
		inner = FakeDriver{}
	}
	if h, ok := inner.(Healthy); ok {
		return h.Health(ctx)
	}
	return nil
}

func (d LogDriver) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	inner := d.Inner
	if inner == nil {
		inner = FakeDriver{}
	}
	resp, err := inner.Chat(ctx, req)
	if d.Log != nil {
		prompt := truncate(lastUser(req.Messages), 120)
		if err != nil {
			d.Log("ai: driver=log model=%s prompt=%q err=%v", req.Model, prompt, err)
		} else if resp != nil {
			d.Log("ai: driver=log model=%s prompt=%q reply=%q tokens=%d",
				req.Model, prompt, truncate(resp.Message.Content, 120), resp.Usage.TotalTokens)
		}
	}
	return resp, err
}

// ChatStream delegates when Inner implements StreamDriver.
func (d LogDriver) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	inner := d.Inner
	if inner == nil {
		inner = FakeDriver{}
	}
	sd, ok := inner.(StreamDriver)
	if !ok {
		return nil, &Error{Kind: KindInvalid, Provider: "log", Err: fmt.Errorf("inner driver does not support streaming")}
	}
	if d.Log != nil {
		d.Log("ai: driver=log stream model=%s prompt=%q", req.Model, truncate(lastUser(req.Messages), 120))
	}
	return sd.ChatStream(ctx, req)
}
