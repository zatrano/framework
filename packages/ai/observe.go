package ai

import (
	"context"
	"time"
)

// Observer receives lightweight AI call telemetry (nil-safe; not a metrics stack).
type Observer interface {
	OnRequest(ctx context.Context, info RequestInfo)
	OnResult(ctx context.Context, info ResultInfo)
}

// RequestInfo describes an AI call about to start.
type RequestInfo struct {
	Provider string // scoped Using name, or empty when using Profile/default
	Profile  string
	Model    string
	Op       string // "chat", "embed", or "stream"
}

// ResultInfo describes a finished AI call (or stream completion / setup failure).
type ResultInfo struct {
	RequestInfo
	Provider  string // provider that produced the outcome (last tried on failure)
	Latency   time.Duration
	Usage     Usage
	Err       error
	Attempts  int // total driver invocations (including retries)
	Fallbacks int // provider switches after the first
}

// FuncObserver adapts functions to Observer.
type FuncObserver struct {
	Request func(ctx context.Context, info RequestInfo)
	Result  func(ctx context.Context, info ResultInfo)
}

func (o FuncObserver) OnRequest(ctx context.Context, info RequestInfo) {
	if o.Request != nil {
		o.Request(ctx, info)
	}
}

func (o FuncObserver) OnResult(ctx context.Context, info ResultInfo) {
	if o.Result != nil {
		o.Result(ctx, info)
	}
}

func (m *Manager) observer() Observer {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.obs
}

func (m *Manager) notifyRequest(ctx context.Context, info RequestInfo) {
	if o := m.observer(); o != nil {
		o.OnRequest(ctx, info)
	}
}

func (m *Manager) notifyResult(ctx context.Context, info ResultInfo) {
	if o := m.observer(); o != nil {
		o.OnResult(ctx, info)
	}
}
