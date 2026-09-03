package tests

import (
	"testing"

	"github.com/zatrano/framework/bootstrap"
)

func TestFrameworkAppBootsWithoutConsumerRoutes(t *testing.T) {
	app := bootstrap.App(bootstrap.Minimal())
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Router() == nil {
		t.Fatal("expected router")
	}
}
