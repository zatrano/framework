package openapi_test

import (
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/openapi"
	"github.com/zatrano/framework/packages/routing"
)

func TestGenerateOpenAPI(t *testing.T) {
	router := routing.New()
	router.Get("/api/health", func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	}).As("api.health")
	router.Get("/api/users/{id}", func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{})
	}).As("api.users.show")

	spec := openapi.Generate(router.Routes(), openapi.Options{
		Title:     "Test",
		Version:   "1.0.0",
		ServerURL: "http://localhost",
	})
	if spec.OpenAPI != "3.0.3" {
		t.Fatalf("unexpected version %s", spec.OpenAPI)
	}
	if _, ok := spec.Paths["/api/health"]; !ok {
		t.Fatal("missing /api/health")
	}
	item := spec.Paths["/api/users/{id}"].(map[string]any)
	get := item["get"].(map[string]any)
	params := get["parameters"].([]map[string]any)
	if len(params) != 1 || params[0]["name"] != "id" {
		t.Fatalf("unexpected params %#v", params)
	}
}
