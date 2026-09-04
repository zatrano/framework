package contracts

// Route is a registered HTTP route (name assignment only on this surface).
type Route interface {
	As(name string) Route
}

// RouteSnapshot is serializable route metadata (handlers are not included).
type RouteSnapshot struct {
	Method string
	Path   string
	Name   string
}

// Router is the HTTP router surface used via Application.Router().
// Handler and middleware values are untyped here so this package does not
// import packages/http or packages/routing.
type Router interface {
	Get(path string, handler any) Route
	Post(path string, handler any) Route
	Use(middleware ...any)
	Group(prefix string, fn func(router Router), middleware ...any)
	Name(prefix string, fn func(router Router))
	Snapshot() []RouteSnapshot
	SaveCache(path string) error
}
