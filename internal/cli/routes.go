package cli

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cobra"

	"github.com/zatrano/framework/pkg/config"
	"github.com/zatrano/framework/pkg/core"
	"github.com/zatrano/framework/pkg/server"
)

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "List registered HTTP routes (Fiber) and exit",
	Long: `Loads the same configuration as serve, builds the router, and prints all routes.
Useful after gen module/wire or when debugging mounts.

Routes are sorted by URL prefix group (first path segment), then path, then method priority.`,
	RunE: runRoutes,
}

func init() {
	routesCmd.Flags().String("env", "", "environment name; default ENV or dev")
	routesCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	routesCmd.Flags().Bool("no-dotenv", false, "do not load .env from the working directory")
	routesCmd.Flags().Bool("json", false, "print as JSON lines (method, path, name)")
	routesCmd.Flags().Bool("all", false, "include middleware-only routes (noisier; default filters them)")
	routesCmd.Flags().Bool("group", false, "insert a blank line when the first path segment changes (table output only)")
	rootCmd.AddCommand(routesCmd)
}

func runRoutes(cmd *cobra.Command, _ []string) error {
	envFlag, _ := cmd.Flags().GetString("env")
	configDir, _ := cmd.Flags().GetString("config-dir")
	noDotenv, _ := cmd.Flags().GetBool("no-dotenv")
	asJSON, _ := cmd.Flags().GetBool("json")
	allRoutes, _ := cmd.Flags().GetBool("all")
	groupBreak, _ := cmd.Flags().GetBool("group")

	cfg, err := config.Load(config.LoadOptions{
		Env:       envFlag,
		ConfigDir: configDir,
		DotEnv:    !noDotenv,
	})
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	app, err := core.Bootstrap(cfg)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	defer func() { _ = app.Close() }()

	fiberApp := core.NewFiber(app)
	server.Mount(app, fiberApp, server.MountOptions{})

	var routes []fiber.Route
	if allRoutes {
		routes = fiberApp.GetRoutes()
	} else {
		// filterUseOption=true drops routes registered only as middleware (cleaner listing).
		routes = fiberApp.GetRoutes(true)
	}
	sort.Slice(routes, func(i, j int) bool {
		gi, gj := pathGroup(routes[i].Path), pathGroup(routes[j].Path)
		if gi != gj {
			return gi < gj
		}
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		ri, rj := methodRank(routes[i].Method), methodRank(routes[j].Method)
		if ri != rj {
			return ri < rj
		}
		return routes[i].Method < routes[j].Method
	})

	if asJSON {
		for _, r := range routes {
			name := strings.TrimSpace(r.Name)
			if name == "" {
				name = "-"
			}
			fmt.Fprintf(os.Stdout, `{"method":%q,"path":%q,"name":%q}`+"\n", r.Method, r.Path, name)
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "METHOD\tPATH\tNAME")
	var prevGroup string
	for _, r := range routes {
		if groupBreak {
			g := pathGroup(r.Path)
			if prevGroup != "" && g != prevGroup {
				_, _ = fmt.Fprintln(w)
			}
			prevGroup = g
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", r.Method, r.Path, name)
	}
	return w.Flush()
}

// pathGroup is the first URL segment (e.g. /api/v1/x -> /api, /health -> /health).
func pathGroup(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return "/"
	}
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "/"
	}
	i := strings.IndexByte(p, '/')
	if i < 0 {
		return "/" + p
	}
	return "/" + p[:i]
}

func methodRank(m string) int {
	switch m {
	case http.MethodGet:
		return 0
	case http.MethodHead:
		return 1
	case http.MethodPost:
		return 2
	case http.MethodPut:
		return 3
	case http.MethodPatch:
		return 4
	case http.MethodDelete:
		return 5
	default:
		return 50
	}
}
