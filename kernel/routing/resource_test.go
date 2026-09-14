package routing_test

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

func TestNamePrefixAndResource(t *testing.T) {
	r := routing.New()
	var got string
	r.Name("api.", func(api *routing.Router) {
		api.Group("/api", func(g *routing.Router) {
			g.Resource("notes", routing.Resource{
				Index: func(req *http.Request) *http.Response {
					return http.JSON(map[string]any{"ok": true})
				},
				Show: func(req *http.Request) *http.Response {
					got = req.Route("note")
					return http.JSON(map[string]any{"id": got})
				},
			}, routing.Only("index", "show"))
		})
	})

	for _, route := range r.Routes() {
		r.RegisterName(route)
	}

	idx, ok := r.Route("api.notes.index")
	if !ok || idx.Path != "/api/notes" {
		t.Fatalf("index route: ok=%v path=%v", ok, idx)
	}
	show, ok := r.Route("api.notes.show")
	if !ok || show.Path != "/api/notes/{note}" {
		t.Fatalf("show route: ok=%v path=%v", ok, show)
	}

	resp := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/api/notes/42", nil)))
	if resp.StatusCode() != 200 || got != "42" {
		t.Fatalf("dispatch show: status=%d got=%q", resp.StatusCode(), got)
	}
}

func TestSubstituteBindings(t *testing.T) {
	routing.ClearBindings()
	t.Cleanup(routing.ClearBindings)

	routing.Bind("note", func(value string, req *http.Request) (any, error) {
		return map[string]any{"id": value, "title": "Hello"}, nil
	})

	r := routing.New()
	r.Use(routing.SubstituteBindings())
	r.Get("/notes/{note}", func(req *http.Request) *http.Response {
		note, _ := req.Get("note").(map[string]any)
		return http.JSON(note)
	})

	resp := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/notes/7", nil)))
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d body=%s", resp.StatusCode(), string(resp.Content()))
	}
	var payload map[string]any
	if err := json.Unmarshal(resp.Content(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["id"] != "7" || payload["title"] != "Hello" {
		t.Fatalf("payload=%v", payload)
	}

	routing.Bind("note", func(value string, req *http.Request) (any, error) {
		return nil, nil
	})
	missing := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/notes/404", nil)))
	if missing.StatusCode() != 404 {
		t.Fatalf("expected 404, got %d", missing.StatusCode())
	}
}

func TestControllerHelper(t *testing.T) {
	r := routing.New()
	type notes struct{}
	routing.Controller(r, notes{}, func(rr routing.RouteRegistrar, c notes) {
		rr.Get("/n", func(req *http.Request) *http.Response { return http.Text("ok") })
	})
	resp := r.Dispatch(http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/n", nil)))
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func TestCoverageVerbsDiscoveryAndCache(t *testing.T) {
	r := routing.New()
	ok := func(req *http.Request) *http.Response { return http.Text("ok") }
	r.Put("/p", ok)
	r.Patch("/p", ok)
	r.Delete("/p", ok)
	r.Options("/p", ok)
	r.Any("/any", ok)
	r.Match([]string{"GET", "POST"}, "/m", ok)
	r.Resource("posts", routing.Resource{Index: ok, Store: ok}, routing.Except("store"), routing.Parameter("post"))
	if routing.From(nil) != nil {
		t.Fatal("From nil")
	}
	routing.RegisterAPI(nil)
	routing.RegisterAPI(func(rr *routing.Router) { rr.Get("/api/z", ok) })
	routing.ApplyAPI(nil)
	api := routing.New()
	routing.ApplyAPI(api)
	if routing.HasBinding("nope") {
		t.Fatal("has binding")
	}
	path := filepath.Join(t.TempDir(), "routes.json")
	if err := r.SaveCache(path); err != nil {
		t.Fatal(err)
	}
	if _, err := routing.LoadRouteCache(path); err != nil {
		t.Fatal(err)
	}
	if err := routing.ClearRouteCache(path); err != nil {
		t.Fatal(err)
	}
	if err := routing.ClearRouteCache(path); err != nil {
		t.Fatal("missing cache")
	}
}
