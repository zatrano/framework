package tests

import (
	"testing"

	"github.com/zatrano/framework/v3/core/contracts"
)

func closeAppLog(t *testing.T, app contracts.App) {
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
