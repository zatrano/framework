package bootstrap

import (
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func closeAppLog(t *testing.T, app *kernel.Application) {
	t.Helper()
	if app == nil {
		return
	}
	t.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
}
