package concurrency_test

import (
	"testing"

	"github.com/zatrano/framework/packages/concurrency"
)

func TestMap(t *testing.T) {
	results := concurrency.Map(map[string]func() (int, error){
		"a": func() (int, error) { return 1, nil },
		"b": func() (int, error) { return 2, nil },
	})
	if results["a"].Value != 1 || results["b"].Value != 2 {
		t.Fatalf("unexpected results: %#v", results)
	}
}
