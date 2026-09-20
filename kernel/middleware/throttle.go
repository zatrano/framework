package middleware

import (
	"fmt"
	"time"

	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

// AttemptLimiter is the HTTP throttle contract. packages/ratelimit implements it.
// The kernel does not import that package and does not know about Redis or databases.
//
// Take must check-and-increment in one critical section. allowed=false is
// fail-closed (HTTP 429). Backend errors are not part of this interface: an
// in-memory limiter cannot fail; a future store maps its own errors onto
// allowed=false if it needs fail-closed behavior.
type AttemptLimiter interface {
	Take(key string, maxAttempts int, decay time.Duration) (allowed bool, remaining int, retryAfter int)
}

// Throttle limits requests using a limiter that is not HTTP-specific.
//
// Fail-open: a nil limiter or nil key function cannot enforce a policy, so the
// request passes through. Fail-closed: Take returning allowed=false yields 429.
// packages/ratelimit.Named is separately fail-closed (HTTP 500) when the named
// policy is missing — that is misconfiguration, not a limiter backend error.
func Throttle(limiter AttemptLimiter, maxAttempts int, decay time.Duration, key func(*http.Request) string) routing.MiddlewareFunc {
	if limiter == nil || key == nil {
		return func(next routing.HandlerFunc) routing.HandlerFunc {
			return next
		}
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if decay <= 0 {
		decay = time.Minute
	}
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			k := key(req)
			allowed, remaining, retryAfter := limiter.Take(k, maxAttempts, decay)
			if !allowed {
				resp := http.JSON(map[string]any{
					"message": "Too Many Attempts.",
				}).Status(429)
				resp.Header("Retry-After", fmt.Sprint(retryAfter))
				resp.Header("X-RateLimit-Limit", fmt.Sprint(maxAttempts))
				resp.Header("X-RateLimit-Remaining", "0")
				return resp
			}
			if remaining < 0 {
				remaining = 0
			}

			resp := next(req)
			if resp != nil {
				resp.Header("X-RateLimit-Limit", fmt.Sprint(maxAttempts))
				resp.Header("X-RateLimit-Remaining", fmt.Sprint(remaining))
			}
			return resp
		}
	}
}
