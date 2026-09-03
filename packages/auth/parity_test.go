package auth_test

import (
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/auth/totp"
	"github.com/zatrano/framework/packages/encryption"
	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/session"
)

func TestValidateOnceLoginUsingIDAndViaRemember(t *testing.T) {
	hash, err := hashing.Hash("secret")
	if err != nil {
		t.Fatal(err)
	}
	provider := newMemoryUserProvider()
	user, err := provider.Create(map[string]any{
		"email": "once@zatrano.test", "password": hash, "name": "Once",
	})
	if err != nil {
		t.Fatal(err)
	}
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))

	ok, err := manager.Validate(map[string]string{"email": "once@zatrano.test", "password": "secret"})
	if err != nil || !ok {
		t.Fatalf("validate=%v err=%v", ok, err)
	}
	ok, err = manager.Validate(map[string]string{"email": "once@zatrano.test", "password": "bad"})
	if err != nil || ok {
		t.Fatalf("expected invalid credentials")
	}

	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	req.SetSession(&memSession{data: map[string]any{}})
	if err := manager.Once(req, user); err != nil {
		t.Fatal(err)
	}
	if manager.User(req) == nil {
		t.Fatal("once user missing")
	}
	if req.Session().Get("auth_user_id") != nil || req.Session().Get(auth.SessionUserKey("web")) != nil {
		t.Fatal("once must not write session")
	}

	req2 := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	req2.SetSession(&memSession{data: map[string]any{}})
	if err := manager.LoginUsingID(req2, user.AuthID()); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(manager.ID(req2)) != fmt.Sprint(user.AuthID()) {
		t.Fatal("login using id failed")
	}

	req3 := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	req3.SetSession(&memSession{data: map[string]any{}})
	if err := manager.Login(req3, user, true); err != nil {
		t.Fatal(err)
	}
	var cookieValue string
	for _, c := range req3.Cookies().Apply() {
		if c.Name == "remember_web" {
			cookieValue = c.Value
		}
	}
	raw := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.AddCookie(&stdhttp.Cookie{Name: "remember_web", Value: cookieValue})
	req4 := http.NewRequest(raw)
	req4.SetSession(&memSession{data: map[string]any{}})
	if manager.User(req4) == nil || !manager.ViaRemember(req4) {
		t.Fatal("via remember expected")
	}
}

func TestLogoutOtherDevicesDestroysForeignSessions(t *testing.T) {
	dir := t.TempDir()
	sessMgr := session.NewManager(dir, 120)
	hash, _ := hashing.Hash("secret")
	provider := newMemoryUserProvider()
	user, _ := provider.Create(map[string]any{"email": "devices@zatrano.test", "password": hash})
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))
	manager.SetSessionManager(sessMgr)

	other, err := sessMgr.Start("")
	if err != nil {
		t.Fatal(err)
	}
	other.Put(auth.SessionUserKey("web"), fmt.Sprint(user.AuthID()))
	if err := sessMgr.Save(other); err != nil {
		t.Fatal(err)
	}

	current, err := sessMgr.Start("")
	if err != nil {
		t.Fatal(err)
	}
	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodPost, "/", nil))
	req.SetSession(current)
	if err := manager.Login(req, user); err != nil {
		t.Fatal(err)
	}
	_ = sessMgr.Save(current)

	if err := manager.LogoutOtherDevices(req, "secret"); err != nil {
		t.Fatal(err)
	}
	// foreign session file should be gone
	bag, err := sessMgr.Start(other.ID())
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(bag.Get(auth.SessionUserKey("web"))) == fmt.Sprint(user.AuthID()) {
		t.Fatal("other session should have been destroyed")
	}
}

func TestLockoutAfterFailedAttempts(t *testing.T) {
	hash, _ := hashing.Hash("secret")
	provider := newMemoryUserProvider()
	_, _ = provider.Create(map[string]any{"email": "lock@zatrano.test", "password": hash})
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))
	manager.SetLockout(3, time.Minute)

	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodPost, "/", nil))
	req.SetSession(&memSession{data: map[string]any{}})
	for i := 0; i < 3; i++ {
		_, err := manager.Attempt(req, map[string]string{"email": "lock@zatrano.test", "password": "bad"})
		if i == 2 && !errors.Is(err, auth.ErrLockout) {
			t.Fatalf("expected lockout, got %v", err)
		}
	}
}

