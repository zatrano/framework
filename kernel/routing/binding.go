package routing

import (
	"sync"

	"github.com/zatrano/framework/v2/kernel/http"
)

// Binder resolves a route parameter into a request attribute.
type Binder func(value string, req *http.Request) (any, error)

var (
	bindersMu sync.RWMutex
	binders   = map[string]Binder{}
)

// Bind registers a model binder for a route parameter name.
func Bind(param string, binder Binder) {
	bindersMu.Lock()
	defer bindersMu.Unlock()
	binders[param] = binder
}

// HasBinding reports whether a binder exists for param.
func HasBinding(param string) bool {
	bindersMu.RLock()
	defer bindersMu.RUnlock()
	_, ok := binders[param]
	return ok
}

// ClearBindings removes all registered binders (mainly for tests).
func ClearBindings() {
	bindersMu.Lock()
	defer bindersMu.Unlock()
	binders = map[string]Binder{}
}

func lookupBinder(param string) (Binder, bool) {
	bindersMu.RLock()
	defer bindersMu.RUnlock()
	fn, ok := binders[param]
	return fn, ok
}

// SubstituteBindings resolves registered binders into request attributes.
// Missing or failed bindings return a 404 JSON response.
func SubstituteBindings() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(req *http.Request) *http.Response {
			for param, value := range req.RouteParams() {
				binder, ok := lookupBinder(param)
				if !ok {
					continue
				}
				model, err := binder(value, req)
				if err != nil || model == nil {
					return http.JSON(map[string]any{"message": "Not Found"}).Status(404)
				}
				req.Set(param, model)
			}
			return next(req)
		}
	}
}
