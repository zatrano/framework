package kernel

import (
	"strings"
	"sync/atomic"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

// bodyLimitLookupHook counts route lookups in tests. Nil in production.
// It runs only on the slow path (a body that can exceed the default).
var bodyLimitLookupHook func()

// bodyLimitSnap is the header-hook ceiling, read once at boot.
// Request.Body and Request.JSON do not use it; they still call the env helpers.
type bodyLimitSnap struct {
	maxRequest int64
	maxBody    int64
	inflight   int64
}

func (app *Application) ensureBodyLimits() bodyLimitSnap {
	if app == nil {
		return bodyLimitSnap{
			maxRequest: http.MaxRequestBytes(),
			maxBody:    http.MaxBodyBytes(),
			inflight:   http.MaxInflightBodyBytes(),
		}
	}
	if atomic.LoadUint32(&app.bodyLimitReady) == 1 {
		return app.bodyLimitSnap
	}
	app.bodyLimitMu.Lock()
	defer app.bodyLimitMu.Unlock()
	if atomic.LoadUint32(&app.bodyLimitReady) == 1 {
		return app.bodyLimitSnap
	}
	app.bodyLimitSnap = bodyLimitSnap{
		maxRequest: http.MaxRequestBytes(),
		maxBody:    http.MaxBodyBytes(),
		inflight:   http.MaxInflightBodyBytes(),
	}
	atomic.StoreUint32(&app.bodyLimitReady, 1)
	return app.bodyLimitSnap
}

// HeaderBodyConfig is the production HeaderReceived hook.
// A bodyless request (no Content-Length, or Content-Length 0, and no
// Transfer-Encoding) returns the carrier default and allocates nothing.
// Limits come from the boot snapshot, not from env.Get.
func (app *Application) HeaderBodyConfig(ctx *rawhttp.Ctx) rawhttp.RequestConfig {
	if app == nil || ctx == nil || !requestHasBody(ctx) {
		return rawhttp.RequestConfig{}
	}
	var override []byte
	if methodIs(ctx.Method, "POST") {
		override = ctx.Header("X-HTTP-Method-Override")
	}
	return app.limitRequest(ctx, ctx.ContentLength(), ctx.Method, ctx.Path, ctx.Header("Content-Type"), override)
}

// headerBodyConfig is the hook installed on the production server.
func (app *Application) headerBodyConfig(ctx *rawhttp.Ctx) rawhttp.RequestConfig {
	return app.HeaderBodyConfig(ctx)
}

// requestHasBody is true when the carrier will read a body.
// ContentLength is -1 for chunked and 0 when Content-Length is absent or zero.
// rawhttp rejects every other Transfer-Encoding before HeaderReceived, so a
// zero length here is a bodyless request. Scanning the header block for
// Transfer-Encoding would allocate nothing but costs more than the bodyless budget.
func requestHasBody(ctx *rawhttp.Ctx) bool {
	return ctx.ContentLength() != 0
}

// decideBodyLimit is the header-time cap. method/path go through the same
// resolver as Dispatch. An override that is not known until the body is read,
// or more than one candidate route, keeps the tightest cap and never raises it.
func (app *Application) decideBodyLimit(method, path, contentType, override string, contentLength int) rawhttp.RequestConfig {
	if app == nil {
		return rawhttp.RequestConfig{}
	}
	return app.limitRequest(nil, contentLength, []byte(method), []byte(path), []byte(contentType), []byte(override))
}

func (app *Application) limitRequest(ctx *rawhttp.Ctx, contentLength int, method, path, contentType, override []byte) rawhttp.RequestConfig {
	lim := app.ensureBodyLimits()
	limit := lim.headerLimit(contentType)
	resolved, ambiguous := resolveOverride(method, override, contentType)
	if resolved == "GET" || resolved == "HEAD" {
		return app.grantBody(ctx, lim, limit, contentLength, contentType)
	}
	tighter := app != nil && app.router != nil && app.router.HasTighterBodyLimit(limit)
	if !bodyNeedsRoute(resolved, contentLength, limit) && !ambiguous && !tighter {
		return app.grantBody(ctx, lim, limit, contentLength, contentType)
	}
	if bodyLimitLookupHook != nil {
		bodyLimitLookupHook()
	}
	chosen := limit
	if ambiguous {
		chosen = app.tightestBodyLimit(string(path), limit, "POST", "PUT", "PATCH", "DELETE")
	} else if app != nil && app.router != nil {
		if n, set := app.router.BodyLimitFor(resolved, string(path)); set {
			chosen = lim.routeCap(n)
		}
	}
	return app.grantBody(ctx, lim, chosen, contentLength, contentType)
}

func (lim bodyLimitSnap) headerLimit(contentType []byte) int64 {
	if earlyBodyMediaBytes(contentType) {
		return lim.maxBody
	}
	return lim.maxRequest
}

func (lim bodyLimitSnap) routeCap(n int64) int64 {
	if n < 0 {
		return lim.maxRequest
	}
	return n
}

func (lim bodyLimitSnap) sustainable(n int64) bool {
	if lim.inflight < 0 {
		return true
	}
	return n <= lim.inflight
}

func routeBodyCap(n int64) int64 {
	if n < 0 {
		return http.MaxRequestBytes()
	}
	return n
}

// tightestBodyLimit is the fail-closed cap when the method is not known yet.
func (app *Application) tightestBodyLimit(path string, fallback int64, methods ...string) int64 {
	chosen := fallback
	if app == nil || app.router == nil {
		return chosen
	}
	lim := app.ensureBodyLimits()
	for _, method := range methods {
		n, set := app.router.BodyLimitFor(method, path)
		capn := fallback
		if set {
			capn = lim.routeCap(n)
		}
		if capn < chosen {
			chosen = capn
		}
	}
	return chosen
}

func bodyNeedsRoute(method string, contentLength int, limit int64) bool {
	if method == "GET" || method == "HEAD" {
		return false
	}
	if contentLength < 0 {
		return true
	}
	return int64(contentLength) > limit
}

func resolveOverride(method, override, contentType []byte) (resolved string, ambiguous bool) {
	resolved = methodToken(method)
	if resolved != "POST" {
		return resolved, false
	}
	override = trimSpaceBytes(override)
	if len(override) > 0 {
		if token := methodToken(override); token == "PUT" || token == "PATCH" || token == "DELETE" {
			return token, false
		}
		return "POST", false
	}
	if mediaIs(contentType, "application/x-www-form-urlencoded") {
		return "POST", true
	}
	return "POST", false
}

func methodToken(method []byte) string {
	switch len(method) {
	case 3:
		if methodIs(method, "GET") {
			return "GET"
		}
		if methodIs(method, "PUT") {
			return "PUT"
		}
	case 4:
		if methodIs(method, "POST") {
			return "POST"
		}
		if methodIs(method, "HEAD") {
			return "HEAD"
		}
	case 5:
		if methodIs(method, "PATCH") {
			return "PATCH"
		}
	case 6:
		if methodIs(method, "DELETE") {
			return "DELETE"
		}
	case 7:
		if methodIs(method, "OPTIONS") {
			return "OPTIONS"
		}
	}
	return strings.ToUpper(strings.TrimSpace(string(method)))
}

func methodIs(method []byte, upper string) bool {
	if len(method) != len(upper) {
		return false
	}
	for i := 0; i < len(method); i++ {
		c := method[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		if c != upper[i] {
			return false
		}
	}
	return true
}

func earlyBodyMediaBytes(contentType []byte) bool {
	media := mediaBytes(contentType)
	if len(media) == 0 {
		return false
	}
	if mediaIsRaw(media, "application/json") || mediaIsRaw(media, "application/x-www-form-urlencoded") {
		return true
	}
	if hasSuffixFold(media, "+json") {
		return true
	}
	return hasPrefixFold(media, "text/")
}

func mediaIs(contentType []byte, want string) bool {
	return mediaIsRaw(mediaBytes(contentType), want)
}

func mediaBytes(contentType []byte) []byte {
	ct := trimSpaceBytes(contentType)
	if len(ct) == 0 {
		return nil
	}
	for i := 0; i < len(ct); i++ {
		if ct[i] == ';' {
			return trimSpaceBytes(ct[:i])
		}
	}
	return ct
}

func mediaIsRaw(media []byte, want string) bool {
	if len(media) != len(want) {
		return false
	}
	for i := 0; i < len(media); i++ {
		c := media[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != want[i] {
			return false
		}
	}
	return true
}

func hasPrefixFold(media []byte, prefix string) bool {
	if len(media) < len(prefix) {
		return false
	}
	return mediaIsRaw(media[:len(prefix)], prefix)
}

func hasSuffixFold(media []byte, suffix string) bool {
	if len(media) < len(suffix) {
		return false
	}
	return mediaIsRaw(media[len(media)-len(suffix):], suffix)
}

func trimSpaceBytes(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}
	return b
}
