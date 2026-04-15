package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"go.uber.org/zap"
)

// AccessLog returns middleware that logs one line per request with latency and status.
func AccessLog(log *zap.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if status == 0 {
			status = fiber.StatusOK
		}
		rid := requestid.FromContext(c)
		fields := []zap.Field{
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get(fiber.HeaderUserAgent)),
		}
		if rid != "" {
			fields = append(fields, zap.String("request_id", rid))
		}
		if err != nil {
			fields = append(fields, zap.Error(err))
			log.Warn("request completed", fields...)
			return err
		}
		if status >= 500 {
			log.Warn("request completed", fields...)
		} else {
			log.Info("request completed", fields...)
		}
		return nil
	}
}

