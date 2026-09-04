package console

import (
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/kernel"
)

func consumerModule(app *kernel.Application) string {
	if app == nil {
		return "your/module"
	}
	mod, err := modulePath(app.BasePath())
	if err != nil || strings.TrimSpace(mod) == "" || mod == "github.com/zatrano/framework" {
		return "your/module"
	}
	return mod
}

func applyConsumerPlaceholders(app *kernel.Application, body string) string {
	mod := consumerModule(app)
	body = strings.ReplaceAll(body, "__MODULE__", mod)
	body = strings.ReplaceAll(body, "github.com/zatrano/framework/app/", mod+"/app/")
	return body
}

func viewsRoot(app *kernel.Application) string {
	return kernel.ViewsDirForCreate(app)
}

func localizationRoot(app *kernel.Application) string {
	return kernel.LocalizationDirForCreate(app)
}

func databaseRoot(app *kernel.Application) string {
	return kernel.DatabaseDirForCreate(app)
}

func joinRoot(root string, parts ...string) string {
	return filepath.Join(append([]string{root}, parts...)...)
}

// scaffoldDest maps starter layout prefixes (views/, lang/, database/) onto
// app/views, app/localization, app/database when scaffolding files.
func scaffoldDest(app *kernel.Application, parts []string) string {
	if app == nil || len(parts) == 0 {
		return ""
	}
	switch parts[0] {
	case "views":
		return joinRoot(viewsRoot(app), parts[1:]...)
	case "lang":
		return joinRoot(localizationRoot(app), parts[1:]...)
	case "database":
		return joinRoot(databaseRoot(app), parts[1:]...)
	default:
		return app.BasePath(parts...)
	}
}
