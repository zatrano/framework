package kernel

import (
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

// bodyLimitLookupHook counts route lookups in tests. Nil in production.
// It runs only on the slow path (a body that can exceed the default).
var bodyLimitLookupHook func()

// headerBodyConfig is the rawhttp HeaderReceived hook.
// GET, HEAD, and bodies at or under the content-type default return without
// matching a route. A larger Content-Length, or a chunked body, matches once.
func (app *Application) headerBodyConfig(ctx *rawhttp.Ctx) rawhttp.RequestConfig {
	if ctx == nil {
		return rawhttp.RequestConfig{}
	}
	limit := http.HeaderBodyLimit(string(ctx.Header("Content-Type")))
	if !bodyLimitNeedsRoute(string(ctx.Method), ctx.ContentLength(), limit) {
		return http.BodyLimitConfig(limit)
	}
	if bodyLimitLookupHook != nil {
		bodyLimitLookupHook()
	}
	if app != nil && app.router != nil {
		if n, set := app.router.BodyLimitFor(string(ctx.Method), string(ctx.Path)); set {
			if n < 0 {
				return http.BodyLimitConfig(http.MaxRequestBytes())
			}
			return http.BodyLimitConfig(n)
		}
	}
	return http.BodyLimitConfig(limit)
}

func bodyLimitNeedsRoute(method string, contentLength int, limit int64) bool {
	switch strings.ToUpper(method) {
	case "GET", "HEAD":
		return false
	}
	if contentLength < 0 {
		return true
	}
	return int64(contentLength) > limit
}
