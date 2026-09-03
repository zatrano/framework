package contracts

import "github.com/zatrano/framework/packages/routing"

// Router is the HTTP router surface used via Application.Router().
type Router interface {
	Get(path string, handler routing.HandlerFunc) *routing.Route
	Post(path string, handler routing.HandlerFunc) *routing.Route
	Use(middleware ...routing.MiddlewareFunc)
	Group(prefix string, fn func(router *routing.Router), middleware ...routing.MiddlewareFunc)
	Name(prefix string, fn func(router *routing.Router))
	Routes() []*routing.Route
	Snapshot() []routing.RouteInfo
	SaveCache(path string) error
}
