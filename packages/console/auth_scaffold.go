package console

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zatrano/framework/core"
)

func registerAuthCommands(console *Application, app *core.Application) {
	console.Register(&MakeAuthCommand{app: app})
}

// MakeAuthCommand scaffolds the full application auth layer.
type MakeAuthCommand struct {
	app *core.Application
}

func (c *MakeAuthCommand) Name() string { return "make:auth" }
func (c *MakeAuthCommand) Description() string {
	return "Scaffold full auth (views, layout, controllers, model, migrations, routes, provider)"
}

func (c *MakeAuthCommand) Handle(args []string) error {
	force := false
	viewsOnly := false
	for _, arg := range args {
		switch arg {
		case "--force", "-f":
			force = true
		case "--views":
			viewsOnly = true
		}
	}

	stubRoot := c.stubRoot()

	type filePair struct {
		stub string
		dest []string
	}

	pairs := []filePair{
		{"layouts/auth.html", []string{"views", "layouts", "auth.html"}},
		{"auth/login.html", []string{"views", "auth", "login.html"}},
		{"auth/register.html", []string{"views", "auth", "register.html"}},
		{"auth/forgot-password.html", []string{"views", "auth", "forgot-password.html"}},
		{"auth/reset-password.html", []string{"views", "auth", "reset-password.html"}},
		{"auth/confirm-password.html", []string{"views", "auth", "confirm-password.html"}},
		{"auth/change-password.html", []string{"views", "auth", "change-password.html"}},
		{"auth/profile.html", []string{"views", "auth", "profile.html"}},
		{"auth/verify-email.html", []string{"views", "auth", "verify-email.html"}},
		{"auth/two-factor-challenge.html", []string{"views", "auth", "two-factor-challenge.html"}},
		{"auth/two-factor.html", []string{"views", "auth", "two-factor.html"}},
		{"auth/logout-other-devices.html", []string{"views", "auth", "logout-other-devices.html"}},
		{"lang/en/auth.json", []string{"lang", "en", "auth.json"}},
		{"lang/tr/auth.json", []string{"lang", "tr", "auth.json"}},
	}

	if !viewsOnly {
		pairs = append(pairs,
			filePair{"go/user_model.go.stub", []string{"app", "models", "user.go"}},
			filePair{"go/social_account_model.go.stub", []string{"app", "models", "social_account.go"}},
			filePair{"go/user_factory.go.stub", []string{"database", "factories", "user_factory.go"}},
			filePair{"go/user_resource.go.stub", []string{"app", "http", "resources", "user_resource.go"}},
			filePair{"go/auth_controller.go.stub", []string{"app", "http", "controllers", "web", "auth_controller.go"}},
			filePair{"go/social_auth_controller.go.stub", []string{"app", "http", "controllers", "web", "social_auth_controller.go"}},
			filePair{"go/authenticate_middleware.go.stub", []string{"app", "http", "middleware", "authenticate.go"}},
			filePair{"go/routes_auth.go.stub", []string{"routes", "auth.go"}},
			filePair{"go/auth_service_provider.go.stub", []string{"app", "providers", "auth_service_provider.go"}},
			filePair{"go/migration_auth.go.stub", []string{"database", "migrations", "create_auth_tables.go"}},
			filePair{"go/migration_social_accounts.go.stub", []string{"database", "migrations", "create_social_accounts_table.go"}},
		)
	}

	created, skipped := 0, 0
	for _, pair := range pairs {
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
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("%s: %w", pair.stub, err)
		}
		fmt.Printf("Created: %s\n", dst)
		created++
	}

	fmt.Printf("\nAuth scaffold ready (%d created, %d skipped).\n", created, skipped)
	if viewsOnly {
		fmt.Println("Mode: --views (layout + auth HTML only)")
	} else {
		fmt.Println("Next steps:")
		fmt.Println("  1. In database/migrations/migrations.go add:")
		fmt.Println("     &CreateUsersTable{}, &CreatePasswordResetTokensTable{}, &CreatePersonalAccessTokensTable{}, &CreateSocialAccountsTable{},")
		fmt.Println("  2. In routes/web.go call: RegisterAuthWeb(app)")
		fmt.Println("  3. In routes/api.go call: RegisterAuthAPI(app)")
		fmt.Println("  4. Set GOOGLE_*/GITHUB_* env vars for social login (optional)")
		fmt.Println("  5. Run: go run ./cmd/zatrano migrate")
	}
	fmt.Println("Use --force to overwrite existing files. Use --views for views only.")
	return nil
}

func (c *MakeAuthCommand) stubRoot() string {
	candidates := []string{
		c.app.BasePath("packages", "console", "stubs"),
		filepath.Join(filepath.Dir(c.app.BasePath()), "packages", "console", "stubs"),
		c.app.BasePath("core", "console", "stubs"),
		filepath.Join(filepath.Dir(c.app.BasePath()), "core", "console", "stubs"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Join(candidate, "layouts", "auth.html")); err == nil && !info.IsDir() {
			return candidate
		}
		if info, err := os.Stat(filepath.Join(candidate, "go", "auth_controller.go.stub")); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return c.app.BasePath("packages", "console", "stubs")
}

var (
	goTypeDecl = regexp.MustCompile(`(?m)^type\s+([A-Za-z_][A-Za-z0-9_]*)\s+`)
	goFuncDecl = regexp.MustCompile(`(?m)^func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
)

// goStubConflicts returns a human reason when writing dst would redeclare symbols already in the package.
func goStubConflicts(stubPath, pkgDir, dst string) (string, error) {
	stubBytes, err := os.ReadFile(stubPath)
	if err != nil {
		return "", err
	}
	stub := string(stubBytes)
	typeNames := uniqueMatches(goTypeDecl, stub)
	funcNames := uniqueMatches(goFuncDecl, stub)

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(pkgDir, entry.Name())
		if filepath.Clean(path) == filepath.Clean(dst) {
			continue // same file will be overwritten under --force
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		text := string(body)
		for _, name := range typeNames {
			for _, existing := range uniqueMatches(goTypeDecl, text) {
				if existing == name {
					return fmt.Sprintf("type %s already in %s", name, entry.Name()), nil
				}
			}
		}
		for _, name := range funcNames {
			for _, existing := range uniqueMatches(goFuncDecl, text) {
				if existing == name {
					return fmt.Sprintf("func %s already in %s", name, entry.Name()), nil
				}
			}
		}
		if strings.Contains(stub, "RegisterAuthWeb") && strings.Contains(text, `.As("login")`) {
			return fmt.Sprintf("login routes already in %s", entry.Name()), nil
		}
		if strings.Contains(stub, "CreateUsersTable") &&
			(strings.Contains(text, "remember_token") || strings.Contains(text, "two_factor_secret") || strings.Contains(text, "CreateUsersTable")) {
			return fmt.Sprintf("auth tables already in %s", entry.Name()), nil
		}
		if strings.Contains(stub, "CreateSocialAccountsTable") &&
			(strings.Contains(text, "provider_uid") || strings.Contains(text, "CreateSocialAccountsTable")) {
			return fmt.Sprintf("social accounts already in %s", entry.Name()), nil
		}
	}
	return "", nil
}

func uniqueMatches(re *regexp.Regexp, src string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		if len(m) < 2 {
			continue
		}
		name := m[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
