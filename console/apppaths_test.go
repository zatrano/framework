package console

import (
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestConsumerModuleFallsBack(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	if got := consumerModule(app); got != "your/module" {
		t.Fatalf("got %q", got)
	}
}
