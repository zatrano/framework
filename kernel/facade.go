package kernel

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/packages/health"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/maintenance"
	"github.com/zatrano/framework/packages/ratelimit"
	"github.com/zatrano/framework/packages/report"
	"github.com/zatrano/framework/packages/routing"
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
	if handler == nil {
		return nil
	}
	switch h := handler.(type) {
	case routing.HandlerFunc:
		return h
	case func(*http.Request) *http.Response:
		return h
	default:
		return func(*http.Request) *http.Response {
			return http.JSON(map[string]any{"message": fmt.Sprintf("unsupported handler %T", handler)}).Status(500)
		}
	}
}

func asMiddlewareList(items []any) []routing.MiddlewareFunc {
	out := make([]routing.MiddlewareFunc, 0, len(items))
	for _, item := range items {
		if mw := asMiddleware(item); mw != nil {
			out = append(out, mw)
		}
	}
	return out
}

func asMiddleware(item any) routing.MiddlewareFunc {
	if item == nil {
		return nil
	}
	switch m := item.(type) {
	case routing.MiddlewareFunc:
		return m
	case func(routing.HandlerFunc) routing.HandlerFunc:
		return m
	default:
		return nil
	}
}

type rateLimiterFacade struct{ inner *ratelimit.Limiter }

func (f *rateLimiterFacade) For(name string, limit contracts.RateLimit) {
	if f == nil || f.inner == nil {
		return
	}
	f.inner.For(name, ratelimit.Limit{MaxAttempts: limit.MaxAttempts, Decay: limit.Decay})
}

type healthFacade struct{ inner *health.Manager }

func (f *healthFacade) Custom(name string, check func(ctx context.Context) error) {
	if f == nil || f.inner == nil {
		return
	}
	f.inner.Custom(name, check)
}

func (f *healthFacade) Database(db *sql.DB) {
	if f == nil || f.inner == nil {
		return
	}
	f.inner.Database(db)
}

func (f *healthFacade) Handler() any {
	if f == nil || f.inner == nil {
		return nil
	}
	return f.inner.Handler()
}

type maintenanceFacade struct{ inner *maintenance.Manager }

func (f *maintenanceFacade) Enable(payload contracts.MaintenancePayload) error {
	if f == nil || f.inner == nil {
		return fmt.Errorf("maintenance unavailable")
	}
	return f.inner.Enable(maintenance.Payload{
		Message:    payload.Message,
		RetryAfter: payload.RetryAfter,
		AllowedIPs: payload.AllowedIPs,
		Secret:     payload.Secret,
		Time:       payload.Time,
	})
}

func (f *maintenanceFacade) Disable() error {
	if f == nil || f.inner == nil {
		return fmt.Errorf("maintenance unavailable")
	}
	return f.inner.Disable()
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
var _ contracts.RateLimiter = (*rateLimiterFacade)(nil)
var _ contracts.Health = (*healthFacade)(nil)
var _ contracts.Maintenance = (*maintenanceFacade)(nil)
var _ contracts.Exceptions = (*exceptionsFacade)(nil)
var _ contracts.Reports = (*reportsFacade)(nil)
