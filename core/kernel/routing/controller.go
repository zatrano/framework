package routing

// RouteRegistrar is the HTTP verb surface Controller() needs. *Router implements it.
type RouteRegistrar interface {
	Get(path string, handler HandlerFunc) *Route
	Post(path string, handler HandlerFunc) *Route
}

// Controller groups route registrations for a single controller instance.
// Keep each controller's routes inside its own Controller() block so files stay organized.
func Controller[T any](r RouteRegistrar, ctrl T, fn func(r RouteRegistrar, c T)) {
	fn(r, ctrl)
}