func TestTwoFactorChallengeFlow(t *testing.T) {
	hash, _ := hashing.Hash("secret")
	provider := newMemoryUserProvider()
	user, _ := provider.Create(map[string]any{"email": "mfa@zatrano.test", "password": hash, "name": "MFA"})
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))

	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodPost, "/", nil))
	req.SetSession(&memSession{data: map[string]any{}})
	if err := manager.Login(req, user); err != nil {
		t.Fatal(err)
	}
	secret, _, _, err := manager.EnableTwoFactor(user)
	if err != nil {
		t.Fatal(err)
	}
	code, err := totp.Code(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.ConfirmTwoFactor(user, code); err != nil {
		t.Fatal(err)
	}
	_ = manager.Logout(req)

	ok, err := manager.Attempt(req, map[string]string{"email": "mfa@zatrano.test", "password": "secret"})
	if ok || !errors.Is(err, auth.ErrTwoFactorRequired) {
		t.Fatalf("expected two-factor required, ok=%v err=%v", ok, err)
	}
	code, err = totp.Code(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ok, err = manager.ChallengeTwoFactor(req, code)
	if err != nil || !ok {
		t.Fatalf("challenge failed: %v", err)
	}
	if !manager.Check(req) {
		t.Fatal("expected authenticated after challenge")
	}
}

func TestShouldUseAndOnceBasic(t *testing.T) {
	hash, _ := hashing.Hash("secret")
	provider := newMemoryUserProvider()
	_, _ = provider.Create(map[string]any{"email": "basic@zatrano.test", "password": hash})
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))
	manager.Extend("api", auth.NewGuard("api", provider))
	manager.ShouldUse("api")
	if manager.GetDefaultDriver() != "api" {
		t.Fatalf("default=%q", manager.GetDefaultDriver())
	}
	manager.SetDefaultDriver("web")
	if manager.GetDefaultDriver() != "web" {
		t.Fatal("set default driver failed")
	}

	raw := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.SetBasicAuth("basic@zatrano.test", "secret")
	req := http.NewRequest(raw)
	req.SetSession(&memSession{data: map[string]any{}})
	if !manager.OnceBasic(req) {
		t.Fatal("once basic failed")
	}
	if manager.User(req) == nil {
		t.Fatal("expected user from once basic")
	}
	if req.Session().Get("auth_user_id") != nil || req.Session().Get(auth.SessionUserKey("web")) != nil {
		t.Fatal("once basic must not write session")
	}
}

func TestTwoFactorSecretsAreEncrypted(t *testing.T) {
	hash, _ := hashing.Hash("secret")
	provider := newMemoryUserProvider()
	user, _ := provider.Create(map[string]any{"email": "enc@zatrano.test", "password": hash})
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))
	crypt, err := encryption.New("zatrano-dev-key")
	if err != nil {
		t.Fatal(err)
	}
	manager.SetEncrypter(crypt)

	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	req.SetSession(&memSession{data: map[string]any{}})
	_ = manager.Login(req, user)
	secret, _, _, err := manager.EnableTwoFactor(user)
	if err != nil {
		t.Fatal(err)
	}
	stored := fmt.Sprint(user.(*auth.GenericUser).Get("two_factor_secret"))
	if stored == "" || stored == secret {
		t.Fatalf("expected encrypted storage, secret=%q stored=%q", secret, stored)
	}
	code, err := totp.Code(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.ConfirmTwoFactor(user, code); err != nil {
		t.Fatal(err)
	}
}

func TestMultiGuardSessionIsolation(t *testing.T) {
	hash, _ := hashing.Hash("secret")
	provider := newMemoryUserProvider()
	webUser, _ := provider.Create(map[string]any{"email": "web@zatrano.test", "password": hash})
	apiUser, _ := provider.Create(map[string]any{"email": "api@zatrano.test", "password": hash})
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))
	manager.Extend("api", auth.NewGuard("api", provider))

	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	req.SetSession(&memSession{data: map[string]any{}})
	if err := manager.Guard("web").Login(req, webUser); err != nil {
		t.Fatal(err)
	}
	if err := manager.Guard("api").Login(req, apiUser); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(manager.Guard("web").ID(req)) != fmt.Sprint(webUser.AuthID()) {
		t.Fatal("web guard polluted")
	}
	if fmt.Sprint(manager.Guard("api").ID(req)) != fmt.Sprint(apiUser.AuthID()) {
		t.Fatal("api guard polluted")
	}
	if req.Session().Get(auth.SessionUserKey("web")) == nil || req.Session().Get(auth.SessionUserKey("api")) == nil {
		t.Fatal("expected per-guard session keys")
	}
}

type recordingDispatcher struct {
	events []string
}

func (d *recordingDispatcher) Dispatch(name string, event any) error {
	d.events = append(d.events, name)
	return nil
}

func TestMarkEmailAsVerifiedDispatchesEvent(t *testing.T) {
	provider := newMemoryUserProvider()
	user, _ := provider.Create(map[string]any{"email": "verify@zatrano.test", "password": "x"})
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))
	disp := &recordingDispatcher{}
	manager.SetDispatcher(disp)

	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	if err := manager.MarkEmailAsVerified(req, user); err != nil {
		t.Fatal(err)
	}
	if !auth.HasVerifiedEmail(user) {
		t.Fatal("expected verified")
	}
	found := false
	for _, name := range disp.events {
		if name == auth.EventVerified {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s, got %#v", auth.EventVerified, disp.events)
	}
}

type memAttemptCounter struct {
	data map[string]any
}

