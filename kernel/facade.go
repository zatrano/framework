package kernel

import (
	"fmt"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/report"
	"github.com/zatrano/framework/v2/kernel/routing"
)

type routerFacade struct{ inner *routing.Router }

func (f *routerFacade) Get(path string, handler any) contracts.Route {
	if f == nil || f.inner == nil {
		return nil
	}
	return &routeFacade{inner: f.inner.Get(path, asHandler(handler))}
}

func (f *routerFacade) Post(path string, handler any) contracts.Route {
	if f == nil || f.inner == nil {
		return nil
	}
	return &routeFacade{inner: f.inner.Post(path, asHandler(handler))}
}

func (f *routerFacade) Use(middleware ...any) {
	if f == nil || f.inner == nil {
		return
	}
	f.inner.Use(asMiddlewareList(middleware)...)
}

func (f *routerFacade) Group(prefix string, fn func(contracts.Router), middleware ...any) {
	if f == nil || f.inner == nil || fn == nil {
		return
	}
	f.inner.Group(prefix, func(*routing.Router) {
		fn(f)
	}, asMiddlewareList(middleware)...)
}

func (f *routerFacade) Name(prefix string, fn func(contracts.Router)) {
	if f == nil || f.inner == nil || fn == nil {
		return
	}
	f.inner.Name(prefix, func(*routing.Router) {
		fn(f)
	})
}

func (f *routerFacade) Snapshot() []contracts.RouteSnapshot {
	if f == nil || f.inner == nil {
		return nil
	}
	src := f.inner.Snapshot()
	out := make([]contracts.RouteSnapshot, len(src))
	for i, r := range src {
		out[i] = contracts.RouteSnapshot{Method: r.Method, Path: r.Path, Name: r.Name}
	}
	return out
}

func (f *routerFacade) SaveCache(path string) error {
	if f == nil || f.inner == nil {
		return fmt.Errorf("router unavailable")
	}
	return f.inner.SaveCache(path)
}

type routeFacade struct{ inner *routing.Route }

func (f *routeFacade) As(name string) contracts.Route {
	if f == nil || f.inner == nil {
		return f
	}
	f.inner.As(name)
	return f
}

func asHandler(handler any) routing.HandlerFunc {
	h, err := handlerOf(handler)
	if err != nil {
		panic("router: " + err.Error())
	}
	return h
}

func handlerOf(handler any) (routing.HandlerFunc, error) {
	if handler == nil {
		return nil, fmt.Errorf("nil handler")
	}
	switch h := handler.(type) {
	case routing.HandlerFunc:
		if h == nil {
			return nil, fmt.Errorf("nil handler")
		}
		return h, nil
	case func(*http.Request) *http.Response:
		if h == nil {
			return nil, fmt.Errorf("nil handler")
		}
		return h, nil
	default:
		return nil, fmt.Errorf("unsupported handler %T", handler)
	}
}

func asMiddlewareList(items []any) []routing.MiddlewareFunc {
	out := make([]routing.MiddlewareFunc, 0, len(items))
	for _, item := range items {
		out = append(out, asMiddleware(item))
	}
	return out
}

func asMiddleware(item any) routing.MiddlewareFunc {
	mw, err := middlewareOf(item)
	if err != nil {
		panic("router: " + err.Error())
	}
	return mw
}

func middlewareOf(item any) (routing.MiddlewareFunc, error) {
	if item == nil {
		return nil, fmt.Errorf("nil middleware")
	}
	switch m := item.(type) {
	case routing.MiddlewareFunc:
		return m, nil
	case func(routing.HandlerFunc) routing.HandlerFunc:
		return m, nil
	default:
		return nil, fmt.Errorf("unsupported middleware %T", item)
	}
}

type exceptionsFacade struct{ inner middlewareProvider }

func (f *exceptionsFacade) Middleware() any {
	if f == nil || f.inner == nil {
		return nil
	}
	return f.inner.Middleware()
}

type reportsFacade struct{ inner *report.Manager }

func (f *reportsFacade) Recent(limit int) []contracts.ReportEvent {
	if f == nil || f.inner == nil {
		return nil
	}
	src := f.inner.Recent(limit)
	out := make([]contracts.ReportEvent, len(src))
	for i, e := range src {
		out[i] = contracts.ReportEvent{
			ID:        e.ID,
			Message:   e.Message,
			Level:     e.Level,
			Path:      e.Path,
			Method:    e.Method,
			CreatedAt: e.CreatedAt,
		}
	}
	return out
}

var _ contracts.Router = (*routerFacade)(nil)
var _ contracts.Exceptions = (*exceptionsFacade)(nil)
var _ contracts.Reports = (*reportsFacade)(nil)
