package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Client is a scoped AI entry point (Using provider or Profile).
type Client struct {
	mgr       *Manager
	provider  string
	profile   string
	providers []string // optional explicit fallback chain (ProfileLive / UsingLive)
}

// Chat runs chat with profile overrides, per-provider retry, and ordered fallback.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if c == nil || c.mgr == nil {
		return nil, fmt.Errorf("ai: client is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	c.mgr.mu.RLock()
	defs := c.mgr.defaults
	names, req2, err := c.resolveChatLocked(req, defs)
	timeout := defs.Timeout
	retry := defs.Retry.normalized()
	fallbackOnTimeout := defs.FallbackOnTimeout
	c.mgr.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	base := RequestInfo{Provider: c.provider, Profile: c.profile, Model: req2.Model, Op: "chat"}
	c.mgr.notifyRequest(ctx, base)

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var (
		lastErr    error
		lastProv   string
		attempts   int
		fallbacks  int
		triedCount int
	)
	for _, name := range names {
		if triedCount > 0 {
			fallbacks++
		}
		triedCount++
		lastProv = name
		d, err := c.mgr.Driver(name)
		if err != nil {
			lastErr = err
			if !Fallbackable(err, fallbackOnTimeout) {
				c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, err))
				return nil, err
			}
			continue
		}
		resp, n, err := callWithRetry(ctx, retry, func(ctx context.Context) (*ChatResponse, error) {
			return d.Chat(ctx, req2)
		})
		attempts += n
		if err == nil {
			usage := Usage{}
			if resp != nil {
				usage = resp.Usage
			}
			c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, usage, nil))
			return resp, nil
		}
		lastErr = err
		if !Fallbackable(err, fallbackOnTimeout) {
			c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, err))
			return nil, err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ai: no providers available")
	}
	c.mgr.notifyResult(ctx, resultFrom(base, lastProv, start, attempts, fallbacks, Usage{}, lastErr))
	return nil, lastErr
}

// Embed runs embeddings with the same Using/Profile scoping when the driver supports it.
func (c *Client) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	if c == nil || c.mgr == nil {
		return nil, fmt.Errorf("ai: client is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	c.mgr.mu.RLock()
	defs := c.mgr.defaults
	names, err := c.resolveNamesLocked()
	timeout := defs.Timeout
	retry := defs.Retry.normalized()
	fallbackOnTimeout := defs.FallbackOnTimeout
	modelDefault := defs.Model
	if c.profile != "" {
		if p, ok := c.mgr.profileLocked(c.profile); ok && p.Model != "" {
			modelDefault = p.Model
		}
	}
	c.mgr.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	if req.Model == "" {
		req.Model = modelDefault
	}

	start := time.Now()
	base := RequestInfo{Provider: c.provider, Profile: c.profile, Model: req.Model, Op: "embed"}
	c.mgr.notifyRequest(ctx, base)

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var (
		lastErr    error
		lastProv   string
		attempts   int
		fallbacks  int
		triedCount int
	)
	for _, name := range names {
		if triedCount > 0 {
			fallbacks++
		}
		triedCount++
		lastProv = name
		d, err := c.mgr.Driver(name)
		if err != nil {
			lastErr = err
			if !Fallbackable(err, fallbackOnTimeout) {
				c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, err))
				return nil, err
			}
			continue
		}
		ed, ok := d.(EmbeddingDriver)
		if !ok {
			lastErr = fmt.Errorf("ai: driver [%s] does not support embeddings", name)
			continue
		}
		resp, n, err := callWithRetry(ctx, retry, func(ctx context.Context) (*EmbedResponse, error) {
			return ed.Embed(ctx, req)
		})
		attempts += n
		if err == nil {
			usage := Usage{}
			if resp != nil {
				usage = resp.Usage
			}
			c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, usage, nil))
			return resp, nil
		}
		lastErr = err
		if !Fallbackable(err, fallbackOnTimeout) {
			c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, err))
			return nil, err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ai: no embedding providers available")
	}
	c.mgr.notifyResult(ctx, resultFrom(base, lastProv, start, attempts, fallbacks, Usage{}, lastErr))
	return nil, lastErr
}

