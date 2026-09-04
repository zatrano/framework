package routing_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/http"
	"github.com/zatrano/framework/routing"
)

func TestCatchAllParamAndFallback(t *testing.T) {
	r := routing.New()
	r.Get("/docs/{*slug}", func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"slug": req.Route("slug")})
	}).As("docs.show")

	r.Fallback(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"fallback": true, "path": req.Path()}).Status(404)
	})

	nested := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/docs/digging-deeper/queues", nil)))
	if nested.StatusCode() != 200 {
		t.Fatalf("status=%d body=%s", nested.StatusCode(), string(nested.Content()))
	}
	if !strings.Contains(string(nested.Content()), `digging-deeper/queues`) {
		t.Fatalf("body=%s", string(nested.Content()))
	}

	missing := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/nope", nil)))
	if missing.StatusCode() != 404 {
		t.Fatalf("fallback missing=%d", missing.StatusCode())
	}
}

func TestRouterRedirectAndNamed(t *testing.T) {
	r := routing.New()
	r.Get("/home", func(req *http.Request) *http.Response {
		return http.Text("home")
	}).As("home")
	r.RegisterName(r.Routes()[0])
	r.Redirect("/go-home", "/home", 302)

	resp := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/go-home", nil)))
	if !resp.IsRedirect() || resp.RedirectURL() != "/home" {
		t.Fatalf("redirect=%v url=%q", resp.IsRedirect(), resp.RedirectURL())
	}

	named := r.RedirectRoute("home", nil)
	if !named.IsRedirect() || named.RedirectURL() != "/home" {
		t.Fatalf("named redirect=%q", named.RedirectURL())
	}
}
