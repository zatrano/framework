package routing_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

func TestWhereNumberAndFallback(t *testing.T) {
	r := routing.New()
	r.Get("/posts/{id}", func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"id": req.Route("id")})
	}).WhereNumber("id").As("posts.show")

	r.Fallback(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"fallback": true, "path": req.Path()}).Status(404)
	})

	ok := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/posts/42", nil)))
	if ok.StatusCode() != 200 {
		t.Fatalf("status=%d body=%s", ok.StatusCode(), string(ok.Content()))
	}
	bad := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/posts/abc", nil)))
	if bad.StatusCode() != 404 {
		t.Fatalf("expected fallback 404 for non-numeric, got %d", bad.StatusCode())
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
