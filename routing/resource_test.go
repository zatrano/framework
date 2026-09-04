package routing_test

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/http"
	"github.com/zatrano/framework/routing"
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
