package routing

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/http"
)

// HandlerFunc handles an HTTP request and returns a response.
type HandlerFunc func(req *http.Request) *http.Response

// MiddlewareFunc wraps a handler.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Route represents a registered route.
type Route struct {
	Method             string
	Path               string
	Name               string
	Handler            HandlerFunc
	Middleware         []MiddlewareFunc
	paramNames         []string
	pattern            *regexp.Regexp
	namePrefix         string
	router             *Router
	frozenName         string
	frozenPath         string
	frozenMethod       string
	frozenHandler      HandlerFunc
	frozenMW           []MiddlewareFunc
	frozenChain        HandlerFunc
	bodyLimit          int64
	bodyLimitSet       bool
	frozenBodyLimit    int64
	frozenBodyLimitSet bool
}

type methodTable struct {
	static map[string]*Route
	tree   *trieNode
}

// Router is the ZATRANO HTTP router.
type Router struct {
	routes              []*Route
	middleware          []MiddlewareFunc
	groupPrefix         string
	groupName           string
	groupMiddleware     []MiddlewareFunc
	named               map[string]*Route
	fallback            HandlerFunc
	frozen              bool
	frozenGlobalMW      []MiddlewareFunc
	frozenFallback      HandlerFunc
	frozenFallbackChain HandlerFunc
	frozenNotFound      HandlerFunc
	byMethod            map[string]*methodTable
	minBodyLimit        int64 // smallest positive BodyLimit; 0 if none
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
	r.frozenGlobalMW = append([]MiddlewareFunc{}, r.middleware...)
	r.frozenFallback = r.fallback
	if r.frozenFallback != nil {
		r.frozenFallbackChain = composeHandler(r.frozenFallback, r.frozenGlobalMW)
	} else {
		r.frozenNotFound = composeHandler(routerNotFound, r.frozenGlobalMW)
	}
	for _, route := range r.routes {
		if route == nil {
			continue
		}
		route.frozenName = route.Name
		route.frozenPath = route.Path
		route.frozenMethod = route.Method
		route.frozenHandler = route.Handler
		route.frozenMW = append([]MiddlewareFunc{}, route.Middleware...)
		stack := append(append([]MiddlewareFunc{}, r.frozenGlobalMW...), route.frozenMW...)
		route.frozenChain = composeHandler(route.frozenHandler, stack)
		route.frozenBodyLimit = route.bodyLimit
		route.frozenBodyLimitSet = route.bodyLimitSet
	}
	r.frozen = true
	return nil
}

