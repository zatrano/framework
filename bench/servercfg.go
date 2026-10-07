package bench

import (
	"net"
	"strings"
	"time"

	khttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

// Server profiles. Production Run is not refactored; these literals mirror it.
//
// v3.0.1 Run (application.go at e4c0969) sets Handler, the four timeouts
// (10s/60s/60s/120s), MaxHeaderBytes 1 MiB (v3.1.0 uses 16 KiB), TrustedProxies from the
// environment, and KeepHijackedConns. ConnState, Concurrency, buffer sizes,
// MaxRequestBodySize, HeaderReceived, and AllowUpgrade stay at zero.
//
// perf/v3 HEAD Run also sets MaxRequestBodySize from the body ceiling,
// HeaderReceived to the unexported body-limit hook, and AllowUpgrade (false
// unless upgrade is configured). ConnState and Concurrency are still unset.
//
// The old harness set Read/Write/Idle to -1 (unlimited) and left every other
// field at zero, so rawhttp skipped per-request deadlines.

const (
	runReadHeaderTimeout = 10 * time.Second
	runReadTimeout       = 60 * time.Second
	runWriteTimeout      = 60 * time.Second
	runIdleTimeout       = 120 * time.Second
	// headerBytesV301 is the v3.0.0–v3.0.1 production header ceiling.
	// rawhttp allocates the connection read buffer from it.
	headerBytesV301 = 1 << 20
	// runMaxHeaderBytes is the v3.1.0 production default (16 KiB).
	runMaxHeaderBytes = 16 << 10
	runMaxBodyBytes   = 32 << 20
)

func serverOld(h rawhttp.Handler) *rawhttp.Server {
	return &rawhttp.Server{
		Handler:      h,
		ReadTimeout:  -1,
		WriteTimeout: -1,
		IdleTimeout:  -1,
	}
}

// serverRunV301 matches Application.Run at v3.0.1 when HTTP_* timeouts and
// TRUSTED_PROXIES are unset.
func serverRunV301(h rawhttp.Handler) *rawhttp.Server {
	return &rawhttp.Server{
		Handler:           h,
		ReadHeaderTimeout: runReadHeaderTimeout,
		ReadTimeout:       runReadTimeout,
		WriteTimeout:      runWriteTimeout,
		IdleTimeout:       runIdleTimeout,
		MaxHeaderBytes:    headerBytesV301,
		KeepHijackedConns: true,
	}
}

// serverRunHead matches Application.Run on perf/v3 HEAD with the same unset
// environment. HeaderReceived is the production hook (boot snapshot, bodyless
// fast path). headFastPath remains the diagnostic copy used by AblateHook.
func serverRunHead(h rawhttp.Handler) *rawhttp.Server {
	s := serverRunV301(h)
	s.MaxHeaderBytes = runMaxHeaderBytes
	s.MaxRequestBodySize = runMaxBodyBytes
	s.AllowUpgrade = false
	s.HeaderReceived = canonicalHook
	s.ConnState = func(net.Conn, rawhttp.ConnState) {}
	return s
}

func headFastPath(ctx *rawhttp.Ctx) rawhttp.RequestConfig {
	return ceilingFromCtx(ctx, nil)
}

// ceilingFromCtx mirrors Application.decideBodyLimit for the header hook.
// routeLimits is "METHOD /path" → BodyLimit. Nil means no route cap, which is
// the plaintext benchmark. GET and HEAD never consult it. A positive cap below
// the content-type default forces a lookup on other methods, including a small
// body. The framework golden test checks these numbers against the real hook.
func ceilingFromCtx(ctx *rawhttp.Ctx, routeLimits map[string]int64) rawhttp.RequestConfig {
	if ctx == nil {
		return rawhttp.RequestConfig{}
	}
	return ceiling(string(ctx.Method), string(ctx.Path), string(ctx.Header("Content-Type")), ctx.ContentLength(), routeLimits)
}

func ceiling(method, path, contentType string, contentLength int, routeLimits map[string]int64) rawhttp.RequestConfig {
	limit := khttp.HeaderBodyLimit(contentType)
	if isSafeBodyMethod(method) {
		return khttp.BodyLimitConfig(limit)
	}
	tighter := false
	for _, n := range routeLimits {
		if n > 0 && n < limit {
			tighter = true
			break
		}
	}
	needs := contentLength < 0 || int64(contentLength) > limit
	chosen := limit
	if needs || tighter {
		key := strings.ToUpper(strings.TrimSpace(method)) + " " + normalizeBenchPath(path)
		if n, ok := routeLimits[key]; ok && n > 0 {
			chosen = n
			if chosen > khttp.MaxRequestBytes() && chosen > khttp.MaxInflightBodyBytes() {
				return khttp.BodyLimitConfig(1)
			}
		}
	}
	return khttp.BodyLimitConfig(chosen)
}

func isSafeBodyMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "HEAD":
		return true
	default:
		return false
	}
}

func normalizeBenchPath(path string) string {
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		return strings.TrimRight(path, "/")
	}
	return path
}
