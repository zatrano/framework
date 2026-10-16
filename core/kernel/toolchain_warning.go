package kernel

import (
	"runtime"

	"github.com/zatrano/framework/v3/core/kernel/goversion"
)

// toolchainWarning is the production boot warning for a binary whose stdlib
// is older than goversion.RecommendedMinimum. The module's go directive does
// not pick the stdlib; the toolchain that built the binary does.
func toolchainWarning(production bool, built string) string {
	if !production || !goversion.OlderThan(built, goversion.RecommendedMinimum) {
		return ""
	}
	return "toolchain: this binary was built with " + built +
		"; stdlib security fixes require " + goversion.RecommendedMinimum +
		" or newer. Rebuild with the go.dev/dl toolchain or GOTOOLCHAIN=" +
		goversion.RecommendedMinimum + " (see SECURITY.md, Build toolchain)."
}

func (app *Application) warnOldToolchain() {
	if app.logger == nil {
		return
	}
	if msg := toolchainWarning(app.IsProduction(), runtime.Version()); msg != "" {
		app.logger.Warning(msg)
	}
}
