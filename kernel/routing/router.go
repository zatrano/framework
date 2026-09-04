package routing

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/zatrano/framework/kernel/http"
)

// HandlerFunc handles an HTTP request and returns a response.
type HandlerFunc func(req *http.Request) *http.Response

// MiddlewareFunc wraps a handler.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Route represents a registered route.
type Route struct {
	Method     string
	Path       string
	Name       string
	Handler    HandlerFunc
	Middleware []MiddlewareFunc
	paramNames []string
	pattern    *regexp.Regexp
	namePrefix string
}

// Router is the ZATRANO HTTP router.
type Router struct {
	routes          []*Route
	middleware      []MiddlewareFunc
	groupPrefix     string
	groupName       string
	groupMiddleware []MiddlewareFunc
	named           map[string]*Route
	fallback        HandlerFunc
}

// New creates a new router.
func New() *Router {
	return &Router{
		routes: make([]*Route, 0),
		named:  make(map[string]*Route),
	}
}

// Use appends global middleware.
func (r *Router) Use(middleware ...MiddlewareFunc) {
	r.middleware = append(r.middleware, middleware...)
}

// Group creates a route group with a shared prefix and middleware.
func (r *Router) Group(prefix string, fn func(router *Router), middleware ...MiddlewareFunc) {
	previousPrefix := r.groupPrefix
	previousMiddleware := r.groupMiddleware

	r.groupPrefix = joinPath(previousPrefix, prefix)
	r.groupMiddleware = append(append([]MiddlewareFunc{}, previousMiddleware...), middleware...)
	fn(r)

	r.groupPrefix = previousPrefix
	r.groupMiddleware = previousMiddleware
}

// Name sets a route name prefix for routes registered inside fn.
func (r *Router) Name(prefix string, fn func(router *Router)) {
	previous := r.groupName
	r.groupName = previous + prefix
	fn(r)
	r.groupName = previous
}

// Get registers a GET route.
func (r *Router) Get(path string, handler HandlerFunc) *Route {
	return r.Add("GET", path, handler)
}

// Post registers a POST route.
func (r *Router) Post(path string, handler HandlerFunc) *Route {
	return r.Add("POST", path, handler)
}

// Put registers a PUT route.
func (r *Router) Put(path string, handler HandlerFunc) *Route {
	return r.Add("PUT", path, handler)
}

// Patch registers a PATCH route.
func (r *Router) Patch(path string, handler HandlerFunc) *Route {
	return r.Add("PATCH", path, handler)
}

// Delete registers a DELETE route.
func (r *Router) Delete(path string, handler HandlerFunc) *Route {
	return r.Add("DELETE", path, handler)
}

// Options registers an OPTIONS route.
func (r *Router) Options(path string, handler HandlerFunc) *Route {
	return r.Add("OPTIONS", path, handler)
}

// Any registers a route for common HTTP methods.
func (r *Router) Any(path string, handler HandlerFunc) []*Route {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	routes := make([]*Route, 0, len(methods))
	for _, method := range methods {
		routes = append(routes, r.Add(method, path, handler))
	}
	return routes
}

// Match registers a route for the given methods.
func (r *Router) Match(methods []string, path string, handler HandlerFunc) []*Route {
	routes := make([]*Route, 0, len(methods))
	for _, method := range methods {
		routes = append(routes, r.Add(strings.ToUpper(method), path, handler))
	}
	return routes
}

// Redirect registers a GET route that redirects to another path.
func (r *Router) Redirect(from, to string, status ...int) *Route {
	code := 302
	if len(status) > 0 && status[0] > 0 {
		code = status[0]
	}
	return r.Get(from, func(req *http.Request) *http.Response {
		return http.Redirect(to, code)
	})
}

// Fallback sets a handler used when no route matches.
func (r *Router) Fallback(handler HandlerFunc) {
	r.fallback = handler
}

// Add registers a route.
func (r *Router) Add(method, path string, handler HandlerFunc) *Route {
	fullPath := joinPath(r.groupPrefix, path)
	paramNames, pattern := compilePath(fullPath)

	route := &Route{
		Method:     strings.ToUpper(method),
		Path:       fullPath,
		Handler:    handler,
		Middleware: append([]MiddlewareFunc{}, r.groupMiddleware...),
		paramNames: paramNames,
		pattern:    pattern,
		namePrefix: r.groupName,
	}
	r.routes = append(r.routes, route)
	return route
}

// As assigns a name to the route (with any active group name prefix).
func (route *Route) As(name string) *Route {
	route.Name = route.namePrefix + name
	return route
}

