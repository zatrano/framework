package kernel_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestRunRejectsNonIntegerAPP_PORT(t *testing.T) {
	t.Setenv("APP_PORT", "abc")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	err := app.Run("")
	if err == nil {
		t.Fatal("expected configuration error")
	}
	if !errors.Is(err, kernel.ErrRuntimeBoot) {
		t.Fatalf("want ErrRuntimeBoot, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "APP_PORT") || !strings.Contains(msg, "integer") {
		t.Fatalf("must name variable and expected type: %v", err)
	}
}

func TestRunRejectsOutOfRangeAPP_PORT(t *testing.T) {
	t.Setenv("APP_PORT", "70000")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	err := app.Run("")
	if err == nil {
		t.Fatal("expected configuration error")
	}
	if !errors.Is(err, kernel.ErrRuntimeBoot) {
		t.Fatalf("want ErrRuntimeBoot, got %v", err)
	}
	if !strings.Contains(err.Error(), "APP_PORT") {
		t.Fatalf("must name APP_PORT: %v", err)
	}
}