// ChatStream streams chat completions. Fallback applies only to stream setup
// (before a channel is returned). Mid-stream errors are delivered on the channel
// and do not switch providers.
func (c *Client) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	if c == nil || c.mgr == nil {
		return nil, fmt.Errorf("ai: client is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	c.mgr.mu.RLock()
	defs := c.mgr.defaults
	names, req2, err := c.resolveChatLocked(req, defs)
	timeout := defs.Timeout
	retry := defs.Retry.normalized()
	fallbackOnTimeout := defs.FallbackOnTimeout
	c.mgr.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	base := RequestInfo{Provider: c.provider, Profile: c.profile, Model: req2.Model, Op: "stream"}
	c.mgr.notifyRequest(ctx, base)

	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}

	var (
		lastErr    error
		lastProv   string
		attempts   int
		fallbacks  int
		triedCount int
	)
	for _, name := range names {
		if triedCount > 0 {
			fallbacks++
		}
		triedCount++
		lastProv = name
		d, err := c.mgr.Driver(name)
		if err != nil {
			lastErr = err
			if !Fallbackable(err, fallbackOnTimeout) {
				if cancel != nil {
					cancel()
				}
				c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, err))
				return nil, err
			}
			continue
		}
		sd, ok := d.(StreamDriver)
		if !ok {
			lastErr = fmt.Errorf("ai: driver [%s] does not support streaming", name)
			continue
		}
		ch, n, err := callWithRetry(ctx, retry, func(ctx context.Context) (<-chan StreamChunk, error) {
			return sd.ChatStream(ctx, req2)
		})
		attempts += n
		if err == nil {
			ch = observeStream(ctx, c.mgr, base, name, start, attempts, fallbacks, ch)
			if cancel != nil {
				return bindStreamCancel(ch, cancel), nil
			}
			return ch, nil
		}
		lastErr = err
		if !Fallbackable(err, fallbackOnTimeout) {
			if cancel != nil {
				cancel()
			}
			c.mgr.notifyResult(ctx, resultFrom(base, name, start, attempts, fallbacks, Usage{}, err))
			return nil, err
		}
	}
	if cancel != nil {
		cancel()
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ai: no streaming providers available")
	}
	c.mgr.notifyResult(ctx, resultFrom(base, lastProv, start, attempts, fallbacks, Usage{}, lastErr))
	return nil, lastErr
}

func resultFrom(base RequestInfo, provider string, start time.Time, attempts, fallbacks int, usage Usage, err error) ResultInfo {
	return ResultInfo{
		RequestInfo: base,
		Provider:    provider,
		Latency:     time.Since(start),
		Usage:       usage,
		Err:         err,
		Attempts:    attempts,
		Fallbacks:   fallbacks,
	}
}

// observeStream emits OnResult when the stream channel is fully drained.
func observeStream(ctx context.Context, mgr *Manager, base RequestInfo, provider string, start time.Time, attempts, fallbacks int, ch <-chan StreamChunk) <-chan StreamChunk {
	if mgr == nil || mgr.observer() == nil {
		return ch
	}
	out := make(chan StreamChunk, 16)
	go func() {
		defer close(out)
		var usage Usage
		var streamErr error
		for chunk := range ch {
			if chunk.Usage != nil {
				usage = *chunk.Usage
			}
			if chunk.Err != nil {
				streamErr = chunk.Err
			}
			out <- chunk
		}
		mgr.notifyResult(ctx, resultFrom(base, provider, start, attempts, fallbacks, usage, streamErr))
	}()
	return out
}

// bindStreamCancel cancels the request context after the stream channel is drained.
func bindStreamCancel(ch <-chan StreamChunk, cancel context.CancelFunc) <-chan StreamChunk {
	out := make(chan StreamChunk, 16)
	go func() {
		defer close(out)
		defer cancel()
		for chunk := range ch {
			out <- chunk
		}
	}()
	return out
}

func callWithRetry[T any](ctx context.Context, policy RetryPolicy, fn func(context.Context) (T, error)) (T, int, error) {
	policy = policy.normalized()
	var zero T
	var lastErr error
	attempts := 0
	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, attempts, err
		}
		attempts++
		out, err := fn(ctx)
		if err == nil {
			return out, attempts, nil
		}
		lastErr = err
		if attempt == policy.MaxRetries || !Retryable(err) {
			break
		}
		delay := policy.backoffDelay(attempt, retryAfterOf(err))
		if sleepErr := sleepCtx(ctx, delay); sleepErr != nil {
			return zero, attempts, sleepErr
		}
	}
	return zero, attempts, lastErr
}

func (c *Client) resolveChatLocked(req ChatRequest, defs Defaults) ([]string, ChatRequest, error) {
	names, err := c.resolveNamesLocked()
	if err != nil {
		return nil, req, err
	}
	if c.profile != "" {
		if p, ok := c.mgr.profileLocked(c.profile); ok {
			req = applyProfile(req, p)
		}
	}
	req = applyDefaults(req, defs)
	req = ensureModel(req)
	return names, req, nil
}

func (c *Client) resolveNamesLocked() ([]string, error) {
	if len(c.providers) > 0 {
		return append([]string(nil), c.providers...), nil
	}
	if c.profile != "" {
		p, ok := c.mgr.profileLocked(c.profile)
		if !ok {
			return nil, fmt.Errorf("ai: profile [%s] not configured", c.profile)
		}
		if len(p.Providers) == 0 {
			return nil, fmt.Errorf("ai: profile [%s] has no providers", c.profile)
		}
		return append([]string(nil), p.Providers...), nil
	}
	name := c.provider
	if name == "" {
		name = c.mgr.defaultDriver
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil, fmt.Errorf("ai: no default provider")
	}
	return []string{name}, nil
}
