package graphql_test

import (
	"testing"

	"github.com/zatrano/framework/packages/graphql"
)

func TestGraphQLExecute(t *testing.T) {
	schema := graphql.NewSchema()
	schema.Query("health", func(args map[string]any) (any, error) { return "ok", nil })
	schema.Query("echo", func(args map[string]any) (any, error) {
		return args["message"], nil
	})
	schema.Mutation("ping", func(args map[string]any) (any, error) {
		return true, nil
	})

	resp := schema.Execute(`{ health echo(message: "hi") }`, nil)
	if resp.Data["health"] != "ok" {
		t.Fatalf("unexpected %#v", resp.Data)
	}
	if resp.Data["echo"] != "hi" {
		t.Fatalf("unexpected echo %#v", resp.Data["echo"])
	}

	mut := schema.Execute(`mutation { ping }`, nil)
	if mut.Data["ping"] != true {
		t.Fatalf("unexpected mutation %#v", mut.Data)
	}
}
