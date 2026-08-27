package console

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zatrano/framework/core"
)

func registerDashboardCommands(console *Application, app *core.Application) {
	console.Register(&MakeDashboardCommand{app: app})
}

// MakeDashboardCommand scaffolds the management dashboard (davet.link-style shell).
type MakeDashboardCommand struct {
	app *core.Application
}

func (c *MakeDashboardCommand) Name() string { return "make:dashboard" }
func (c *MakeDashboardCommand) Description() string {
	return "Scaffold dashboard (layout, CSS/JS, modules, /api/v1, i18n)"
}

var dashboardModules = []string{
	"users", "notifications", "roles", "rbac", "settings", "analytics", "api", "impersonate",
}

func (c *MakeDashboardCommand) Handle(args []string) error {
	force := false
	viewsOnly := false
	only := map[string]bool{}
	enabled := map[string]bool{}
	hasModuleFlag := false
	hasOnly := false

	for _, arg := range args {
		switch {
		case arg == "--force" || arg == "-f":
			force = true
		case arg == "--views":
			viewsOnly = true
		case strings.HasPrefix(arg, "--only="):
			hasOnly = true
			for _, part := range strings.Split(strings.TrimPrefix(arg, "--only="), ",") {
				part = strings.ToLower(strings.TrimSpace(part))
				if part != "" {
					only[part] = true
				}
			}
		case arg == "--users", arg == "--notifications", arg == "--roles", arg == "--rbac",
			arg == "--settings", arg == "--analytics", arg == "--api", arg == "--impersonate":
			hasModuleFlag = true
			enabled[strings.TrimPrefix(arg, "--")] = true
		case strings.HasPrefix(arg, "--no-"):
			name := strings.TrimPrefix(arg, "--no-")
			enabled[name] = false
			hasModuleFlag = true
		}
	}

	mods := map[string]bool{}
	for _, name := range dashboardModules {
		mods[name] = true
	}
	switch {
	case hasOnly:
		for _, name := range dashboardModules {
			mods[name] = only[name]
		}
		if mods["impersonate"] {
			mods["users"] = true
		}
	case hasModuleFlag:
		for _, name := range dashboardModules {
			if v, ok := enabled[name]; ok {
				mods[name] = v
			} else {
				mods[name] = false
			}
		}
		if mods["impersonate"] {
			mods["users"] = true
		}
	default:
		for _, name := range dashboardModules {
			mods[name] = true
		}
	}

	stubRoot := c.stubRoot()
	type filePair struct {
		stub    string
		dest    []string
		modules []string // empty = always; else any listed module must be on
	}

	pairs := []filePair{
		{"layouts/dashboard.html", []string{"views", "layouts", "dashboard.html"}, nil},
		{"partials/dashboard/nav.html", []string{"views", "partials", "dashboard", "nav.html"}, nil},
		{"dashboard/home.html", []string{"views", "dashboard", "home.html"}, nil},
		{"public/css/dashboard.css", []string{"public", "css", "dashboard.css"}, nil},
		{"public/js/dashboard-shell.js", []string{"public", "js", "dashboard-shell.js"}, nil},
		{"lang/en/dashboard.json", []string{"lang", "en", "dashboard.json"}, nil},
		{"lang/tr/dashboard.json", []string{"lang", "tr", "dashboard.json"}, nil},
		{"dashboard/users/index.html", []string{"views", "dashboard", "users", "index.html"}, []string{"users"}},
		{"dashboard/users/create.html", []string{"views", "dashboard", "users", "create.html"}, []string{"users"}},
		{"dashboard/users/edit.html", []string{"views", "dashboard", "users", "edit.html"}, []string{"users"}},
		{"dashboard/notifications/index.html", []string{"views", "dashboard", "notifications", "index.html"}, []string{"notifications"}},
		{"dashboard/notifications/send.html", []string{"views", "dashboard", "notifications", "send.html"}, []string{"notifications"}},
		{"dashboard/notifications/bulk.html", []string{"views", "dashboard", "notifications", "bulk.html"}, []string{"notifications"}},
		{"dashboard/roles/index.html", []string{"views", "dashboard", "roles", "index.html"}, []string{"roles"}},
		{"dashboard/roles/create.html", []string{"views", "dashboard", "roles", "create.html"}, []string{"roles"}},
		{"dashboard/roles/edit.html", []string{"views", "dashboard", "roles", "edit.html"}, []string{"roles"}},
		{"dashboard/rbac/matrix.html", []string{"views", "dashboard", "rbac", "matrix.html"}, []string{"rbac"}},
		{"dashboard/settings/index.html", []string{"views", "dashboard", "settings", "index.html"}, []string{"settings"}},
		{"dashboard/analytics/index.html", []string{"views", "dashboard", "analytics", "index.html"}, []string{"analytics"}},
		{"public/js/dashboard-analytics.js", []string{"public", "js", "dashboard-analytics.js"}, []string{"analytics"}},
	}

	if !viewsOnly {
		pairs = append(pairs,
			filePair{"go/dashboard_helpers.go.stub", []string{"app", "http", "controllers", "web", "dashboard_helpers.go"}, nil},
			filePair{"go/dashboard_controller.go.stub", []string{"app", "http", "controllers", "web", "dashboard_controller.go"}, nil},
			filePair{"go/routes_dashboard.go.stub", []string{"routes", "dashboard.go"}, nil},
			filePair{"go/dashboard_service_provider.go.stub", []string{"app", "providers", "dashboard_service_provider.go"}, nil},
			filePair{"go/migration_dashboard.go.stub", []string{"database", "migrations", "create_dashboard_tables.go"}, nil},
			filePair{"go/dashboard_users_controller.go.stub", []string{"app", "http", "controllers", "web", "dashboard_users_controller.go"}, []string{"users"}},
			filePair{"go/impersonate_middleware.go.stub", []string{"app", "http", "middleware", "impersonate.go"}, []string{"impersonate"}},
			filePair{"go/dashboard_notifications_controller.go.stub", []string{"app", "http", "controllers", "web", "dashboard_notifications_controller.go"}, []string{"notifications"}},
			filePair{"go/dashboard_roles_controller.go.stub", []string{"app", "http", "controllers", "web", "dashboard_roles_controller.go"}, []string{"roles"}},
			filePair{"go/dashboard_rbac_controller.go.stub", []string{"app", "http", "controllers", "web", "dashboard_rbac_controller.go"}, []string{"rbac"}},
			filePair{"go/dashboard_settings_controller.go.stub", []string{"app", "http", "controllers", "web", "dashboard_settings_controller.go"}, []string{"settings"}},
			filePair{"go/dashboard_analytics_controller.go.stub", []string{"app", "http", "controllers", "web", "dashboard_analytics_controller.go"}, []string{"analytics"}},
			filePair{"go/dashboard_api_controller.go.stub", []string{"app", "http", "controllers", "web", "dashboard_api_controller.go"}, []string{"api"}},
			filePair{"go/dashboard_models.go.stub", []string{"app", "models", "dashboard.go"}, []string{"roles", "rbac", "settings"}},
		)
	}

	created, skipped := 0, 0
	for _, pair := range pairs {
		if len(pair.modules) > 0 {
			ok := false
			for _, m := range pair.modules {
				if mods[m] {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		src := filepath.Join(stubRoot, filepath.FromSlash(pair.stub))
		dst := c.app.BasePath(pair.dest...)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if !force {
			if _, err := os.Stat(dst); err == nil {
				fmt.Printf("Skipped (exists): %s\n", dst)
				skipped++
				continue
			}
		}
		if strings.HasSuffix(dst, ".go") {
			if conflict, err := goStubConflicts(src, filepath.Dir(dst), dst); err != nil {
				return err
			} else if conflict != "" {
				fmt.Printf("Skipped (declared): %s — %s\n", dst, conflict)
				skipped++
				continue
			}
		}
		body, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("%s: %w", pair.stub, err)
		}
		text := filterDashboardModules(string(body), mods)
		if strings.HasSuffix(dst, ".go") && !mods["api"] {
			text = stripUnusedImport(text, "github.com/zatrano/framework/packages/api")
		}
		if strings.HasSuffix(dst, ".go") && !mods["impersonate"] {
			text = stripUnusedImport(text, "github.com/zatrano/framework/app/http/middleware")
		}
		base := filepath.Base(dst)
		if (base == "dashboard_controller.go" || base == "dashboard_analytics_controller.go") && !mods["notifications"] {
			text = stripUnusedImport(text, "github.com/zatrano/framework/packages/notification")
		}
		if base == "dashboard_controller.go" && !mods["users"] && !mods["roles"] {
			text = stripUnusedImport(text, "github.com/zatrano/framework/packages/orm")
			if !mods["notifications"] {
				text = stripUnusedImport(text, "github.com/zatrano/framework/app/models")
			}
		}
		if base == "dashboard_analytics_controller.go" && !mods["roles"] && !mods["settings"] && !mods["users"] {
			// users count always queries User
		}
		if base == "routes_dashboard.go" && !mods["api"] {
			text = stripUnusedImport(text, "github.com/zatrano/framework/packages/api")
		}
		if err := os.WriteFile(dst, []byte(text), 0o644); err != nil {
			return fmt.Errorf("%s: %w", pair.stub, err)
		}
		fmt.Printf("Created: %s\n", dst)
		created++
	}

	if !viewsOnly {
		if err := ensureUserIsAdminField(c.app.BasePath("app", "models", "user.go")); err != nil {
			fmt.Printf("Note: could not patch User.IsAdmin: %v\n", err)
		}
	}

	on := make([]string, 0, len(dashboardModules))
	for _, name := range dashboardModules {
		if mods[name] {
			on = append(on, name)
		}
	}
	fmt.Printf("\nDashboard scaffold ready (%d created, %d skipped).\n", created, skipped)
	fmt.Printf("Modules: %s\n", strings.Join(on, ", "))
	if viewsOnly {
		fmt.Println("Mode: --views (HTML/CSS/JS/lang only)")
	} else {
		fmt.Println("Next steps:")
		fmt.Println("  1. Ensure make:auth has been run (User model + auth routes).")
		fmt.Println("  2. Register DashboardServiceProvider in bootstrap/providers.")
		fmt.Println("  3. In database/migrations/migrations.go register CreateDashboard* migrations.")
		fmt.Println("  4. In routes/web.go call: RegisterDashboardWeb(app)")
		if mods["api"] {
			fmt.Println("  5. In routes/api.go call: RegisterDashboardAPI(app)")
			fmt.Println("  6. Run: go run ./cmd/zatrano migrate")
			fmt.Println("  7. Set users.is_admin=1 for an admin account.")
		} else {
			fmt.Println("  5. Run: go run ./cmd/zatrano migrate")
			fmt.Println("  6. Set users.is_admin=1 for an admin account.")
		}
	}
	fmt.Println("Flags: --force --views --only=users,rbac --users --notifications --roles --rbac --settings --analytics --api --impersonate")
	return nil
}

func (c *MakeDashboardCommand) stubRoot() string {
	candidates := []string{
		c.app.BasePath("packages", "console", "stubs"),
		filepath.Join(filepath.Dir(c.app.BasePath()), "packages", "console", "stubs"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Join(candidate, "layouts", "dashboard.html")); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return c.app.BasePath("packages", "console", "stubs")
}

func filterDashboardModules(text string, mods map[string]bool) string {
	order := []string{"impersonate", "users", "notifications", "roles", "rbac", "settings", "analytics", "api"}
	for _, name := range order {
		if mods[name] {
			continue
		}
		text = stripModuleBlock(text, name)
	}
	for _, name := range order {
		text = regexp.MustCompile(`(?m)^[ \t]*// @module `+regexp.QuoteMeta(name)+`[ \t]*\r?\n`).ReplaceAllString(text, "")
		text = regexp.MustCompile(`(?m)^[ \t]*// @endmodule `+regexp.QuoteMeta(name)+`[ \t]*\r?\n`).ReplaceAllString(text, "")
		text = regexp.MustCompile(`(?m)^[ \t]*<!-- @module `+regexp.QuoteMeta(name)+` -->[ \t]*\r?\n`).ReplaceAllString(text, "")
		text = regexp.MustCompile(`(?m)^[ \t]*<!-- @endmodule `+regexp.QuoteMeta(name)+` -->[ \t]*\r?\n`).ReplaceAllString(text, "")
	}
	return text
}

// FilterDashboardModulesForTest exports module filtering for unit tests.
func FilterDashboardModulesForTest(text string, mods map[string]bool) string {
	return filterDashboardModules(text, mods)
}

func stripModuleBlock(text, name string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?s)// @module ` + regexp.QuoteMeta(name) + `\r?\n.*?// @endmodule ` + regexp.QuoteMeta(name) + `\r?\n`),
		regexp.MustCompile(`(?s)<!-- @module ` + regexp.QuoteMeta(name) + ` -->\r?\n.*?<!-- @endmodule ` + regexp.QuoteMeta(name) + ` -->\r?\n`),
	}
	for _, re := range patterns {
		text = re.ReplaceAllString(text, "")
	}
	return text
}

func stripUnusedImport(text, importPath string) string {
	re := regexp.MustCompile(`(?m)^\t"` + regexp.QuoteMeta(importPath) + `"\r?\n`)
	return re.ReplaceAllString(text, "")
}

func ensureUserIsAdminField(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Note: app/models/user.go missing — run make:auth, then add IsAdmin bool `db:\"is_admin\" json:\"is_admin\"`")
			return nil
		}
		return err
	}
	text := string(body)
	if strings.Contains(text, "IsAdmin") {
		return nil
	}
	re := regexp.MustCompile("(?m)^(\tEmail\\s+string\\s+`[^`]+`)\\r?\\n")
	if re.MatchString(text) {
		text = re.ReplaceAllString(text, "$1\n\tIsAdmin                 bool       `db:\"is_admin\" json:\"is_admin\"`\n")
		fmt.Printf("Patched: %s (added IsAdmin)\n", path)
		return os.WriteFile(path, []byte(text), 0o644)
	}
	re = regexp.MustCompile("(?m)^(\tPassword\\s+string\\s+`[^`]+`)\\r?\\n")
	if re.MatchString(text) {
		text = re.ReplaceAllString(text, "$1\n\tIsAdmin                 bool       `db:\"is_admin\" json:\"is_admin\"`\n")
		fmt.Printf("Patched: %s (added IsAdmin)\n", path)
		return os.WriteFile(path, []byte(text), 0o644)
	}
	fmt.Println("Note: add IsAdmin bool `db:\"is_admin\" json:\"is_admin\"` to models.User manually.")
	return nil
}
