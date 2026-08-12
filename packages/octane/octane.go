package octane

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Runtime tracks concurrent request metrics and worker hints.
//
// This is not a multi-process application server. Metrics middleware is real
// (request counts / in-flight / peak). The octane:start command sets GOMAXPROCS
// from the worker hint and serves the normal HTTP application via Application.Run.
type Runtime struct {
	workers   int
	startedAt time.Time
	requests  atomic.Int64
	inFlight  atomic.Int64
	peak      atomic.Int64
}

// New creates a runtime with the given worker hint (GOMAXPROCS guidance).
func New(workers int) *Runtime {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &Runtime{
		workers:   workers,
		startedAt: time.Now().UTC(),
	}
}

// Workers returns the configured worker count hint.
func (r *Runtime) Workers() int {
	return r.workers
}

// SetWorkers updates the worker hint used by octane:start for GOMAXPROCS.
func (r *Runtime) SetWorkers(n int) {
	if n > 0 {
		r.workers = n
	}
}

// Stats returns runtime statistics.
func (r *Runtime) Stats() map[string]any {
	return map[string]any{
		"workers":    r.workers,
		"gomaxprocs": runtime.GOMAXPROCS(0),
		"num_cpu":    runtime.NumCPU(),
		"requests":   r.requests.Load(),
		"in_flight":  r.inFlight.Load(),
		"peak":       r.peak.Load(),
		"uptime_sec": int(time.Since(r.startedAt).Seconds()),
		"started_at": r.startedAt.Format(time.RFC3339),
	}
}

// Middleware tracks in-flight and total requests (real metrics).
func (r *Runtime) Middleware() routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			r.requests.Add(1)
			cur := r.inFlight.Add(1)
			for {
				peak := r.peak.Load()
				if cur <= peak || r.peak.CompareAndSwap(peak, cur) {
					break
				}
			}
			defer r.inFlight.Add(-1)
			return next(req)
		}
	}
}

// Handler returns JSON runtime stats.
func (r *Runtime) Handler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		return http.JSON(r.Stats())
	}
}
