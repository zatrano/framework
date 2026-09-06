package console

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/config"
	"github.com/zatrano/framework/v2/kernel/routing"
)

func registerCacheCommands(console *Application, app *kernel.Application) {
	console.Register(
		&ConfigCacheCommand{app: app},
		&ConfigClearCommand{app: app},
		&RouteListCommand{app: app},
		&RouteCacheCommand{app: app},
		&RouteClearCommand{app: app},
	)
}

func configCachePath(app *kernel.Application) string {
	return app.BasePath("storage", "framework", "cache", config.CacheFileName)
}

func routeCachePath(app *kernel.Application) string {
	return app.BasePath("storage", "framework", "cache", "routes.json")
}

type ConfigCacheCommand struct{ app *kernel.Application }

func (c *ConfigCacheCommand) Name() string { return "config:cache" }
func (c *ConfigCacheCommand) Description() string {
	return "Create a cache file for faster configuration loading"
}
func (c *ConfigCacheCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	path := configCachePath(c.app)
	if err := config.SaveCache(path, c.app.Config()); err != nil {
		return err
	}
	fmt.Printf("Configuration cached successfully: %s\n", path)
	return nil
}

type ConfigClearCommand struct{ app *kernel.Application }

func (c *ConfigClearCommand) Name() string        { return "config:clear" }
func (c *ConfigClearCommand) Description() string { return "Remove the configuration cache file" }
func (c *ConfigClearCommand) Handle(args []string) error {
	path := configCachePath(c.app)
	if err := config.ClearCache(path); err != nil {
		return err
	}
	fmt.Println("Configuration cache cleared successfully.")
	return nil
}

type RouteListCommand struct{ app *kernel.Application }

func (c *RouteListCommand) Name() string        { return "route:list" }
func (c *RouteListCommand) Description() string { return "List all registered routes" }
func (c *RouteListCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "METHOD\tURI\tNAME")
	for _, route := range c.app.Router().Snapshot() {
		name := route.Name
		if name == "" {
			name = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", route.Method, route.Path, name)
	}
	return w.Flush()
}

type RouteCacheCommand struct{ app *kernel.Application }

func (c *RouteCacheCommand) Name() string { return "route:cache" }
func (c *RouteCacheCommand) Description() string {
	return "Create a route cache file for inspection/deploy checks"
}
func (c *RouteCacheCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	path := routeCachePath(c.app)
	if err := c.app.Router().SaveCache(path); err != nil {
		return err
	}
	fmt.Printf("Routes cached successfully: %s\n", path)
	return nil
}

type RouteClearCommand struct{ app *kernel.Application }

func (c *RouteClearCommand) Name() string        { return "route:clear" }
func (c *RouteClearCommand) Description() string { return "Remove the route cache file" }
func (c *RouteClearCommand) Handle(args []string) error {
	path := routeCachePath(c.app)
	if err := routing.ClearRouteCache(path); err != nil {
		return err
	}
	fmt.Println("Route cache cleared successfully.")
	_ = filepath.Dir(path)
	return nil
}
