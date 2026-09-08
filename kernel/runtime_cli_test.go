package kernel_test

import (
	"errors"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestRunStartFailureIsRuntimeBoot(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	startErr := errors.New("start boom")
	app.RegisterProviders(&lifecycleProbe{name: "fail", fail: startErr})
	err := app.Run(":0")
	if !errors.Is(err, kernel.ErrRuntimeBoot) {
		t.Fatalf("Run Start failure must wrap ErrRuntimeBoot, got %v", err)
	}
	if !errors.Is(err, startErr) {
		t.Fatalf("original Start error must remain inspectable, got %v", err)
	}
	if errors.Is(err, kernel.ErrRuntimeShutdown) {
		t.Fatal("Start failure is boot, not shutdown")
	}
}
