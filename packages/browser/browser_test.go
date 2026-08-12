package browser_test

import (
	"testing"

	"github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/packages/browser"
)

func TestBrowserVisitHome(t *testing.T) {
	app := bootstrap.App()
	b, err := browser.New(app)
	if err != nil {
		t.Fatal(err)
	}
	b.Visit("/").AssertOK().AssertSee("ZATRANO").AssertPathIs("/")
}
