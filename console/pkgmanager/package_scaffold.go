package pkgmanager

import (
	"os"
	"path/filepath"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/kernel"
)

// packageAppDirs is the consumer app/ tree created when a package is enabled.
// Names match the historical starter folders; they are not a second catalog.
func packageAppDirs(name string) []string {
	switch name {
	case "queue":
		return []string{"app/jobs"}
	case "broadcasting":
		return []string{"app/broadcasting"}
	case "notification":
		return []string{"app/notifications"}
	case "authorization":
		return []string{"app/policies"}
	case "database":
		return []string{"app/database", "app/database/migrations", "app/database/seeders", "app/database/factories"}
	case "orm":
		return []string{"app/models"}
	case "validation":
		return []string{"app/http/requests", "app/rules"}
	case "localization":
		return []string{"app/localization"}
	case "view":
		return []string{"app/views", "app/views/web", "app/views/layout"}
	case "factory":
		return []string{"app/database/factories"}
	case "resources":
		return []string{"app/http/resources"}
	default:
		return nil
	}
}

func scaffoldPackageDirs(app *kernel.Application, names []string) error {
	if app == nil {
		return nil
	}
	for _, name := range names {
		for _, rel := range packageAppDirs(name) {
			dir := app.BasePath(filepath.FromSlash(rel))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			keep := filepath.Join(dir, ".gitkeep")
			if _, err := os.Stat(keep); err == nil {
				continue
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				return err
			}
			if len(entries) > 0 {
				continue
			}
			if err := os.WriteFile(keep, []byte(""), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func scaffoldPackageFiles(app *kernel.Application, names []string) error {
	if app == nil {
		return nil
	}
	for _, name := range names {
		meta, ok := addons.Lookup(name)
		if !ok || meta.Scaffold == nil {
			continue
		}
		if err := meta.Scaffold(app); err != nil {
			return err
		}
	}
	return nil
}
