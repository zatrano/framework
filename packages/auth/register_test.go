package auth_test

import (
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/http"
)

type memoryUserProvider struct {
	nextID int
	users  map[string]*auth.GenericUser
}

func newMemoryUserProvider() *memoryUserProvider {
	return &memoryUserProvider{
		nextID: 1,
		users:  map[string]*auth.GenericUser{},
	}
}

func (p *memoryUserProvider) RetrieveByID(id any) (auth.Authenticatable, error) {
	u, ok := p.users[fmt.Sprint(id)]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (p *memoryUserProvider) RetrieveByCredentials(credentials map[string]string) (auth.Authenticatable, error) {
	email := credentials["email"]
	for _, u := range p.users {
		if fmt.Sprint(u.Get("email")) == email {
			return u, nil
		}
	}
	return nil, nil
}

func (p *memoryUserProvider) ValidateCredentials(user auth.Authenticatable, credentials map[string]string) bool {
	return hashing.Check(credentials["password"], user.AuthPassword())
}

func (p *memoryUserProvider) Create(attrs map[string]any) (auth.Authenticatable, error) {
	id := p.nextID
	p.nextID++
	attrs["id"] = id
	u := &auth.GenericUser{Attributes: attrs}
	p.users[fmt.Sprint(id)] = u
	return u, nil
}

func (p *memoryUserProvider) UpdatePassword(email, hashedPassword string) error {
	for _, u := range p.users {
		if fmt.Sprint(u.Get("email")) == email {
			u.Attributes["password"] = hashedPassword
			return nil
		}
	}
	return fmt.Errorf("user not found")
}

func (p *memoryUserProvider) UpdateAttributes(id any, attrs map[string]any) error {
	u, ok := p.users[fmt.Sprint(id)]
	if !ok {
		return fmt.Errorf("user not found")
	}
	for k, v := range attrs {
		u.Attributes[k] = v
	}
	return nil
}

func (p *memoryUserProvider) RetrieveByToken(id, token string) (auth.Authenticatable, error) {
	u, ok := p.users[fmt.Sprint(id)]
	if !ok {
		return nil, nil
	}
	if fmt.Sprint(u.Get("remember_token")) != token {
		return nil, nil
	}
	return u, nil
}

func (p *memoryUserProvider) UpdateRememberToken(user auth.Authenticatable, token string) error {
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

func TestRegisterAndChangePassword(t *testing.T) {
	provider := newMemoryUserProvider()
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))

	raw := httptest.NewRequest(stdhttp.MethodPost, "/register", nil)
	req := http.NewRequest(raw)
	req.SetSession(&memSession{data: map[string]any{}})

	user, err := manager.Register(req, map[string]any{
		"name":     "Ada",
		"email":    "ada@zatrano.test",
		"password": "secret1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if user == nil || fmt.Sprint(user.AuthID()) != "1" {
		t.Fatalf("user=%v", user)
	}
	if !manager.Check(req) {
		t.Fatal("expected logged in after register")
	}
	if !hashing.Check("secret1", user.AuthPassword()) {
		t.Fatal("password not hashed")
	}

	_, err = manager.Register(req, map[string]any{
		"name": "Dup", "email": "ada@zatrano.test", "password": "secret1",
	}, false)
	if err == nil || err.Error() != "email already taken" {
		t.Fatalf("expected email already taken, got %v", err)
	}

	if err := manager.ChangePassword(req, "wrong", "secret2"); err == nil {
		t.Fatal("expected incorrect current password")
	}
	if err := manager.ChangePassword(req, "secret1", "secret2"); err != nil {
		t.Fatal(err)
	}
	if !hashing.Check("secret2", user.AuthPassword()) {
		t.Fatal("password not updated")
	}
}

func TestRememberTokenIsHashedInProvider(t *testing.T) {
	provider := newMemoryUserProvider()
	user, _ := provider.Create(map[string]any{
		"email": "r@zatrano.test", "password": "x",
	})
	guard := auth.NewGuard("web", provider)
	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	req.SetSession(&memSession{data: map[string]any{}})
	if err := guard.Login(req, user, true); err != nil {
		t.Fatal(err)
	}
	stored := fmt.Sprint(user.(*auth.GenericUser).Get("remember_token"))
	if stored == "" || stored == "<nil>" {
		t.Fatal("missing token")
	}
	var plain string
	for _, c := range req.Cookies().Apply() {
		if c.Name == "remember_web" {
			_, plain, _ = decodeCookie(c.Value)
		}
	}
	if plain == "" || plain == stored {
		t.Fatalf("expected hashed storage, plain=%q stored=%q", plain, stored)
	}
	if auth.HashRememberToken(plain) != stored {
		t.Fatalf("hash mismatch plain=%q stored=%q", plain, stored)
	}
}

func decodeCookie(value string) (id, token string, ok bool) {
	for i := 0; i < len(value); i++ {
		if value[i] == '|' {
			return value[:i], value[i+1:], true
		}
	}
	return "", "", false
}
