package routing

import (
	"strings"

	"github.com/zatrano/framework/v2/kernel/support/str"
)

// Resource holds optional REST resource handlers.
type Resource struct {
	Index   HandlerFunc
	Create  HandlerFunc
	Store   HandlerFunc
	Show    HandlerFunc
	Edit    HandlerFunc
	Update  HandlerFunc
	Destroy HandlerFunc
}

type resourceConfig struct {
	parameter string
	only      map[string]bool
	except    map[string]bool
}

// ResourceOption customizes Resource registration.
type ResourceOption func(*resourceConfig)

// Parameter sets the route parameter name (default: singular resource name).
func Parameter(name string) ResourceOption {
	return func(cfg *resourceConfig) {
		cfg.parameter = name
	}
}

// Only registers only the given resource actions.
func Only(actions ...string) ResourceOption {
	return func(cfg *resourceConfig) {
		cfg.only = make(map[string]bool, len(actions))
		for _, action := range actions {
			cfg.only[strings.ToLower(action)] = true
		}
	}
}

// Except skips the given resource actions.
func Except(actions ...string) ResourceOption {
	return func(cfg *resourceConfig) {
		cfg.except = make(map[string]bool, len(actions))
		for _, action := range actions {
			cfg.except[strings.ToLower(action)] = true
		}
	}
}

// Resource registers standard REST routes for name (e.g. "posts").
func (r *Router) Resource(name string, res Resource, opts ...ResourceOption) []*Route {
	name = strings.Trim(name, "/")
	cfg := &resourceConfig{
		parameter: str.Singular(name),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.parameter == "" {
		cfg.parameter = "id"
	}

	allow := func(action string) bool {
		if len(cfg.only) > 0 && !cfg.only[action] {
			return false
		}
		if cfg.except[action] {
			return false
		}
		return true
	}

	base := "/" + name
	item := base + "/{" + cfg.parameter + "}"
	registered := make([]*Route, 0, 7)

	add := func(method, path, action string, handler HandlerFunc) {
		if handler == nil || !allow(action) {
			return
		}
		registered = append(registered, r.Add(method, path, handler).As(name+"."+action))
	}

	add("GET", base, "index", res.Index)
	add("GET", base+"/create", "create", res.Create)
	add("POST", base, "store", res.Store)
	add("GET", item, "show", res.Show)
	add("GET", item+"/edit", "edit", res.Edit)
	add("PUT", item, "update", res.Update)
	add("PATCH", item, "update", res.Update)
	add("DELETE", item, "destroy", res.Destroy)

	return registered
}
