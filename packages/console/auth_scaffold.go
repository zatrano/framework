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
	socialProviders := []string{"google", "github"} // default both when social stubs enabled
	socialFlagSet := false
	for _, arg := range args {
		switch {
		case arg == "--force" || arg == "-f":
			force = true
		case arg == "--views":
			viewsOnly = true
		case strings.HasPrefix(arg, "--social="):
			socialFlagSet = true
			raw := strings.TrimSpace(strings.TrimPrefix(arg, "--social="))
			socialProviders = nil
			for _, p := range strings.Split(raw, ",") {
				p = strings.ToLower(strings.TrimSpace(p))
				if p == "google" || p == "github" {
					socialProviders = appendUnique(socialProviders, p)
				}
			}
		}
	}
	if !socialFlagSet {
		socialProviders = []string{"google", "github"}
	}
	wantGoogle := containsStr(socialProviders, "google")
	wantGitHub := containsStr(socialProviders, "github")
	wantSocial := !viewsOnly && (wantGoogle || wantGitHub)

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
			filePair{"go/user_factory.go.stub", []string{"database", "factories", "user_factory.go"}},
			filePair{"go/user_resource.go.stub", []string{"app", "http", "resources", "user_resource.go"}},
			filePair{"go/auth_controller.go.stub", []string{"app", "http", "controllers", "web", "auth_controller.go"}},
			filePair{"go/authenticate_middleware.go.stub", []string{"app", "http", "middleware", "authenticate.go"}},
			filePair{"go/routes_auth.go.stub", []string{"routes", "auth.go"}},
			filePair{"go/auth_service_provider.go.stub", []string{"app", "providers", "auth_service_provider.go"}},
			filePair{"go/migration_auth.go.stub", []string{"database", "migrations", "create_auth_tables.go"}},
		)
		if wantSocial {
			pairs = append(pairs,
				filePair{"go/social_account_model.go.stub", []string{"app", "models", "social_account.go"}},
				filePair{"go/social_auth_controller.go.stub", []string{"app", "http", "controllers", "web", "social_auth_controller.go"}},
				filePair{"go/migration_social_accounts.go.stub", []string{"database", "migrations", "create_social_accounts_table.go"}},
			)
		}
	}

	created, skipped := 0, 0
	var routesAuthPath, loginViewPath, registerViewPath string
	for _, pair := range pairs {
		src := filepath.Join(stubRoot, filepath.FromSlash(pair.stub))
		dst := c.app.BasePath(pair.dest...)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if !force {
			if _, err := os.Stat(dst); err == nil {
				if strings.HasSuffix(dst, "auth_service_provider.go") {
					fmt.Printf("Skipped (exists): %s — add gates/policies manually or use --force\n", dst)
				} else {
					fmt.Printf("Skipped (exists): %s\n", dst)
				}
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
		switch pair.stub {
		case "go/routes_auth.go.stub":
			routesAuthPath = dst
		case "auth/login.html":
			loginViewPath = dst
		case "auth/register.html":
			registerViewPath = dst
		}
	}

	if !viewsOnly && routesAuthPath != "" {
		if err := filterAuthSocialRoutes(routesAuthPath, wantGoogle, wantGitHub); err != nil {
			return err
		}
	}
	if loginViewPath != "" {
		if err := filterAuthSocialLinks(loginViewPath, wantGoogle, wantGitHub); err != nil {
			return err
		}
	}
	if registerViewPath != "" {
		if err := filterAuthSocialLinks(registerViewPath, wantGoogle, wantGitHub); err != nil {
			return err
		}
	}

	fmt.Printf("\nAuth scaffold ready (%d created, %d skipped).\n", created, skipped)
	if viewsOnly {
		fmt.Println("Mode: --views (layout + auth HTML only)")
	} else {
		fmt.Println("Next steps:")
		fmt.Println("  1. In database/migrations/migrations.go add:")
		migLine := "     &CreateUsersTable{}, &CreatePasswordResetTokensTable{}, &CreatePersonalAccessTokensTable{},"
		if wantSocial {
			migLine += " &CreateSocialAccountsTable{},"
		}
		fmt.Println(migLine)
		fmt.Println("  2. In routes/web.go call: RegisterAuthWeb(app)")
		fmt.Println("  3. In routes/api.go call: RegisterAuthAPI(app)  // mounts /api/v1/auth")
		if wantSocial {
			var envs []string
			if wantGoogle {
				envs = append(envs, "GOOGLE_*")
			}
			if wantGitHub {
				envs = append(envs, "GITHUB_*")
			}
			fmt.Printf("  4. Set %s env vars for social login (optional)\n", strings.Join(envs, "/"))
			fmt.Println("  5. Run: go run ./cmd/zatrano migrate")
		} else {
			fmt.Println("  4. Run: go run ./cmd/zatrano migrate")
		}
	}
	fmt.Println("Use --force to overwrite existing files. Use --views for views only.")
	fmt.Println("Use --social=google|github|google,github to select OAuth providers.")
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

func containsStr(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func appendUnique(list []string, v string) []string {
	if containsStr(list, v) {
		return list
	}
	return append(list, v)
}

func filterAuthSocialRoutes(path string, wantGoogle, wantGitHub bool) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(body), "\n")
	var out []string
	for _, line := range lines {
		lower := strings.ToLower(line)
		isGoogle := strings.Contains(lower, "google")
		isGitHub := strings.Contains(lower, "github")
		if strings.Contains(line, "social :=") || strings.Contains(line, "social:=") {
			if !wantGoogle && !wantGitHub {
				continue
			}
			out = append(out, line)
			continue
		}
		if isGoogle && !wantGoogle && (strings.Contains(line, "router.Get") || strings.Contains(line, "social.")) {
			continue
		}
		if isGitHub && !wantGitHub && (strings.Contains(line, "router.Get") || strings.Contains(line, "social.")) {
			continue
		}
		out = append(out, line)
	}
	// Drop unused social controller binding when no provider routes remain.
	if !wantGoogle && !wantGitHub {
		filtered := out[:0]
		for _, line := range out {
			if strings.Contains(line, "SocialAuthController") {
				continue
			}
			filtered = append(filtered, line)
		}
		out = filtered
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

func filterAuthSocialLinks(path string, wantGoogle, wantGitHub bool) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(body)
	if !wantGoogle {
		text = regexp.MustCompile(`(?m)^[ \t]*<a href="/login/google">.*</a>[ \t]*\r?\n`).ReplaceAllString(text, "")
	}
	if !wantGitHub {
		text = regexp.MustCompile(`(?m)^[ \t]*<a href="/login/github">.*</a>[ \t]*\r?\n`).ReplaceAllString(text, "")
	}
	if !wantGoogle && !wantGitHub {
		text = regexp.MustCompile(`(?m)^[ \t]*<p class="auth-divider">.*</p>[ \t]*\r?\n`).ReplaceAllString(text, "")
		text = regexp.MustCompile(`(?ms)[ \t]*<div class="auth-social">.*?</div>[ \t]*\r?\n`).ReplaceAllString(text, "")
	}
	return os.WriteFile(path, []byte(text), 0o644)
}