func (m *memAttemptCounter) Get(key string) (any, bool) {
	v, ok := m.data[key]
	return v, ok
}
func (m *memAttemptCounter) Put(key string, value any, ttl time.Duration) error {
	if m.data == nil {
		m.data = map[string]any{}
	}
	m.data[key] = value
	return nil
}
func (m *memAttemptCounter) Forget(key string) error {
	delete(m.data, key)
	return nil
}

func TestLockoutUsesSharedCache(t *testing.T) {
	hash, _ := hashing.Hash("secret")
	provider := newMemoryUserProvider()
	_, _ = provider.Create(map[string]any{"email": "cachelock@zatrano.test", "password": hash})
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))
	manager.SetLockout(2, time.Minute)
	counter := &memAttemptCounter{data: map[string]any{}}
	manager.SetLockoutCache(counter)

	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodPost, "/", nil))
	req.SetSession(&memSession{data: map[string]any{}})
	_, _ = manager.Attempt(req, map[string]string{"email": "cachelock@zatrano.test", "password": "bad"})
	_, err := manager.Attempt(req, map[string]string{"email": "cachelock@zatrano.test", "password": "bad"})
	if !errors.Is(err, auth.ErrLockout) {
		t.Fatalf("expected lockout, got %v", err)
	}
	if len(counter.data) == 0 {
		t.Fatal("expected cache writes")
	}
}

func TestTrustedDeviceSkipsTwoFactorChallenge(t *testing.T) {
	hash, _ := hashing.Hash("secret")
	provider := newMemoryUserProvider()
	user, _ := provider.Create(map[string]any{"email": "device@zatrano.test", "password": hash, "name": "Device"})
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))
	crypt, err := encryption.New("zatrano-dev-key")
	if err != nil {
		t.Fatal(err)
	}
	manager.SetEncrypter(crypt)
	manager.SetRememberDeviceDays(30)

	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodPost, "/", nil))
	req.SetSession(&memSession{data: map[string]any{}})
	if err := manager.Login(req, user); err != nil {
		t.Fatal(err)
	}
	secret, _, _, err := manager.EnableTwoFactor(user)
	if err != nil {
		t.Fatal(err)
	}
	code, err := totp.Code(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.ConfirmTwoFactor(user, code); err != nil {
		t.Fatal(err)
	}
	_ = manager.Logout(req)

	// Pending challenge then remember device
	ok, err := manager.Attempt(req, map[string]string{"email": "device@zatrano.test", "password": "secret"})
	if ok || !errors.Is(err, auth.ErrTwoFactorRequired) {
		t.Fatalf("expected 2fa required, ok=%v err=%v", ok, err)
	}
	code, err = totp.Code(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ok, err = manager.ChallengeTwoFactor(req, code, true)
	if err != nil || !ok {
		t.Fatalf("challenge: %v", err)
	}
	var deviceCookie string
	for _, c := range req.Cookies().Apply() {
		if c.Name == "2fa_device_web" {
			deviceCookie = c.Value
		}
	}
	if deviceCookie == "" {
		t.Fatal("expected device cookie")
	}
	_ = manager.Logout(req)

	raw := httptest.NewRequest(stdhttp.MethodPost, "/", nil)
	raw.AddCookie(&stdhttp.Cookie{Name: "2fa_device_web", Value: deviceCookie})
	req2 := http.NewRequest(raw)
	req2.SetSession(&memSession{data: map[string]any{}})
	ok, err = manager.Attempt(req2, map[string]string{"email": "device@zatrano.test", "password": "secret"})
	if err != nil || !ok {
		t.Fatalf("trusted device should skip 2fa: ok=%v err=%v", ok, err)
	}
	if !manager.Check(req2) {
		t.Fatal("expected authenticated")
	}
}

func TestChallengePreservesLoginRememberFlag(t *testing.T) {
	hash, _ := hashing.Hash("secret")
	provider := newMemoryUserProvider()
	user, _ := provider.Create(map[string]any{"email": "rem2fa@zatrano.test", "password": hash})
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))

	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodPost, "/", nil))
	req.SetSession(&memSession{data: map[string]any{}})
	_ = manager.Login(req, user)
	secret, _, _, _ := manager.EnableTwoFactor(user)
	code, _ := totp.Code(secret, time.Now())
	_ = manager.ConfirmTwoFactor(user, code)
	_ = manager.Logout(req)

	ok, err := manager.Attempt(req, map[string]string{"email": "rem2fa@zatrano.test", "password": "secret"}, true)
	if ok || !errors.Is(err, auth.ErrTwoFactorRequired) {
		t.Fatalf("expected 2fa, ok=%v err=%v", ok, err)
	}
	code, _ = totp.Code(secret, time.Now())
	ok, err = manager.ChallengeTwoFactor(req, code)
	if err != nil || !ok {
		t.Fatalf("challenge: %v", err)
	}
	found := false
	for _, c := range req.Cookies().Apply() {
		if c.Name == "remember_web" && c.Value != "" && c.MaxAge >= 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("expected remember cookie after 2fa with remember=true login")
	}
}
