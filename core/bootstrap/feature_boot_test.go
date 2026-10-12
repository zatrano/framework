package bootstrap

import (
	"testing"
)

func TestFrameworkAppBootsWithoutConsumerRoutes(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := App(WithBasePath(t.TempDir()))
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	closeAppLog(t, app)
	if app.Router() == nil {
		t.Fatal("expected router")
	}
}
