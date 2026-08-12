package jsonapi_test

import (
	"testing"

	"github.com/zatrano/framework/packages/jsonapi"
)

func TestJSONAPIDocument(t *testing.T) {
	doc := jsonapi.One(jsonapi.Make("users", "1", map[string]any{"name": "Ada"}))
	r, ok := doc.Data.(jsonapi.Resource)
	if !ok || r.Type != "users" || r.Attributes["name"] != "Ada" {
		t.Fatalf("%+v", doc)
	}
	many := jsonapi.Many([]jsonapi.Resource{
		jsonapi.Make("users", "1", map[string]any{"name": "Ada"}),
		jsonapi.Make("users", "2", map[string]any{"name": "Grace"}),
	})
	items, ok := many.Data.([]jsonapi.Resource)
	if !ok || len(items) != 2 {
		t.Fatalf("%+v", many)
	}
}