// Through assigns route-specific middleware.
func (route *Route) Through(middleware ...MiddlewareFunc) *Route {
	route.Middleware = append(route.Middleware, middleware...)
	return route
}

// RegisterName stores a named route on the router.
func (r *Router) RegisterName(route *Route) {
	if route.Name != "" {
		r.named[route.Name] = route
	}
}

// Routes returns all registered routes.
func (r *Router) Routes() []*Route {
	return r.routes
}

// Route finds a named route.
func (r *Router) Route(name string) (*Route, bool) {
	route, ok := r.named[name]
	return route, ok
}

// Dispatch finds a matching route and executes it.
func (r *Router) Dispatch(req *http.Request) *http.Response {
	normalizeDispatchPath(req)

	for _, route := range r.routes {
		if route.Method != req.Method() {
			continue
		}

		matches := route.pattern.FindStringSubmatch(req.Path())
		if matches == nil {
			continue
		}

		params := make(map[string]string, len(route.paramNames))
		for i, name := range route.paramNames {
			params[name] = matches[i+1]
		}
		req.SetRouteParams(params)
		req.SetRouteName(route.Name)

		handler := route.Handler
		stack := append(append([]MiddlewareFunc{}, r.middleware...), route.Middleware...)
		for i := len(stack) - 1; i >= 0; i-- {
			handler = stack[i](handler)
		}
		return handler(req)
	}

	if r.fallback != nil {
		handler := r.fallback
		stack := append([]MiddlewareFunc{}, r.middleware...)
		for i := len(stack) - 1; i >= 0; i-- {
			handler = stack[i](handler)
		}
		return handler(req)
	}

	return http.Abort(404, "Not Found")
}

// RedirectRoute redirects to a named route.
func (r *Router) RedirectRoute(name string, params map[string]string, status ...int) *http.Response {
	path, err := r.URL(name, params)
	if err != nil {
		return http.Abort(500, err.Error())
	}
	return http.Redirect(path, status...)
}

// URL generates a URL for a named route.
func (r *Router) URL(name string, params ...map[string]string) (string, error) {
	route, ok := r.named[name]
	if !ok {
		return "", fmt.Errorf("route [%s] not defined", name)
	}

	path := route.Path
	if len(params) > 0 {
		for key, value := range params[0] {
			path = strings.ReplaceAll(path, "{"+key+"}", value)
			path = strings.ReplaceAll(path, "{"+key+"?}", value)
			path = strings.ReplaceAll(path, "{*"+key+"}", value)
			path = strings.ReplaceAll(path, "{"+key+"*}", value)
		}
	}

	// Remove unused optional params.
	re := regexp.MustCompile(`\{[^}]+\?\}`)
	path = re.ReplaceAllString(path, "")
	path = strings.ReplaceAll(path, "//", "/")
	if path == "" {
		path = "/"
	}
	return path, nil
}

func joinPath(prefix, path string) string {
	if prefix == "" {
		if path == "" {
			return "/"
		}
		if !strings.HasPrefix(path, "/") {
			return "/" + path
		}
		return path
	}

	prefix = strings.TrimSuffix(prefix, "/")
	if path == "" || path == "/" {
		return prefix
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return prefix + path
}

func compilePath(path string) ([]string, *regexp.Regexp) {
	if path == "" {
		path = "/"
	}

	var names []string
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
			optional := strings.HasSuffix(name, "?")
			name = strings.TrimSuffix(name, "?")
			catchAll := strings.HasPrefix(name, "*") || strings.HasSuffix(name, "*")
			name = strings.TrimPrefix(name, "*")
			name = strings.TrimSuffix(name, "*")
			names = append(names, name)
			fragment := `[^/]+`
			if optional {
				fragment = `[^/]*`
			}
			if catchAll {
				// {*slug} / {slug*} matches the rest of the path, including slashes.
				fragment = `.+`
			}
			parts[i] = `(` + fragment + `)`
		} else {
			parts[i] = regexp.QuoteMeta(part)
		}
	}

	pattern := "^" + strings.Join(parts, "/") + "$"
	if path == "/" {
		pattern = "^/$"
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		// Hostile / invalid UTF-8 paths must not panic route registration.
		return names, regexp.MustCompile(`^\b\B$`)
	}
	return names, re
}

// normalizeDispatchPath strips a trailing slash from the request path (except "/")
// so /dashboard and /dashboard/ match the same route. Query string is untouched.
func normalizeDispatchPath(req *http.Request) {
	if req == nil || req.Raw() == nil || req.Raw().URL == nil {
		return
	}
	path := req.Raw().URL.Path
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		req.Raw().URL.Path = strings.TrimRight(path, "/")
	}
}
