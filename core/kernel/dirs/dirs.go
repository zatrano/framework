package dirs

import (
	"os"
	"path/filepath"

	"github.com/zatrano/framework/v3/core/contracts"
)

// Dir returns preferred if it exists as a directory, otherwise fallback.
func Dir(app contracts.App, preferred, fallback []string) string {
	if app == nil {
		return filepath.Join(fallback...)
	}
	p := app.BasePath(preferred...)
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p
	}
	return app.BasePath(fallback...)
}

// DirForCreate prefers an existing preferred or fallback directory; otherwise preferred (new app tree).
func DirForCreate(app contracts.App, preferred, fallback []string) string {
	if app == nil {
		return filepath.Join(preferred...)
	}
	p := app.BasePath(preferred...)
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p
	}
	f := app.BasePath(fallback...)
	if st, err := os.Stat(f); err == nil && st.IsDir() {
		return f
	}
	return p
}

// TemplatesDir is templates/ (Canvas SSR). Legacy templates is not used in V3.
func TemplatesDir(app contracts.App) string {
	if app == nil {
		return "templates"
	}
	return app.BasePath("templates")
}

// TemplatesDirForCreate is the templates root used when scaffolding files.
func TemplatesDirForCreate(app contracts.App) string {
	return TemplatesDir(app)
}

// LocalizationDir is app/localization when present, otherwise lang/.
func LocalizationDir(app contracts.App) string {
	return Dir(app, []string{"app", "localization"}, []string{"lang"})
}

// LocalizationDirForCreate is the locale root used when scaffolding files.
func LocalizationDirForCreate(app contracts.App) string {
	return DirForCreate(app, []string{"app", "localization"}, []string{"lang"})
}

// DatabaseDir is database/ at the application root (V3).
func DatabaseDir(app contracts.App) string {
	if app == nil {
		return "database"
	}
	return app.BasePath("database")
}

// DatabaseDirForCreate is the database root used when scaffolding files.
func DatabaseDirForCreate(app contracts.App) string {
	return DatabaseDir(app)
}

// CanonicalConsumerDirs are required of every app created by zatrano new.
func CanonicalConsumerDirs() []string {
	return []string{
		"core",
		"app/http/handlers/api",
		"app/http/handlers/web",
		"app/providers",
		"app/routes",
		"app/routes/api",
		"app/routes/web",
		"templates",
		"database",
		"cmd/app",
	}
}

// CanonicalRouteDirs are where RegisterWeb / RegisterAPI belong.
func CanonicalRouteDirs() []string {
	return []string{
		"app/routes/web",
		"app/routes/api",
	}
}

// OptionalWebScaffoldDirs are template-package directories, not kernel requirements.
func OptionalWebScaffoldDirs() []string {
	return []string{
		"templates/web",
		"templates/layouts",
		"templates/components",
		"app/localization",
		"public/css",
		"public/js",
	}
}
