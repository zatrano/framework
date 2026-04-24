package middlewares

import (
	"strings"
	"time"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/packages/requestid"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func ZapLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		path := c.Path()
		if shouldSkipLog(path) {
			return c.Next()
		}

		start := time.Now()
		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()
		method := c.Method()
		ip := getRealIP(c)
		reqID := requestid.FromFiber(c)

		fields := []zap.Field{
			zap.String("request_id", reqID), // ← tüm log'larda request_id
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", ip),
		}

		if ua := c.Get("User-Agent"); ua != "" && len(ua) < 200 {
			fields = append(fields, zap.String("user_agent", ua))
		}
		if referer := c.Get("Referer"); referer != "" && len(referer) < 500 {
			fields = append(fields, zap.String("referer", referer))
		}
		if err != nil {
			fields = append(fields, zap.Error(err))
		}

		logByStatus(fields, status, latency, method)
		return err
	}
}

func shouldSkipLog(path string) bool {
	if strings.HasPrefix(path, "/health") ||
		strings.HasPrefix(path, "/metrics") ||
		path == "/favicon.ico" {
		return true
	}
	if strings.HasPrefix(path, "/public/") ||
		strings.HasPrefix(path, "/uploads/") {
		return true
	}
	if strings.HasPrefix(path, "/.well-known/") {
		return true
	}
	return false
}

func getRealIP(c fiber.Ctx) string {
	if ip := c.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := c.Get("X-Forwarded-For"); ip != "" {
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	return c.IP()
}

func logByStatus(fields []zap.Field, status int, latency time.Duration, method string) {
	msg := "request"
	if status >= 500 {
		msg = "server_error"
	} else if status >= 400 && status != 404 {
		msg = "client_error"
	} else if latency > time.Second {
		msg = "slow_request"
		fields = append(fields, zap.Bool("slow", true))
	}

	switch {
	case status >= 500:
		logconfig.Log.Error(msg, fields...)
	case status >= 400:
		if status == 404 {
			logconfig.Log.Info(msg, fields...)
		} else {
			logconfig.Log.Warn(msg, fields...)
		}
	default:
		if method != "GET" || latency > 500*time.Millisecond {
			logconfig.Log.Info(msg, fields...)
		} else {
			logconfig.Log.Debug(msg, fields...)
		}
	}
}
