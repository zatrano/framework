package routing

import "sync"

// RouteGroup names a self-registered route set ("web" or "api").
type RouteGroup string

const (
	GroupWeb RouteGroup = "web"
	GroupAPI RouteGroup = "api"
)

var (
	discoveryMu sync.Mutex
	webFns      []func(*Router)
	apiFns      []func(*Router)
)

// RegisterWeb appends a web-group route registrar (typically from init()).
func RegisterWeb(fn func(*Router)) {
	if fn == nil {
		return
	}
	discoveryMu.Lock()
	webFns = append(webFns, fn)
	discoveryMu.Unlock()
}

// RegisterAPI appends an API-group route registrar (typically from init()).
func RegisterAPI(fn func(*Router)) {
	if fn == nil {
		return
	}
	discoveryMu.Lock()
	apiFns = append(apiFns, fn)
	discoveryMu.Unlock()
}

// ApplyWeb runs every registered web function against r.
func ApplyWeb(r *Router) {
	if r == nil {
		return
	}
	discoveryMu.Lock()
	fns := make([]func(*Router), len(webFns))
	copy(fns, webFns)
	discoveryMu.Unlock()
	for _, fn := range fns {
		fn(r)
	}
}

// ApplyAPI runs every registered API function against r.
func ApplyAPI(r *Router) {
	if r == nil {
		return
	}
	discoveryMu.Lock()
	fns := make([]func(*Router), len(apiFns))
	copy(fns, apiFns)
	discoveryMu.Unlock()
	for _, fn := range fns {
		fn(r)
	}
}
