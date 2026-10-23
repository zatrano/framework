package kernel_test

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel"
	zhttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/middleware"
	"github.com/zatrano/framework/v3/core/kernel/routing"
)

const strongTestKey = "zatrano-ci-test-key-not-prod!!!!"

// productionEnv sets a production environment that boots, so each test
// changes exactly one setting.
func productionEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strongTestKey)
	for _, key := range []string{
		"APP_URL", "COOKIE_SECURE", "SESSION_SECURE", "CSRF_DISABLED", "TRUSTED_PROXIES",
		"APP_KEY_WEAK_IN_PRODUCTION", "APP_DEBUG_IN_PRODUCTION", "COOKIE_INSECURE_IN_PRODUCTION",
		"CORS_ALLOWED_ORIGINS", "CORS_ALLOW_CREDENTIALS", "CORS_NULL_ORIGIN_WITH_CREDENTIALS",
		"HTTP_POOL_REQUESTS",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func bootErr(t *testing.T, setup func(*kernel.Application)) error {
	t.Helper()
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if setup != nil {
		setup(app)
	}
	return app.Bootstrap()
}

func expectRefusal(t *testing.T, err error, mention string) {
	t.Helper()
	if err == nil {
		t.Fatalf("production boot accepted the configuration (want a refusal mentioning %s)", mention)
	}
	if !strings.Contains(err.Error(), "refusing to boot") && !strings.Contains(err.Error(), "cors:") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), mention) {
		t.Fatalf("refusal %q does not name %s", err, mention)
	}
}

func TestProductionBaselineBoots(t *testing.T) {
	productionEnv(t)
	if err := bootErr(t, nil); err != nil {
		t.Fatal(err)
	}
}

func TestProductionRejectsWeakAppKey(t *testing.T) {
	weak := map[string]string{
		"one byte":       strings.Repeat("s", 32),
		"two bytes":      strings.Repeat("ab", 16),
		"zero key":       "base64:" + base64.StdEncoding.EncodeToString(make([]byte, 32)),
		"short sequence": strings.Repeat("abcdefg", 5)[:32],
	}
	for name, key := range weak {
		t.Run(name, func(t *testing.T) {
			productionEnv(t)
			t.Setenv("APP_KEY", key)
			expectRefusal(t, bootErr(t, nil), "APP_KEY_WEAK_IN_PRODUCTION")
		})
	}
}

func TestProductionWeakAppKeyOptOut(t *testing.T) {
	productionEnv(t)
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	t.Setenv("APP_KEY_WEAK_IN_PRODUCTION", "allow")
	if err := bootErr(t, nil); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_KEY_WEAK_IN_PRODUCTION", "true")
	expectRefusal(t, bootErr(t, nil), "APP_KEY")
}

func TestProductionWeakAppKeyOptOutKeepsLengthCheck(t *testing.T) {
	productionEnv(t)
	t.Setenv("APP_KEY", "short")
	t.Setenv("APP_KEY_WEAK_IN_PRODUCTION", "allow")
	if err := bootErr(t, nil); err == nil {
		t.Fatal("the weak-key opt-out also skipped the length check")
	}
}

func TestProductionRejectsDebug(t *testing.T) {
	productionEnv(t)
	t.Setenv("APP_DEBUG", "true")
	expectRefusal(t, bootErr(t, nil), "APP_DEBUG")
}

func TestProductionDebugOptOut(t *testing.T) {
	productionEnv(t)
	t.Setenv("APP_DEBUG", "true")
	t.Setenv("APP_DEBUG_IN_PRODUCTION", "allow")
	if err := bootErr(t, nil); err != nil {
		t.Fatal(err)
	}
}

func TestProductionDebugDefaultsOff(t *testing.T) {
	productionEnv(t)
	t.Setenv("APP_DEBUG", "")
	os.Unsetenv("APP_DEBUG")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.IsDebug() {
		t.Fatal("unset APP_DEBUG is on in production")
	}
}

func TestLocalDebugDefaultsOn(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("APP_DEBUG", "")
	os.Unsetenv("APP_DEBUG")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if !app.IsDebug() {
		t.Fatal("unset APP_DEBUG is off in local")
	}
}

func TestProductionRejectsInsecureCookiesOverHTTPS(t *testing.T) {
	for _, key := range []string{"COOKIE_SECURE", "SESSION_SECURE"} {
		t.Run(key, func(t *testing.T) {
			productionEnv(t)
			t.Setenv("APP_URL", "https://example.com")
			t.Setenv(key, "false")
			expectRefusal(t, bootErr(t, nil), key)
		})
	}
}

func TestProductionInsecureCookieOptOut(t *testing.T) {
	productionEnv(t)
	t.Setenv("APP_URL", "https://example.com")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_INSECURE_IN_PRODUCTION", "allow")
	if err := bootErr(t, nil); err != nil {
		t.Fatal(err)
	}
}

