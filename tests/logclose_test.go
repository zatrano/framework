package tests

import (
	"testing"

	"github.com/zatrano/framework/v2/contracts"
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
