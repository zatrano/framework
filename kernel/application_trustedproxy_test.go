package kernel_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel"
)

func prodProxyApp(t *testing.T) *kernel.Application {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	return app
}

func TestProductionWildcardTrustedProxiesFailsBootstrap(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "*")
	t.Setenv("TRUST_PROXIES_ALLOW_STAR", "true")
	app := prodProxyApp(t)
	err := app.Bootstrap()
	if err == nil {
		t.Fatal("production boot accepted TRUSTED_PROXIES=*")
	}
	if !strings.Contains(err.Error(), "TRUSTED_PROXIES") {
		t.Fatalf("error=%v", err)
	}
	if app.Bootstrapped() {
		t.Fatal("wildcard proxy config must not leave the app bootstrapped")
	}
	if !app.BootstrapFailed() {
		t.Fatal("expected BootstrapFailed")
	}
	if err := app.Start(); err == nil {
		t.Fatal("Start must not succeed after wildcard proxy boot failure")
	}
}

func TestProductionSpecificTrustedProxiesBoots(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "192.0.2.1")
	app := prodProxyApp(t)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
}

func TestProductionEmptyTrustedProxiesBoots(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")
	app := prodProxyApp(t)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
}

func TestDevelopmentWildcardTrustedProxiesBoots(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("TRUSTED_PROXIES", "*")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedTrustedProxiesFailsBootstrap(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("TRUSTED_PROXIES", "not-a-proxy")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err == nil {
		t.Fatal("malformed TRUSTED_PROXIES was accepted")
	}
}
