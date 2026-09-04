package routing_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
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

func TestURLEscapesPathParams(t *testing.T) {
	r := routing.New()
	r.Get("/users/{id}", func(req *http.Request) *http.Response {
		return http.Text("ok")
	}).As("users.show")
	r.RegisterName(r.Routes()[0])
	got, err := r.URL("users.show", map[string]string{"id": "hello/world"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/users/hello%2Fworld" {
		t.Fatalf("got %q", got)
	}
}

func TestURLCatchAllKeepsSlashes(t *testing.T) {
	r := routing.New()
	r.Get("/docs/{*slug}", func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"slug": req.Route("slug")})
	}).As("docs.show")
	r.RegisterName(r.Routes()[0])
	got, err := r.URL("docs.show", map[string]string{"slug": "a/b"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/docs/a/b" {
		t.Fatalf("got %q", got)
	}
}

func TestCatchAllEmptyAndRoot(t *testing.T) {
	r := routing.New()
	r.Get("/docs/{*slug}", func(req *http.Request) *http.Response {
		return http.Text(req.Route("slug"))
	})
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}
	missing := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/docs", nil)))
	if missing.StatusCode() == 200 {
		t.Fatal("empty catch-all should not match /docs")
	}
	ok := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/docs/x", nil)))
	if ok.StatusCode() != 200 || string(ok.Content()) != "x" {
		t.Fatalf("got status=%d body=%s", ok.StatusCode(), ok.Content())
	}
}

func TestRouterFreeze(t *testing.T) {
	r := routing.New()
	r.Get("/", func(req *http.Request) *http.Response { return http.Text("ok") })
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	r.Post("/x", func(req *http.Request) *http.Response { return http.Text("no") })
}

func TestCompiledStaticBeatsParam(t *testing.T) {
	r := routing.New()
	r.Get("/users/{id}", func(req *http.Request) *http.Response {
		return http.Text("param:" + req.Route("id"))
	})
	r.Get("/users/new", func(req *http.Request) *http.Response {
		return http.Text("static")
	})
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}
	resp := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/users/new", nil)))
	if string(resp.Content()) != "static" {
		t.Fatalf("got %s", resp.Content())
	}
	param := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/users/42", nil)))
	if string(param.Content()) != "param:42" {
		t.Fatalf("got %s", param.Content())
	}
}

func TestTrieParamAndCatchAll(t *testing.T) {
	r := routing.New()
	r.Get("/files/{id}/meta", func(req *http.Request) *http.Response {
		return http.Text("meta:" + req.Route("id"))
	})
	r.Get("/files/{*path}", func(req *http.Request) *http.Response {
		return http.Text("all:" + req.Route("path"))
	})
	r.Get("/docs/{page?}", func(req *http.Request) *http.Response {
		return http.Text("page:" + req.Route("page"))
	})
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}

	meta := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/files/abc/meta", nil)))
	if string(meta.Content()) != "meta:abc" {
		t.Fatalf("meta=%s", meta.Content())
	}
	all := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/files/a/b", nil)))
	if string(all.Content()) != "all:a/b" {
		t.Fatalf("all=%s", all.Content())
	}
	docs := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/docs", nil)))
	if string(docs.Content()) != "page:" {
		t.Fatalf("docs empty optional=%s", docs.Content())
	}
	page := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/docs/intro", nil)))
	if string(page.Content()) != "page:intro" {
		t.Fatalf("page=%s", page.Content())
	}
}

func TestUnfrozenStillFirstMatchWins(t *testing.T) {
	r := routing.New()
	r.Get("/users/{id}", func(req *http.Request) *http.Response {
		return http.Text("param")
	})
	r.Get("/users/new", func(req *http.Request) *http.Response {
		return http.Text("static")
	})
	resp := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/users/new", nil)))
	if string(resp.Content()) != "param" {
		t.Fatalf("unfrozen should keep registration order, got %s", resp.Content())
	}
}

func TestDuplicateStaticRouteFreezeErrors(t *testing.T) {
	r := routing.New()
	h := func(req *http.Request) *http.Response { return http.Text("ok") }
	r.Get("/ping", h)
	r.Get("/ping", h)
	if err := r.Freeze(); err == nil {
		t.Fatal("expected duplicate route error")
	}
	if r.Frozen() {
		t.Fatal("failed freeze must not lock the router")
	}
}

func TestAmbiguousParamRouteFreezeErrors(t *testing.T) {
	r := routing.New()
	h := func(req *http.Request) *http.Response { return http.Text("ok") }
	r.Get("/users/{id}", h)
	r.Get("/users/{name}", h)
	if err := r.Freeze(); err == nil {
		t.Fatal("expected ambiguous route error")
	}
}

func TestFrozenRouteCannotAsOrThrough(t *testing.T) {
	r := routing.New()
	route := r.Get("/users", func(req *http.Request) *http.Response { return http.Text("ok") })
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected As panic after freeze")
		}
	}()
	route.As("users")
}

func TestFrozenRouteCannotThrough(t *testing.T) {
	r := routing.New()
	route := r.Get("/users", func(req *http.Request) *http.Response { return http.Text("ok") })
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected Through panic after freeze")
		}
	}()
	route.Through(func(next routing.HandlerFunc) routing.HandlerFunc { return next })
}

func TestGroupPanicRestoresPrefix(t *testing.T) {
	r := routing.New()
	func() {
		defer func() { _ = recover() }()
		r.Group("/api", func(router *routing.Router) {
			panic("boom")
		})
	}()
	r.Get("/ok", func(req *http.Request) *http.Response { return http.Text("ok") })
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}
	resp := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/ok", nil)))
	if resp.StatusCode() != 200 {
		t.Fatalf("group panic leaked prefix, status=%d", resp.StatusCode())
	}
	leaked := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/api/ok", nil)))
	if leaked.StatusCode() == 200 {
		t.Fatal("group prefix should have been restored")
	}
}

func TestNamePanicRestoresPrefix(t *testing.T) {
	r := routing.New()
	func() {
		defer func() { _ = recover() }()
		r.Name("api.", func(router *routing.Router) {
			panic("boom")
		})
	}()
	r.Get("/x", func(req *http.Request) *http.Response { return http.Text("ok") }).As("show")
	r.RegisterName(r.Routes()[0])
	if _, ok := r.Route("show"); !ok {
		t.Fatal("name prefix leaked; expected show")
	}
	if _, ok := r.Route("api.show"); ok {
		t.Fatal("name prefix was not restored")
	}
}

func TestDuplicateRouteNameFreezeErrors(t *testing.T) {
	r := routing.New()
	h := func(req *http.Request) *http.Response { return http.Text("ok") }
	r.Get("/a", h).As("same")
	r.Get("/b", h).As("same")
	if err := r.Freeze(); err == nil {
		t.Fatal("expected duplicate route name error")
	}
	if r.Frozen() {
		t.Fatal("failed freeze must not lock the router")
	}
}

func TestURLMissingRequiredParamErrors(t *testing.T) {
	r := routing.New()
	r.Get("/users/{id}", func(req *http.Request) *http.Response { return http.Text("ok") }).As("users.show")
	r.RegisterName(r.Routes()[0])
	if _, err := r.URL("users.show"); err == nil {
		t.Fatal("expected missing required parameter error")
	}
	got, err := r.URL("users.show", map[string]string{"id": "9"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/users/9" {
		t.Fatalf("got %q", got)
	}
}
