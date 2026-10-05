package bench

import (
	"strings"
	"time"

	"github.com/zatrano/rawhttp"
)

// Server profiles. Production Run is not refactored; these literals mirror it.
//
// v3.0.1 Run (application.go at e4c0969) sets Handler, the four timeouts
// (10s/60s/60s/120s), MaxHeaderBytes 1 MiB, TrustedProxies from the
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
	runMaxHeaderBytes    = 1 << 20
	runMaxBodyBytes      = 32 << 20
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
		MaxHeaderBytes:    runMaxHeaderBytes,
		KeepHijackedConns: true,
	}
}

// serverRunHead matches Application.Run on perf/v3 HEAD with the same unset
// environment. HeaderReceived is a harness copy of the GET/HEAD fast path
// (ceiling, no route lookup). The production method is unexported, and this
// module does not change the framework.
func serverRunHead(h rawhttp.Handler) *rawhttp.Server {
	s := serverRunV301(h)
	s.MaxRequestBodySize = runMaxBodyBytes
	s.AllowUpgrade = false
	s.HeaderReceived = headFastPath
	return s
}

func headFastPath(ctx *rawhttp.Ctx) rawhttp.RequestConfig {
	if ctx == nil {
		return rawhttp.RequestConfig{}
	}
	// GET and HEAD return the content-type ceiling without matching a route.
	// A missing Content-Type uses the server ceiling (32 MiB), same as
	// HeaderBodyLimit. Other methods on the plaintext bench are not used.
	method := strings.ToUpper(string(ctx.Method))
	if method == "GET" || method == "HEAD" || method == "POST" || method == "OPTIONS" {
		return rawhttp.RequestConfig{MaxRequestBodySize: runMaxBodyBytes}
	}
	return rawhttp.RequestConfig{MaxRequestBodySize: runMaxBodyBytes}
}
