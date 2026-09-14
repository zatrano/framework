package cookie_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel/cookie"
	"github.com/zatrano/framework/v2/kernel/encryption"
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

func TestJarForeverForgetGetHasClear(t *testing.T) {
	cookie.SetProductionPolicy(false)
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("SESSION_SECURE", "")
	jar := cookie.NewJar()
	jar.Forever("remember", "tok")
	jar.ForeverSecure("secure", "v", true)
	got := jar.Apply()
	if len(got) != 2 {
		t.Fatalf("queued=%d", len(got))
	}
	if got[0].MaxAge <= 0 || got[1].MaxAge <= 0 {
		t.Fatal("forever max-age")
	}
	if !got[1].Secure {
		t.Fatal("ForeverSecure")
	}
	jar.Forget("remember")
	after := jar.Apply()
	if len(after) != 2 {
		t.Fatalf("after forget apply=%d", len(after))
	}
	foundForget := false
	for _, c := range after {
		if c.Name == "remember" && c.MaxAge < 0 {
			foundForget = true
		}
	}
	if !foundForget {
		t.Fatal("expected forget cookie")
	}
	jar.Clear()
	if len(jar.Apply()) != 0 {
		t.Fatal("clear")
	}

	forever := cookie.ForeverCookie("a", "b")
	if forever.MaxAge <= 0 {
		t.Fatal("ForeverCookie")
	}
	forget := cookie.ForgetCookie("a")
	if forget.MaxAge != -1 {
		t.Fatal("ForgetCookie")
	}
	neg := cookie.Make("x", "y", -3)
	if neg.MaxAge != -1 {
		t.Fatal("negative minutes")
	}
	zero := cookie.Make("x", "y", 0)
	if zero.MaxAge != 0 {
		t.Fatal("session cookie")
	}

	raw := httptest.NewRequest("GET", "/", nil)
	raw.AddCookie(&stdhttp.Cookie{Name: "sid", Value: "abc"})
	if !cookie.Has(raw, "sid") || cookie.Get(raw, "sid") != "abc" {
		t.Fatal("get/has")
	}
	if cookie.Has(raw, "missing") {
		t.Fatal("missing has")
	}
	if cookie.Get(raw, "missing", "fb") != "fb" {
		t.Fatal("fallback")
	}
	if cookie.Get(raw, "missing") != "" {
		t.Fatal("empty fallback")
	}
}

func TestEncryptedCookieHelpers(t *testing.T) {
	enc, err := encryption.New(strings.Repeat("k", 32))
	if err != nil {
		t.Fatal(err)
	}
	jar := cookie.NewJar()
	if err := jar.QueueEncrypted(nil, "plain", "v", 5); err != nil {
		t.Fatal(err)
	}
	if jar.Apply()[0].Value != "v" {
		t.Fatal("nil encrypter queues plaintext")
	}
	jar.Clear()
	if err := jar.QueueEncrypted(enc, "sid", "secret", 5); err != nil {
		t.Fatal(err)
	}
	payload := jar.Apply()[0].Value
	if payload == "secret" {
		t.Fatal("expected ciphertext")
	}
	if got := cookie.DecryptValue(enc, payload); got != "secret" {
		t.Fatalf("decrypt=%q", got)
	}
	if cookie.DecryptValue(enc, "") != "" {
		t.Fatal("empty raw")
	}
	if cookie.DecryptValue(enc, "", "fb") != "fb" {
		t.Fatal("empty fallback")
	}
	if cookie.DecryptValue(nil, "raw") != "raw" {
		t.Fatal("nil enc")
	}
	if cookie.DecryptValue(enc, "nope", "fb") != "fb" {
		t.Fatal("bad payload fallback")
	}
	if cookie.DecryptValue(enc, "nope") != "" {
		t.Fatal("bad payload empty")
	}
}
