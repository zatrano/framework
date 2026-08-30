package ai

import (
	"context"
	"fmt"
	"strings"
)

// Client is a scoped AI entry point (Using provider or Profile).
type Client struct {
	mgr      *Manager
	provider string
	profile  string
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

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var lastErr error
	for _, name := range names {
		d, err := c.mgr.Driver(name)
		if err != nil {
			lastErr = err
			if !Fallbackable(err, fallbackOnTimeout) {
				return nil, err
			}
			continue
		}
		resp, err := callWithRetry(ctx, retry, func(ctx context.Context) (*ChatResponse, error) {
			return d.Chat(ctx, req2)
		})
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !Fallbackable(err, fallbackOnTimeout) {
			return nil, err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ai: no providers available")
	}
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

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var lastErr error
	for _, name := range names {
		d, err := c.mgr.Driver(name)
		if err != nil {
			lastErr = err
			if !Fallbackable(err, fallbackOnTimeout) {
				return nil, err
			}
			continue
		}
		ed, ok := d.(EmbeddingDriver)
		if !ok {
			lastErr = fmt.Errorf("ai: driver [%s] does not support embeddings", name)
			continue
		}
		resp, err := callWithRetry(ctx, retry, func(ctx context.Context) (*EmbedResponse, error) {
			return ed.Embed(ctx, req)
		})
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !Fallbackable(err, fallbackOnTimeout) {
			return nil, err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ai: no embedding providers available")
	}
	return nil, lastErr
}

func callWithRetry[T any](ctx context.Context, policy RetryPolicy, fn func(context.Context) (T, error)) (T, error) {
	policy = policy.normalized()
	var zero T
	var lastErr error
	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		out, err := fn(ctx)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if attempt == policy.MaxRetries || !Retryable(err) {
			break
		}
		delay := policy.backoffDelay(attempt, retryAfterOf(err))
		if sleepErr := sleepCtx(ctx, delay); sleepErr != nil {
			return zero, sleepErr
		}
	}
	return zero, lastErr
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
