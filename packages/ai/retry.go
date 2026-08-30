package ai

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryPolicy controls per-provider retries before fallback.
type RetryPolicy struct {
	MaxRetries   int           // attempts after the first try (default 2 → up to 3 total calls)
	InitialDelay time.Duration // default 200ms
	MaxDelay     time.Duration // default 2s
	Multiplier   float64       // default 2.0
}

// DefaultRetryPolicy returns conservative defaults for production.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:   2,
		InitialDelay: 200 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Multiplier:   2.0,
	}
}

func (p RetryPolicy) normalized() RetryPolicy {
	d := DefaultRetryPolicy()
	if p.MaxRetries < 0 {
		p.MaxRetries = 0
	}
	if p.InitialDelay <= 0 {
		p.InitialDelay = d.InitialDelay
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = d.MaxDelay
	}
	if p.Multiplier < 1 {
		p.Multiplier = d.Multiplier
	}
	return p
}

// backoffDelay returns wait before attempt index (0 = first retry after failure).
func (p RetryPolicy) backoffDelay(attempt int, retryAfter time.Duration) time.Duration {
	p = p.normalized()
	if retryAfter > 0 {
		if retryAfter > p.MaxDelay {
			return p.MaxDelay
		}
		return retryAfter
	}
	// delay = InitialDelay * Multiplier^attempt + small jitter
	delay := float64(p.InitialDelay) * math.Pow(p.Multiplier, float64(attempt))
	if delay > float64(p.MaxDelay) {
		delay = float64(p.MaxDelay)
	}
	jitter := delay * 0.1 * rand.Float64() //nolint:gosec // non-crypto jitter
	return time.Duration(delay + jitter)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
