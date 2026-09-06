package console

import (
	"strings"

	"github.com/zatrano/framework/v2/kernel"
)

func consumerModule(app *kernel.Application) string {
	if app == nil {
		return "your/module"
	}
	mod, err := modulePath(app.BasePath())
	if err != nil || strings.TrimSpace(mod) == "" || mod == "github.com/zatrano/framework/v2" {
		return "your/module"
	}
	return mod
}