func TestProductionPlainHTTPCookiesBoot(t *testing.T) {
	productionEnv(t)
	t.Setenv("APP_URL", "http://intranet.local")
	t.Setenv("COOKIE_SECURE", "false")
	if err := bootErr(t, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCORSRejectsCredentialsWithAnyOrigin(t *testing.T) {
	productionEnv(t)
	t.Setenv("CORS_ALLOW_CREDENTIALS", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	expectRefusal(t, bootErr(t, nil), "wildcard")
}

func TestCORSRejectsCredentialsWithNullOrigin(t *testing.T) {
	productionEnv(t)
	t.Setenv("CORS_ALLOW_CREDENTIALS", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com, null")
	expectRefusal(t, bootErr(t, nil), "CORS_NULL_ORIGIN_WITH_CREDENTIALS")
	t.Setenv("CORS_NULL_ORIGIN_WITH_CREDENTIALS", "allow")
	if err := bootErr(t, nil); err != nil {
		t.Fatal(err)
	}
}

func TestProductionRejectsTrustAllProxyNetworks(t *testing.T) {
	for _, value := range []string{
		"*", "0.0.0.0/0", "::/0", "10.0.0.0/8, 0.0.0.0/0",
		"0.0.0.0/1, 128.0.0.0/1", "0.0.0.0/1", "::ffff:0:0/96", "::/1, 8000::/1",
	} {
		t.Run(value, func(t *testing.T) {
			productionEnv(t)
			t.Setenv("TRUSTED_PROXIES", value)
			expectRefusal(t, bootErr(t, nil), "TRUSTED_PROXIES")
		})
	}
}

func TestProductionPrivateProxyNetworksBoot(t *testing.T) {
	productionEnv(t)
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7")
	if err := bootErr(t, nil); err != nil {
		t.Fatal(err)
	}
}

// APP_ENV=prod is production; the short spelling does not skip the checks.
func TestProdSpellingRunsProductionChecks(t *testing.T) {
	productionEnv(t)
	t.Setenv("APP_ENV", "prod")
	t.Setenv("APP_DEBUG", "true")
	expectRefusal(t, bootErr(t, nil), "APP_DEBUG")
}

// Without TRUSTED_PROXIES no peer can choose the client IP.
func TestForwardedHeadersIgnoredWithoutTrustedProxies(t *testing.T) {
	productionEnv(t)
	var seen string
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ip", func(req *zhttp.Request) *zhttp.Response {
		seen = req.IP()
		return zhttp.Text(req.IP())
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	er := serveHandle(t, app, "GET", "/ip", map[string]string{
		"X-Forwarded-For": "203.0.113.9",
		"X-Real-IP":       "203.0.113.10",
		"Forwarded":       "for=203.0.113.11",
	}, "")
	if er.Status != 200 {
		t.Fatalf("status=%d", er.Status)
	}
	if strings.HasPrefix(seen, "203.0.113.") {
		t.Fatalf("IP()=%q came from a forwarding header with no trusted proxy", seen)
	}
}

type sessionLikeBridge struct{}

func (sessionLikeBridge) Middleware() []any {
	return []any{func(next routing.HandlerFunc) routing.HandlerFunc { return next }}
}

func (sessionLikeBridge) Finalize(req any, resp any) any { return resp }

func sessionApp(t *testing.T, mode string) error {
	t.Helper()
	productionEnv(t)
	return bootErr(t, func(app *kernel.Application) {
		app.SetHTTPBridge(sessionLikeBridge{})
		app.Router().Get("/", func(req *zhttp.Request) *zhttp.Response { return zhttp.Text("ok") })
		switch mode {
		case "optout":
			t.Setenv("CSRF_DISABLED", "true")
		case "installed":
			app.Router().Use(middleware.CSRF)
		}
	})
}

func TestProductionRejectsSessionsWithoutCSRF(t *testing.T) {
	expectRefusal(t, sessionApp(t, "refuse"), "CSRF_DISABLED")
	if err := sessionApp(t, "optout"); err != nil {
		t.Fatalf("CSRF_DISABLED=true: %v", err)
	}
	if err := sessionApp(t, "installed"); err != nil {
		t.Fatalf("CSRF installed: %v", err)
	}
}

// CSRF that wraps a handler outside this application does not count for it.
func TestCSRFElsewhereInProcessDoesNotSatisfyTheCheck(t *testing.T) {
	_ = middleware.CSRF(func(*zhttp.Request) *zhttp.Response { return zhttp.Text("other") })
	if !middleware.CSRFInstalled() {
		t.Fatal("CSRF wrap was not counted")
	}
	expectRefusal(t, sessionApp(t, "refuse"), "CSRF_DISABLED")
}
