package kernel_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/encryption"
)

func TestProductionRejectsPasswordAPP_KEY(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_KEY", "password")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err == nil {
		t.Fatal("production boot accepted password APP_KEY")
	}
}

func TestProductionKeyErrorDoesNotLeakValue(t *testing.T) {
	secret := "hunter2-production-key-value"
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_KEY", secret)
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	err := app.Bootstrap()
	if err == nil {
		t.Fatal("expected production key error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("APP_KEY value leaked: %v", err)
	}
}

func TestProductionRejectsPaddedShortKey(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_KEY", "short")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err == nil {
		t.Fatal("production boot accepted non-AES-length APP_KEY")
	}
}

func TestProductionAccepts32ByteKey(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
}

func TestProductionRejectsLocalDevKey(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_KEY", encryption.LocalDevKey)
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err == nil {
		t.Fatal("production boot accepted local dev key")
	}
}
