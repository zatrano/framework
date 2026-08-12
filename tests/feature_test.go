package tests

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/bootstrap"
	testkit "github.com/zatrano/framework/packages/testing"
)

func TestHealthEndpoint(t *testing.T) {
	app := bootstrap.App()
	tc, err := testkit.New(app)
	if err != nil {
		t.Fatal(err)
	}

	tc.AcceptJSON().Get("/api/health").
		AssertOK().
		AssertJSONContains("status", "ok")

	tc.AcceptJSON().Get("/up").
		AssertOK().
		AssertJSONContains("status", "ok")
}

func TestWelcomeUsesLayout(t *testing.T) {
	app := bootstrap.App()
	tc, err := testkit.New(app)
	if err != nil {
		t.Fatal(err)
	}
	html := tc.Get("/")
	html.AssertOK()
	body := string(html.Body)
	if !strings.Contains(body, "ZATRANO") {
		t.Fatalf("expected brand in welcome page")
	}
}
