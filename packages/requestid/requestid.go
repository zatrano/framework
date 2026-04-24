package requestid

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const HeaderName = "X-Request-ID"
const LocalsKey = "request_id"

type contextKey string

const ctxKey contextKey = "request_id"

// Middleware — her isteğe benzersiz bir UUID atar.
// Gelen X-Request-ID header'ı varsa onu kullanır (upstream proxy uyumu).
// Tüm Zap log alanlarına otomatik eklenir; alt servislere header olarak iletilir.
func Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		reqID := c.Get(HeaderName)
		if reqID == "" {
			reqID = uuid.New().String()
		}
		c.Set(HeaderName, reqID)
		c.Locals(LocalsKey, reqID)
		return c.Next()
	}
}

// FromFiber — Fiber context'inden request ID'yi alır.
func FromFiber(c fiber.Ctx) string {
	if id, ok := c.Locals(LocalsKey).(string); ok {
		return id
	}
	return ""
}

// SetToContext — request ID'yi stdlib context'e atar (servis katmanı için).
func SetToContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey, id)
}

// FromContext — stdlib context'ten request ID'yi okur.
func FromContext(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKey).(string); ok {
		return id
	}
	return ""
}
