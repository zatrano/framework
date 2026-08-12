package resources_test

import (
	"testing"

	"github.com/zatrano/framework/packages/resources"
)

func TestWrapAndCollection(t *testing.T) {
	items := []int{1, 2}
	out := resources.WrapCollection(items, func(n int) map[string]any {
		return map[string]any{"n": n}
	})
	data, ok := out["data"].([]map[string]any)
	if !ok || len(data) != 2 || data[0]["n"] != 1 {
		t.Fatalf("unexpected resource: %#v", out)
	}
}
