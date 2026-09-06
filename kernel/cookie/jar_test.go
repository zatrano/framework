package cookie_test

import (
	stdhttp "net/http"
	"testing"

	"github.com/zatrano/framework/v2/kernel/cookie"
)

func TestQueueSecureInProduction(t *testing.T) {
	cookie.SetProductionPolicy(true)
	t.Cleanup(func() { cookie.SetProductionPolicy(false) })
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("SESSION_SECURE", "")
	jar := cookie.NewJar()
	jar.Queue("sid", "1", 10)
	got := jar.Apply()
	if len(got) != 1 {
		t.Fatalf("cookies=%d", len(got))
	}
	c := got[0]
	if !c.Secure {
		t.Fatal("production framework jar cookie must be Secure")
	}
	if !c.HttpOnly {
		t.Fatal("HttpOnly must be preserved")
	}
	if c.SameSite != stdhttp.SameSiteLaxMode {
		t.Fatalf("SameSite=%v", c.SameSite)
	}
}

func TestQueueNotSecureInDevelopment(t *testing.T) {
	cookie.SetProductionPolicy(false)
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("SESSION_SECURE", "")
	jar := cookie.NewJar()
	jar.Queue("sid", "1", 10)
	c := jar.Apply()[0]
	if c.Secure {
		t.Fatal("development jar cookie must not force Secure")
	}
	if !c.HttpOnly || c.SameSite != stdhttp.SameSiteLaxMode {
		t.Fatalf("defaults changed: httponly=%v samesite=%v", c.HttpOnly, c.SameSite)
	}
}

func TestQueueRawDoesNotForceSecure(t *testing.T) {
	cookie.SetProductionPolicy(true)
	t.Cleanup(func() { cookie.SetProductionPolicy(false) })
	raw := &stdhttp.Cookie{Name: "pref", Value: "dark", Path: "/", Secure: false, HttpOnly: false}
	jar := cookie.NewJar()
	jar.QueueRaw(raw)
	c := jar.Apply()[0]
	if c.Secure || c.HttpOnly {
		t.Fatal("QueueRaw must not override explicit cookie attributes")
	}
}

func TestCookieSecureEnvStillAppliesInDevelopment(t *testing.T) {
	cookie.SetProductionPolicy(false)
	t.Setenv("COOKIE_SECURE", "true")
	if !cookie.SecureByDefault() {
		t.Fatal("COOKIE_SECURE=true must still enable Secure")
	}
}

func TestSecureByDefaultIgnoresProcessAppEnv(t *testing.T) {
	cookie.SetProductionPolicy(false)
	t.Cleanup(func() { cookie.SetProductionPolicy(false) })
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("SESSION_SECURE", "")
	if cookie.SecureByDefault() {
		t.Fatal("unbound production policy must not read process APP_ENV")
	}
	cookie.SetProductionPolicy(true)
	t.Setenv("APP_ENV", "local")
	if !cookie.SecureByDefault() {
		t.Fatal("bootstrapped production policy must ignore mutated APP_ENV")
	}
}
