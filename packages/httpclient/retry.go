package httpclient

import (
	"math"
	"time"
)

// RetryPolicy controls outbound request retries.
type RetryPolicy struct {
	MaxAttempts int
	Wait        time.Duration
	Multiplier  float64
	RetryOn     func(status int, err error) bool
}

// DefaultRetry returns a sensible retry policy for transient failures.
func DefaultRetry(maxAttempts int) RetryPolicy {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return RetryPolicy{
		MaxAttempts: maxAttempts,
		Wait:        100 * time.Millisecond,
		Multiplier:  2,
		RetryOn: func(status int, err error) bool {
			if err != nil {
				return true
			}
			return status == 408 || status == 429 || status >= 500
		},
	}
}

// Retry attaches a retry policy to the pending request.
func (p *PendingRequest) Retry(policy RetryPolicy) *PendingRequest {
	p.retry = &policy
	return p
}

// RetryTimes is a shorthand for DefaultRetry.
func (p *PendingRequest) RetryTimes(times int) *PendingRequest {
	return p.Retry(DefaultRetry(times))
}

func (p *PendingRequest) backoff(attempt int) time.Duration {
	if p.retry == nil {
		return 0
	}
	wait := p.retry.Wait
	if wait <= 0 {
		wait = 100 * time.Millisecond
	}
	mult := p.retry.Multiplier
	if mult <= 0 {
		mult = 2
	}
	return time.Duration(float64(wait) * math.Pow(mult, float64(attempt)))
}
