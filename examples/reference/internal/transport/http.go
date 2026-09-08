package transport

import (
	"errors"

	"github.com/zatrano/framework/v2/examples/reference/internal/domain"
	"github.com/zatrano/framework/v2/examples/reference/internal/service"
	"github.com/zatrano/framework/v2/examples/reference/internal/worker"
	"github.com/zatrano/framework/v2/kernel/http"
)

// Up is process liveness after Bootstrap (kernel already 503s before Booted).
func Up(_ *http.Request) *http.Response {
	return http.JSON(map[string]any{"status": "ok"})
}

// Status is application-level information. It is not a substitute for /up.
func Status(name string, w *worker.Ticker) func(*http.Request) *http.Response {
	return func(_ *http.Request) *http.Response {
		return http.JSON(map[string]any{
			"name":   name,
			"worker": w != nil && w.Running(),
		})
	}
}

// ShowItem serves one item.
func ShowItem(items *service.Items) func(*http.Request) *http.Response {
	return func(req *http.Request) *http.Response {
		item, err := items.Get(req.Route("id"))
		if err != nil {
			return MapError(err)
		}
		return http.JSON(map[string]any{
			"id":   item.ID,
			"name": item.Name,
		})
	}
}

// MapError translates application errors to HTTP. Client bodies never include
// wrapped infrastructure messages (those may contain secrets).
func MapError(err error) *http.Response {
	switch {
	case err == nil:
		return http.InternalServerError("internal error")
	case errors.Is(err, domain.ErrNotFound):
		return http.NotFound("item not found")
	case errors.Is(err, domain.ErrInvalidRequest):
		return http.BadRequest("invalid request")
	case errors.Is(err, domain.ErrUnavailable):
		return http.ServiceUnavailable("dependency unavailable")
	case errors.Is(err, domain.ErrConfiguration):
		return http.InternalServerError("configuration failure")
	default:
		return http.InternalServerError("internal error")
	}
}
