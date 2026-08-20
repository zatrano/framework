package auth_test

import (
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/session"
)

type memoryRememberProvider struct {
	users map[string]*auth.GenericUser
}

func newMemoryRememberProvider(users ...*auth.GenericUser) *memoryRememberProvider {
	p := &memoryRememberProvider{users: map[string]*auth.GenericUser{}}
	for _, u := range users {
		p.users[fmt.Sprint(u.AuthID())] = u
	}
	return p
}

func (p *memoryRememberProvider) RetrieveByID(id any) (auth.Authenticatable, error) {
	u, ok := p.users[fmt.Sprint(id)]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (p *memoryRememberProvider) RetrieveByCredentials(credentials map[string]string) (auth.Authenticatable, error) {
	email := credentials["email"]
	for _, u := range p.users {
		if fmt.Sprint(u.Get("email")) == email {
			return u, nil
		}
	}
	return nil, nil
}

func (p *memoryRememberProvider) ValidateCredentials(user auth.Authenticatable, credentials map[string]string) bool {
	return hashing.Check(credentials["password"], user.AuthPassword())
}

func (p *memoryRememberProvider) RetrieveByToken(id, token string) (auth.Authenticatable, error) {
	u, ok := p.users[fmt.Sprint(id)]
	if !ok {
		return nil, nil
	}
	stored := strings.TrimSpace(fmt.Sprint(u.Get("remember_token")))
	if stored == "" || stored != token {
		return nil, nil
	}
	return u, nil
}

func (p *memoryRememberProvider) UpdateRememberToken(user auth.Authenticatable, token string) error {
	u, ok := p.users[fmt.Sprint(user.AuthID())]
	if !ok {
		return fmt.Errorf("user not found")
	}
	if token == "" {
		u.Attributes["remember_token"] = nil
		return nil
	}
	u.Attributes["remember_token"] = token
	return nil
}

func newAuthRequest(path string) *http.Request {
	raw := httptest.NewRequest(stdhttp.MethodGet, path, nil)
	req := http.NewRequest(raw)
	req.SetSession(&memSession{data: map[string]any{}})
	return req
}

func TestRememberMeLoginRestoreAndLogout(t *testing.T) {
	hash, err := hashing.Hash("secret")
	if err != nil {
		t.Fatal(err)
	}
	user := &auth.GenericUser{Attributes: map[string]any{
		"id":       7,
		"email":    "ada@zatrano.test",
		"password": hash,
	}}
	provider := newMemoryRememberProvider(user)
	guard := auth.NewGuard("web", provider)

	req := newAuthRequest("/")
	ok, err := guard.Attempt(req, map[string]string{
		"email":    "ada@zatrano.test",
		"password": "secret",
	}, true)
	if err != nil || !ok {
		t.Fatalf("attempt ok=%v err=%v", ok, err)
	}

	cookies := req.Cookies().Apply()
	var rememberValue string
	for _, c := range cookies {
		if c.Name == "remember_web" {
			rememberValue = c.Value
			break
		}
	}
	if rememberValue == "" || !strings.Contains(rememberValue, "|") {
		t.Fatalf("remember cookie missing: %#v", cookies)
	}
	storedToken := fmt.Sprint(user.Get("remember_token"))
	if storedToken == "" || storedToken == "<nil>" {
		t.Fatal("remember token not stored on user")
	}

	// New request with empty session but remember cookie restores login.
	raw2 := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	raw2.AddCookie(&stdhttp.Cookie{Name: "remember_web", Value: rememberValue})
	req2 := http.NewRequest(raw2)
	req2.SetSession(&memSession{data: map[string]any{}})
	restored := guard.User(req2)
	if restored == nil || fmt.Sprint(restored.AuthID()) != "7" {
		t.Fatalf("restored=%v", restored)
	}
	if sessID := fmt.Sprint(req2.Session().Get(auth.SessionUserKey("web"))); sessID != "7" {
		t.Fatalf("session not rehydrated: %q", sessID)
	}

	if err := guard.Logout(req2); err != nil {
		t.Fatal(err)
	}
	if guard.User(req2) != nil {
		t.Fatal("expected nil user after logout")
	}
	if fmt.Sprint(user.Get("remember_token")) != "<nil>" && user.Get("remember_token") != nil {
		t.Fatalf("token not cleared: %#v", user.Get("remember_token"))
	}

	forget := false
	for _, c := range req2.Cookies().Apply() {
		if c.Name == "remember_web" && c.MaxAge < 0 {
			forget = true
		}
	}
	if !forget {
		t.Fatal("expected forget cookie after logout")
	}

	// Cookie no longer valid after token clear.
	raw3 := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	raw3.AddCookie(&stdhttp.Cookie{Name: "remember_web", Value: rememberValue})
	req3 := http.NewRequest(raw3)
	req3.SetSession(&memSession{data: map[string]any{}})
	if guard.User(req3) != nil {
		t.Fatal("stale remember cookie should not authenticate")
	}
}

func TestRememberCookieEncodeDecode(t *testing.T) {
	hash, err := hashing.Hash("x")
	if err != nil {
		t.Fatal(err)
	}
	user := &auth.GenericUser{Attributes: map[string]any{"id": 1, "email": "a@b.c", "password": hash}}
	provider := newMemoryRememberProvider(user)
	guard := auth.NewGuard("api", provider)

	req := newAuthRequest("/")
	if err := guard.Login(req, user, false); err != nil {
		t.Fatal(err)
	}
	for _, c := range req.Cookies().Apply() {
		if strings.HasPrefix(c.Name, "remember_") {
			t.Fatal("remember cookie should not queue when remember=false")
		}
	}
}

func TestInvalidRememberCookieIgnored(t *testing.T) {
	provider := newMemoryRememberProvider(&auth.GenericUser{Attributes: map[string]any{
		"id": 1, "email": "a@b.c", "password": "x", "remember_token": "good",
	}})
	guard := auth.NewGuard("web", provider)

	raw := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.AddCookie(&stdhttp.Cookie{Name: "remember_web", Value: "1|bad"})
	req := http.NewRequest(raw)
	req.SetSession(&memSession{data: map[string]any{}})
	if guard.User(req) != nil {
		t.Fatal("bad token should not authenticate")
	}
}

func TestRememberMeCookieSecurity(t *testing.T) {
	hash, err := hashing.Hash("secret")
	if err != nil {
		t.Fatal(err)
	}
	user := &auth.GenericUser{Attributes: map[string]any{
		"id":       9,
		"email":    "secure@zatrano.test",
		"password": hash,
	}}
	provider := newMemoryRememberProvider(user)
	guard := auth.NewGuard("web", provider)

	req := newAuthRequest("/")
	req.Set("_forwarded_proto", "https")
	ok, err := guard.Attempt(req, map[string]string{
		"email":    "secure@zatrano.test",
		"password": "secret",
	}, true)
	if err != nil || !ok {
		t.Fatalf("attempt ok=%v err=%v", ok, err)
	}

	var found *stdhttp.Cookie
	for _, c := range req.Cookies().Apply() {
		if c.Name == "remember_web" {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("remember cookie missing")
	}
	if !found.HttpOnly {
		t.Fatal("remember cookie must be HttpOnly")
	}
	if found.SameSite != stdhttp.SameSiteLaxMode {
		t.Fatalf("SameSite=%v want Lax", found.SameSite)
	}
	if found.Path != "/" {
		t.Fatalf("Path=%q", found.Path)
	}
	if !found.Secure {
		t.Fatal("remember cookie must be Secure on HTTPS request")
	}
}

func TestRememberHashesEqual(t *testing.T) {
	a := auth.HashRememberToken("token-a")
	b := auth.HashRememberToken("token-a")
	c := auth.HashRememberToken("token-b")
	if !auth.RememberHashesEqual(a, b) {
		t.Fatal("identical digests must match")
	}
	if auth.RememberHashesEqual(a, c) {
		t.Fatal("different digests must not match")
	}
	if auth.RememberHashesEqual(a, a[:len(a)-1]) {
		t.Fatal("length mismatch must not match")
	}
}

func TestSessionFixation(t *testing.T) {
	dir := t.TempDir()
	sessMgr := session.NewManager(dir, 120)
	bag, err := sessMgr.Start("")
	if err != nil {
		t.Fatal(err)
	}
	oldID := bag.ID()

	hash, err := hashing.Hash("secret")
	if err != nil {
		t.Fatal(err)
	}
	user := &auth.GenericUser{Attributes: map[string]any{
		"id": 42, "email": "fix@zatrano.test", "password": hash,
	}}
	guard := auth.NewGuard("web", newMemoryRememberProvider(user))

	raw := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req := http.NewRequest(raw)
	req.SetSession(bag)

	if err := guard.Login(req, user); err != nil {
		t.Fatal(err)
	}
	if bag.ID() == oldID {
		t.Fatal("login must regenerate session id (session fixation)")
	}
	if fmt.Sprint(bag.Get(auth.SessionUserKey("web"))) != "42" {
		t.Fatal("user id must survive regenerate")
	}
}
