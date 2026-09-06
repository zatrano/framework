package tests

import (
	"testing"

	"github.com/zatrano/framework/v2/bootstrap"
)

func TestFrameworkAppBootsWithoutConsumerRoutes(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := bootstrap.App()
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Router() == nil {
		t.Fatal("expected router")
	}
}
