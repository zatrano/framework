package kernel

import (
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/middleware"
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
	return app.decideBodyLimit(
		string(ctx.Method),
		string(ctx.Path),
		string(ctx.Header("Content-Type")),
		string(ctx.Header("X-HTTP-Method-Override")),
		ctx.ContentLength(),
	)
}

// decideBodyLimit is the header-time cap. method/path go through the same
// resolver as Dispatch. An override that is not known until the body is read,
// or more than one candidate route, keeps the tightest cap and never raises it.
func (app *Application) decideBodyLimit(method, path, contentType, override string, contentLength int) rawhttp.RequestConfig {
	limit := http.HeaderBodyLimit(contentType)
	resolved, ambiguous := middleware.MethodOverrideFromHeader(method, override, contentType)
	if isBodyLimitSafeMethod(resolved) {
		return http.BodyLimitConfig(limit)
	}
	tighter := app != nil && app.router != nil && app.router.HasTighterBodyLimit(limit)
	if !bodyLimitNeedsRoute(resolved, contentLength, limit) && !ambiguous && !tighter {
		return http.BodyLimitConfig(limit)
	}
	if bodyLimitLookupHook != nil {
		bodyLimitLookupHook()
	}
	chosen := limit
	if ambiguous {
		chosen = app.tightestBodyLimit(path, limit, "POST", "PUT", "PATCH", "DELETE")
	} else if app != nil && app.router != nil {
		if n, set := app.router.BodyLimitFor(resolved, path); set {
			chosen = routeBodyCap(n)
		}
	}
	if !bodyLimitSustainable(chosen) {
		// The route cap is above the server ceiling and above the in-flight
		// budget, so no reservation can ever succeed. Cut at one byte: the
		// engine answers 413. This is not a retryable 503.
		return http.BodyLimitConfig(1)
	}
	return http.BodyLimitConfig(chosen)
}

func routeBodyCap(n int64) int64 {
	if n < 0 {
		return http.MaxRequestBytes()
	}
	return n
}

// bodyLimitSustainable reports whether a route cap can be granted.
// rawhttp RequestConfig.MaxRequestBodySize replaces the server ceiling, so a
// cap above MaxRequestBytes is honored. A cap that is also above the in-flight
// budget can never be reserved; the caller must reject it.
func bodyLimitSustainable(n int64) bool {
	budget := http.MaxInflightBodyBytes()
	if budget < 0 || n <= http.MaxRequestBytes() {
		return true
	}
	return n <= budget
}

// tightestBodyLimit is the fail-closed cap when the method is not known yet.
// It is the minimum of the content-type default and every candidate route cap,
// so the header-time limit is never above the route Dispatch may select.
func (app *Application) tightestBodyLimit(path string, fallback int64, methods ...string) int64 {
	chosen := fallback
	if app == nil || app.router == nil {
		return chosen
	}
	for _, method := range methods {
		n, set := app.router.BodyLimitFor(method, path)
		capn := fallback
		if set {
			capn = routeBodyCap(n)
		}
		if capn < chosen {
			chosen = capn
		}
	}
	return chosen
}

func isBodyLimitSafeMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "GET", "HEAD":
		return true
	default:
		return false
	}
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
