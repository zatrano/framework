package pipeline_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/pipeline"
)

func TestPipelineThrough(t *testing.T) {
	result := pipeline.Send("ada").
		Through(
			pipeline.Via(func(v any) any { return strings.ToUpper(v.(string)) }),
			func(passable any, next func(any) any) any {
				return next(passable.(string) + "!")
			},
		).
		ThenReturn()
	if result != "ADA!" {
		t.Fatalf("got %v", result)
	}
}
