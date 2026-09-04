package routing

import (
	"fmt"
	"net/url"
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
	router     *Router
}

type methodTable struct {
	static map[string]*Route
	tree   *trieNode
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
	frozen          bool
	byMethod        map[string]*methodTable
}

// New creates a new router.
func New() *Router {
	return &Router{
		routes: make([]*Route, 0),
		named:  make(map[string]*Route),
	}
}

func (r *Router) mutate() {
	if r != nil && r.frozen {
		panic("router: cannot modify routes after bootstrap")
	}
}

// Freeze compiles the lookup table and makes the routing table immutable.
// After freeze, exact static paths are preferred over parameterized routes.
// A second call is a no-op. Ambiguous or duplicate routes return an error.
func (r *Router) Freeze() error {
	if r == nil {
		return nil
	}
	if r.frozen {
		return nil
	}
	tables, err := r.compileTables()
	if err != nil {
		return err
	}
	named, err := r.compileNamed()
	if err != nil {
		return err
	}
	r.byMethod = tables
	r.named = named
	r.frozen = true
	return nil
}

func (r *Router) compileNamed() (map[string]*Route, error) {
	named := make(map[string]*Route)
	for _, route := range r.routes {
		if route == nil || route.Name == "" {
			continue
		}
		if _, exists := named[route.Name]; exists {
			return nil, fmt.Errorf("router: duplicate route name %s", route.Name)
		}
		named[route.Name] = route
	}
	return named, nil
}

func (r *Router) compileTables() (map[string]*methodTable, error) {
	tables := make(map[string]*methodTable)
	for _, route := range r.routes {
		t := tables[route.Method]
		if t == nil {
			t = &methodTable{static: make(map[string]*Route)}
			tables[route.Method] = t
		}
		if len(route.paramNames) == 0 {
			if _, exists := t.static[route.Path]; exists {
				return nil, fmt.Errorf("router: duplicate route %s %s", route.Method, route.Path)
			}
			t.static[route.Path] = route
			continue
		}
		if t.tree == nil {
			t.tree = newTrieNode()
		}
		if err := t.tree.insert(parseRouteSegs(route.Path), route); err != nil {
			return nil, err
		}
	}
	return tables, nil
}

// Frozen reports whether the router has been frozen.
func (r *Router) Frozen() bool {
	return r != nil && r.frozen
}

// Use appends global middleware.
func (r *Router) Use(middleware ...MiddlewareFunc) {
	r.mutate()
	r.middleware = append(r.middleware, middleware...)
}

// Group creates a route group with a shared prefix and middleware.
func (r *Router) Group(prefix string, fn func(router *Router), middleware ...MiddlewareFunc) {
	r.mutate()
	previousPrefix := r.groupPrefix
	previousMiddleware := r.groupMiddleware
	r.groupPrefix = joinPath(previousPrefix, prefix)
	r.groupMiddleware = append(append([]MiddlewareFunc{}, previousMiddleware...), middleware...)
	defer func() {
		r.groupPrefix = previousPrefix
		r.groupMiddleware = previousMiddleware
	}()
	fn(r)
}

// Name sets a route name prefix for routes registered inside fn.
func (r *Router) Name(prefix string, fn func(router *Router)) {
	r.mutate()
	previous := r.groupName
	r.groupName = previous + prefix
	defer func() {
		r.groupName = previous
	}()
	fn(r)
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
	r.mutate()
	r.fallback = handler
}

// Add registers a route.
func (r *Router) Add(method, path string, handler HandlerFunc) *Route {
	r.mutate()
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
		router:     r,
	}
	r.routes = append(r.routes, route)
	return route
}

// As assigns a name to the route (with any active group name prefix).
func (route *Route) As(name string) *Route {
	if route != nil && route.router != nil {
		route.router.mutate()
	}
	route.Name = route.namePrefix + name
	return route
}

// Through assigns route-specific middleware.
func (route *Route) Through(middleware ...MiddlewareFunc) *Route {
	if route != nil && route.router != nil {
		route.router.mutate()
	}
	route.Middleware = append(route.Middleware, middleware...)
	return route
}

// RegisterName stores a named route on the router.
func (r *Router) RegisterName(route *Route) {
	r.mutate()
	if route.Name != "" {
		r.named[route.Name] = route
	}
}

// Routes returns a copy of the registered route slice.
func (r *Router) Routes() []*Route {
	out := make([]*Route, len(r.routes))
	copy(out, r.routes)
	return out
}

// Route finds a named route.
func (r *Router) Route(name string) (*Route, bool) {
	route, ok := r.named[name]
	return route, ok
}

// Dispatch finds a matching route and executes it.
func (r *Router) Dispatch(req *http.Request) *http.Response {
	normalizeDispatchPath(req)
	if route := r.match(req); route != nil {
		return r.invoke(req, route)
	}
	if r.fallback != nil {
		return r.invokeHandler(req, r.fallback, r.middleware)
	}
	return http.Abort(404, "Not Found")
}

func (r *Router) match(req *http.Request) *Route {
	path := req.Path()
	method := req.Method()
	if r.byMethod != nil {
		if t := r.byMethod[method]; t != nil {
			if route := t.static[path]; route != nil {
				req.SetRouteParams(map[string]string{})
				req.SetRouteName(route.Name)
				return route
			}
			if t.tree != nil {
				params := map[string]string{}
				if route := t.tree.lookup(requestSegs(path), params); route != nil {
					req.SetRouteParams(params)
					req.SetRouteName(route.Name)
					return route
				}
			}
		}
		return nil
	}
	for _, route := range r.routes {
		if route.Method != method {
			continue
		}
		if r.bind(req, route, path) {
			return route
		}
	}
	return nil
}

func (r *Router) bind(req *http.Request, route *Route, path string) bool {
	matches := route.pattern.FindStringSubmatch(path)
	if matches == nil {
		return false
	}
	params := make(map[string]string, len(route.paramNames))
	for i, name := range route.paramNames {
		params[name] = matches[i+1]
	}
	req.SetRouteParams(params)
	req.SetRouteName(route.Name)
	return true
}

func (r *Router) invoke(req *http.Request, route *Route) *http.Response {
	stack := append(append([]MiddlewareFunc{}, r.middleware...), route.Middleware...)
	return r.invokeHandler(req, route.Handler, stack)
}

func (r *Router) invokeHandler(req *http.Request, handler HandlerFunc, stack []MiddlewareFunc) *http.Response {
	for i := len(stack) - 1; i >= 0; i-- {
		handler = stack[i](handler)
	}
	return handler(req)
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
			path = strings.ReplaceAll(path, "{*"+key+"}", escapePathValue(value, true))
			path = strings.ReplaceAll(path, "{"+key+"*}", escapePathValue(value, true))
			path = strings.ReplaceAll(path, "{"+key+"}", escapePathValue(value, false))
			path = strings.ReplaceAll(path, "{"+key+"?}", escapePathValue(value, false))
		}
	}

	// Remove unused optional params.
	re := regexp.MustCompile(`\{[^}]+\?\}`)
	path = re.ReplaceAllString(path, "")
	path = strings.ReplaceAll(path, "//", "/")
	if path == "" {
		path = "/"
	}
	if strings.Contains(path, "{") {
		return "", fmt.Errorf("route [%s] missing required parameter", name)
	}
	return path, nil
}

func escapePathValue(value string, catchAll bool) string {
	if !catchAll {
		return url.PathEscape(value)
	}
	parts := strings.Split(value, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
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
