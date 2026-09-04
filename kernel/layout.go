package kernel

import (
	"os"
	"path/filepath"

	"github.com/zatrano/framework/contracts"
)

// LayoutDir returns preferred if it exists as a directory, otherwise fallback.
func LayoutDir(app contracts.App, preferred, fallback []string) string {
	if app == nil {
		return filepath.Join(fallback...)
	}
	p := app.BasePath(preferred...)
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p
	}
	return app.BasePath(fallback...)
}

// LayoutDirForCreate prefers an existing preferred or fallback directory; otherwise preferred (new layout).
func LayoutDirForCreate(app contracts.App, preferred, fallback []string) string {
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
	return LayoutDir(app, []string{"app", "views"}, []string{"views"})
}

// ViewsDirForCreate is the views root used when scaffolding files.
func ViewsDirForCreate(app contracts.App) string {
	return LayoutDirForCreate(app, []string{"app", "views"}, []string{"views"})
}

// LocalizationDir is app/localization when present, otherwise lang/.
func LocalizationDir(app contracts.App) string {
	return LayoutDir(app, []string{"app", "localization"}, []string{"lang"})
}

// LocalizationDirForCreate is the locale root used when scaffolding files.
func LocalizationDirForCreate(app contracts.App) string {
	return LayoutDirForCreate(app, []string{"app", "localization"}, []string{"lang"})
}

// DatabaseDir is app/database when present, otherwise database/.
func DatabaseDir(app contracts.App) string {
	return LayoutDir(app, []string{"app", "database"}, []string{"database"})
}

// DatabaseDirForCreate is the database root used when scaffolding files.
func DatabaseDirForCreate(app contracts.App) string {
	return LayoutDirForCreate(app, []string{"app", "database"}, []string{"database"})
}
