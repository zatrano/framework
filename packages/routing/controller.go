package routing

// Controller groups route registrations for a single controller instance.
// Keep each controller's routes inside its own Controller() block so files stay organized.
func Controller[T any](r *Router, ctrl T, fn func(r *Router, c T)) {
	fn(r, ctrl)
}
