package ratelimit

import "github.com/zatrano/framework/contracts"

// From resolves the rate limiter from the application container.
func From(app contracts.App) *Limiter {
	if app == nil {
		return nil
	}
	raw, err := app.Make("rateLimiter")
	if err != nil {
		return nil
	}
	l, _ := raw.(*Limiter)
	return l
}