func composeHandler(handler HandlerFunc, stack []MiddlewareFunc) HandlerFunc {
	if handler == nil {
		return nil
	}
	for i := len(stack) - 1; i >= 0; i-- {
		handler = stack[i](handler)
	}
	return handler
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

// BodyLimit sets the header-time body cap for this route, in bytes.
// A negative value uses the server ceiling. A positive value is passed to
// rawhttp as RequestConfig.MaxRequestBodySize and may exceed the server
// ceiling. The router is consulted only for requests that can exceed the
// content-type default (Content-Length above that default, or a chunked body).
// GET and HEAD never consult it. The match is lookupRoute, the same resolver
// Dispatch uses. An unknown method override or several candidate routes keeps
// the tightest cap. A cap above both the server ceiling and the in-flight
// budget is not granted.
// Callers use routing.From(app) so the concrete *Route is returned.
func (route *Route) BodyLimit(n int64) *Route {
	if route == nil {
		return nil
	}
	if route.router != nil {
		route.router.mutate()
	}
	route.bodyLimit = n
	route.bodyLimitSet = true
	if n > 0 {
		if budget := http.MaxInflightBodyBytes(); budget >= 0 && n > budget {
			log.Printf("[WARN] route body limit %d exceeds HTTP_MAX_INFLIGHT_BODY_BYTES %d; a request at that cap is rejected with 413", n, budget)
		}
	}
	if route.router != nil && n > 0 && (route.router.minBodyLimit == 0 || n < route.router.minBodyLimit) {
		route.router.minBodyLimit = n
	}
	return route
}

// BodyLimitsAbove lists routes whose BodyLimit is above budget.
// A negative budget means the budget is off and the list is empty.
func (r *Router) BodyLimitsAbove(budget int64) []string {
	if r == nil || budget < 0 {
		return nil
	}
	var out []string
	for _, route := range r.routes {
		if route == nil || !route.bodyLimitSet || route.bodyLimit <= budget {
			continue
		}
		out = append(out, fmt.Sprintf("%s %s = %d", route.Method, route.Path, route.bodyLimit))
	}
	return out
}

// HasTighterBodyLimit reports whether some route cap is below the content-type
// default. Header-time lookup must run in that case so a small body is not
// granted more than the route Dispatch will use.
func (r *Router) HasTighterBodyLimit(than int64) bool {
	return r != nil && r.minBodyLimit > 0 && r.minBodyLimit < than
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

func (route *Route) dispatchName() string {
	if route == nil {
		return ""
	}
	if route.router != nil && route.router.frozen {
		return route.frozenName
	}
	return route.Name
}

func (route *Route) dispatchPath() string {
	if route == nil {
		return ""
	}
	if route.router != nil && route.router.frozen {
		return route.frozenPath
	}
	return route.Path
}

func (route *Route) dispatchMethod() string {
	if route == nil {
		return ""
	}
	if route.router != nil && route.router.frozen {
		return route.frozenMethod
	}
	return route.Method
}

func (route *Route) dispatchHandler() HandlerFunc {
	if route == nil {
		return nil
	}
	if route.router != nil && route.router.frozen {
		return route.frozenHandler
	}
	return route.Handler
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
	fallback := r.fallback
	if r.frozen {
		fallback = r.frozenFallback
	}
	if fallback != nil {
		if r.frozen && r.frozenFallbackChain != nil {
			return r.frozenFallbackChain(req)
		}
		mw := r.middleware
		if r.frozen {
			mw = r.frozenGlobalMW
		}
		return r.invokeHandler(req, fallback, mw)
	}
	if r.frozen && r.frozenNotFound != nil {
		return r.frozenNotFound(req)
	}
	return r.invokeHandler(req, routerNotFound, r.middleware)
}

func routerNotFound(*http.Request) *http.Response {
	return http.Abort(404, "Not Found")
}

// BodyLimitFor reports the route cap for method+path.
// set is false when no route matches or the route did not call BodyLimit.
// Path and method rules are lookupRoute, the same resolver Dispatch uses.
func (r *Router) BodyLimitFor(method, path string) (limit int64, set bool) {
	route, _ := r.lookupRoute(method, path)
	if route == nil {
		return 0, false
	}
	if r.frozen {
		return route.frozenBodyLimit, route.frozenBodyLimitSet
	}
	return route.bodyLimit, route.bodyLimitSet
}

// lookupRoute is the single method+path resolver. Dispatch (via match) and
// header-time BodyLimit both call it. The path loses a trailing slash except
// for "/". Case, percent-encoding, "." and ".." segments, and extra slashes
// are left as the carrier delivered them. params is non-nil only when a
// frozen pattern route matched.
func (r *Router) lookupRoute(method, path string) (*Route, map[string]string) {
	hit := r.resolve(method, path, true)
	return hit.route, hit.params
}

// routeHit is one resolver result. subs is the unfrozen regexp capture,
// including the full match at index 0, so bind does not run the pattern again.
type routeHit struct {
	route  *Route
	params map[string]string
	subs   []string
}

// resolve matches method+path. normalize folds a trailing slash once.
// Callers that already folded the path pass normalize false.
func (r *Router) resolve(method, path string, normalize bool) routeHit {
	if r == nil {
		return routeHit{}
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if normalize {
		path = normalizeRoutePath(path)
	}
	if r.byMethod != nil {
		t := r.byMethod[method]
		if t == nil {
			return routeHit{}
		}
		if route := t.static[path]; route != nil {
			return routeHit{route: route}
		}
		if t.tree == nil {
			return routeHit{}
		}
		params := map[string]string{}
		if route := t.tree.lookup(requestSegs(path), params); route != nil {
			return routeHit{route: route, params: params}
		}
		return routeHit{}
	}
	for _, route := range r.routes {
		if route == nil || route.Method != method || route.pattern == nil {
			continue
		}
		subs := route.pattern.FindStringSubmatch(path)
		if subs == nil {
			continue
		}
		return routeHit{route: route, subs: subs}
	}
	return routeHit{}
}

func normalizeRoutePath(path string) string {
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		return strings.TrimRight(path, "/")
	}
	return path
}

func (r *Router) match(req *http.Request) *Route {
	if req == nil {
		return nil
	}
	path := req.Path()
	method := req.Method()
	// Frozen exact static hit. Dispatch already stripped a trailing slash, and
	// registration stores the method uppercase, so GET /plaintext does not
	// normalize or call lookupRoute. A miss falls through and folds the path
	// once.
	if r.byMethod != nil {
		if t := r.byMethod[method]; t != nil {
			if route := t.static[path]; route != nil {
				req.SetRouteParams(nil)
				req.SetRouteName(route.dispatchName())
				return route
			}
		}
	}
	hit := r.resolve(method, normalizeRoutePath(path), false)
	if hit.route == nil {
		return nil
	}
	if r.byMethod == nil {
		if !r.bindSubs(req, hit.route, hit.subs) {
			return nil
		}
		return hit.route
	}
	req.SetRouteParams(hit.params)
	req.SetRouteName(hit.route.dispatchName())
	return hit.route
}

// bindSubs fills route params from a match resolve already computed.
func (r *Router) bindSubs(req *http.Request, route *Route, matches []string) bool {
	if len(matches) == 0 {
		return false
	}
	params := make(map[string]string, len(route.paramNames))
	for i, name := range route.paramNames {
		params[name] = matches[i+1]
	}
	req.SetRouteParams(params)
	req.SetRouteName(route.dispatchName())
	return true
}

func (r *Router) invoke(req *http.Request, route *Route) *http.Response {
	if r.frozen && route != nil && route.frozenChain != nil {
		return route.frozenChain(req)
	}
	global := r.middleware
	local := route.Middleware
	stack := append(append([]MiddlewareFunc{}, global...), local...)
	return r.invokeHandler(req, route.dispatchHandler(), stack)
}

// Through runs handler through global middleware without route matching.
func (r *Router) Through(req *http.Request, handler HandlerFunc) *http.Response {
	if r == nil {
		return handler(req)
	}
	mw := r.middleware
	if r.frozen {
		mw = r.frozenGlobalMW
	}
	return r.invokeHandler(req, handler, mw)
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

	path := route.dispatchPath()
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
	if req == nil {
		return
	}
	path := req.Path()
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		req.SetPath(strings.TrimRight(path, "/"))
	}
}
