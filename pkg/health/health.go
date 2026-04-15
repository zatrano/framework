package health

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/core"
	"github.com/zatrano/framework/pkg/meta"
)

// Register attaches liveness, readiness, and aggregated status routes.
func Register(a *core.App, app *fiber.App) {
	app.Get("/health", liveness)

	app.Get("/ready", func(c fiber.Ctx) error {
		return readiness(c, a)
	})

	app.Get("/status", func(c fiber.Ctx) error {
		return status(c, a)
	})
}

func liveness(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": meta.Version,
	})
}

func readiness(c fiber.Ctx, a *core.App) error {
	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	st := probe(ctx, a)
	if !st.Ready {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"ready":   false,
			"checks":  st.Checks,
			"message": "one or more required dependencies are unavailable",
		})
	}
	return c.JSON(fiber.Map{
		"ready":  true,
		"checks": st.Checks,
	})
}

func status(c fiber.Ctx, a *core.App) error {
	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	st := probe(ctx, a)
	return c.JSON(fiber.Map{
		"app":       a.Config.AppName,
		"env":       a.Config.Env,
		"version":   meta.Version,
		"ready":     st.Ready,
		"checks":    st.Checks,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// Status is the aggregated readiness view.
type Status struct {
	Ready  bool           `json:"ready"`
	Checks map[string]any `json:"checks"`
}

// Probe runs the same checks as HTTP /status without starting a server.
func Probe(ctx context.Context, a *core.App) Status {
	return probe(ctx, a)
}

func probe(ctx context.Context, a *core.App) Status {
	checks := map[string]any{}
	ready := true

	// PostgreSQL
	switch {
	case strings.TrimSpace(a.Config.DatabaseURL) == "" && !a.Config.DatabaseRequired:
		checks["postgres"] = fiber.Map{"configured": false, "required": false, "ok": true}
	case a.DB == nil:
		checks["postgres"] = fiber.Map{"configured": true, "ok": false, "error": "gorm db is nil"}
		ready = false
	default:
		sqlDB, err := a.DB.DB()
		if err != nil {
			checks["postgres"] = fiber.Map{"configured": true, "ok": false, "error": err.Error()}
			ready = false
			break
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			checks["postgres"] = fiber.Map{"configured": true, "ok": false, "error": err.Error()}
			ready = false
		} else {
			checks["postgres"] = fiber.Map{"configured": true, "ok": true}
		}
	}

	// Redis
	switch {
	case strings.TrimSpace(a.Config.RedisURL) == "" && !a.Config.RedisRequired:
		checks["redis"] = fiber.Map{"configured": false, "required": false, "ok": true}
	case a.Redis == nil:
		checks["redis"] = fiber.Map{"configured": true, "ok": false, "error": "redis client is nil"}
		ready = false
	default:
		if err := a.Redis.Ping(ctx).Err(); err != nil {
			checks["redis"] = fiber.Map{"configured": true, "ok": false, "error": err.Error()}
			ready = false
		} else {
			checks["redis"] = fiber.Map{"configured": true, "ok": true}
		}
	}

	if a.Config.DatabaseRequired {
		if pg, ok := checks["postgres"].(fiber.Map); ok {
			if o, _ := pg["ok"].(bool); !o {
				ready = false
			}
		}
	}
	if a.Config.RedisRequired {
		if rd, ok := checks["redis"].(fiber.Map); ok {
			if o, _ := rd["ok"].(bool); !o {
				ready = false
			}
		}
	}

	hostname, _ := os.Hostname()
	checks["host"] = hostname

	return Status{Ready: ready, Checks: checks}
}
