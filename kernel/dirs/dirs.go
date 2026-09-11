package dirs

import (
	"os"
	"path/filepath"

	"github.com/zatrano/framework/v2/contracts"
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

// ViewsDir is app/views when present, otherwise views/.
func ViewsDir(app contracts.App) string {
	return Dir(app, []string{"app", "views"}, []string{"views"})
}

// ViewsDirForCreate is the views root used when scaffolding files.
func ViewsDirForCreate(app contracts.App) string {
	return DirForCreate(app, []string{"app", "views"}, []string{"views"})
}

// LocalizationDir is app/localization when present, otherwise lang/.
func LocalizationDir(app contracts.App) string {
	return Dir(app, []string{"app", "localization"}, []string{"lang"})
}

// LocalizationDirForCreate is the locale root used when scaffolding files.
func LocalizationDirForCreate(app contracts.App) string {
	return DirForCreate(app, []string{"app", "localization"}, []string{"lang"})
}

// DatabaseDir is app/database when present, otherwise database/.
func DatabaseDir(app contracts.App) string {
	return Dir(app, []string{"app", "database"}, []string{"database"})
}

// DatabaseDirForCreate is the database root used when scaffolding files.
func DatabaseDirForCreate(app contracts.App) string {
	return DirForCreate(app, []string{"app", "database"}, []string{"database"})
}

// CanonicalConsumerDirs are the application directories required of every
// app created by zatrano new. Doctor and agents use this list; they must not
// infer it from console/templates.
func CanonicalConsumerDirs() []string {
	return []string{
		"app/console",
		"app/http/controllers/api",
		"app/http/controllers/web",
		"app/providers",
		"app/routes",
		"app/routes/api",
		"app/routes/web",
		"bootstrap",
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

// OptionalWebScaffoldDirs are view-package directories, not kernel requirements.
func OptionalWebScaffoldDirs() []string {
	return []string{
		"app/views",
		"app/localization",
		"public/css",
		"public/js",
	}
}
