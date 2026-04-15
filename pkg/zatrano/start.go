package zatrano

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"

	"github.com/zatrano/framework/pkg/config"
	"github.com/zatrano/framework/pkg/core"
	"github.com/zatrano/framework/pkg/meta"
	"github.com/zatrano/framework/pkg/server"
)

// StartOptions configures the embedded HTTP server (same behavior as `zatrano serve`).
type StartOptions struct {
	Env       string
	ConfigDir string
	Addr      string
	NoDotenv  bool
	// RegisterRoutes mounts app-specific modules (e.g. internal/routes.Register). Optional.
	RegisterRoutes func(a *core.App, app *fiber.App)
}

// Start boots the ZATRANO HTTP server. Intended for generated apps: `zatrano.Run()`.
func Start(opts StartOptions) error {
	cfg, err := config.Load(config.LoadOptions{
		Env:       opts.Env,
		ConfigDir: opts.ConfigDir,
		DotEnv:    !opts.NoDotenv,
	})
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if opts.Addr != "" {
		cfg.HTTPAddr = opts.Addr
	}

	app, err := core.Bootstrap(cfg)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			app.Log.Warn("shutdown resources", zap.Error(err))
		}
	}()

	fiberApp := core.NewFiber(app)
	server.Mount(app, fiberApp, server.MountOptions{RegisterRoutes: opts.RegisterRoutes})
	app.Fiber = fiberApp

	app.Log.Info("zatrano starting",
		zap.String("version", meta.Version),
		zap.String("env", cfg.Env),
		zap.String("addr", cfg.HTTPAddr),
	)

	errCh := make(chan error, 1)
	ready := make(chan struct{})
	go func() {
		errCh <- fiberApp.Listen(cfg.HTTPAddr, fiber.ListenConfig{
			DisableStartupMessage: true,
			BeforeServeFunc: func(*fiber.App) error {
				close(ready)
				return nil
			},
		})
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		return fmt.Errorf("server exited before becoming ready")
	case <-ready:
		app.Log.Info("listening", zap.String("url", localBaseURL(cfg.HTTPAddr)))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	app.Log.Info("shutdown signal received, draining connections...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := fiberApp.ShutdownWithContext(shutdownCtx); err != nil {
		return fmt.Errorf("fiber shutdown: %w", err)
	}

	if err := <-errCh; err != nil {
		return fmt.Errorf("server: %w", err)
	}
	app.Log.Info("server stopped cleanly")
	return nil
}

// Run reads environment variables and config from the working directory (with .env) and starts the server.
func Run() error {
	return Start(StartOptions{})
}

func localBaseURL(addr string) string {
	if len(addr) > 0 && addr[0] == ':' {
		return "http://127.0.0.1" + addr
	}
	return "http://" + addr
}
