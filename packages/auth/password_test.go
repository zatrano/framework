package auth_test

import (
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/http"
)

type stubPasswordProvider struct {
	users map[string]*auth.GenericUser
}

func (p *stubPasswordProvider) RetrieveByID(id any) (auth.Authenticatable, error) {
	return nil, nil
}

func (p *stubPasswordProvider) RetrieveByCredentials(credentials map[string]string) (auth.Authenticatable, error) {
	email := credentials["email"]
	if u, ok := p.users[email]; ok {
		return u, nil
	}
	return nil, nil
}

func (p *stubPasswordProvider) ValidateCredentials(user auth.Authenticatable, credentials map[string]string) bool {
	return true
}

func (p *stubPasswordProvider) UpdatePassword(email, hashedPassword string) error {
	if u, ok := p.users[email]; ok {
		u.Attributes["password"] = hashedPassword
	}
	return nil
}

func TestPasswordBrokerReset(t *testing.T) {
	provider := &stubPasswordProvider{users: map[string]*auth.GenericUser{
		"ada@zatrano.test": {Attributes: map[string]any{"id": 1, "email": "ada@zatrano.test", "password": "old"}},
	}}
	tokens := auth.NewMemoryTokenRepository(time.Hour)
	broker := auth.NewPasswordBroker(tokens, provider, time.Hour)

	token, err := broker.CreateToken("ada@zatrano.test")
	if err != nil {
		t.Fatal(err)
	}
	if !broker.TokenValid("ada@zatrano.test", token) {
		t.Fatal("expected valid token")
	}
	if err := broker.Reset("ada@zatrano.test", token, "secret123"); err != nil {
		t.Fatal(err)
	}
	if broker.TokenValid("ada@zatrano.test", token) {
		t.Fatal("token should be consumed")
	}
}

func TestPasswordBrokerUnknownUserAndThrottle(t *testing.T) {
	provider := &stubPasswordProvider{users: map[string]*auth.GenericUser{
		"ada@zatrano.test": {Attributes: map[string]any{"id": 1, "email": "ada@zatrano.test", "password": "old"}},
	}}
	tokens := auth.NewMemoryTokenRepository(time.Hour)
	broker := auth.NewPasswordBroker(tokens, provider, time.Hour)
	broker.SetThrottle(time.Minute)

	_, err := broker.CreateToken("missing@zatrano.test")
	if !errors.Is(err, auth.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
	if err := broker.SendResetLink("missing@zatrano.test", "http://example.test/reset"); err != nil {
		t.Fatalf("SendResetLink should hide missing users: %v", err)
	}

	token, err := broker.CreateToken("ada@zatrano.test")
	if err != nil || token == "" {
		t.Fatalf("create token: %v", err)
	}
	_, err = broker.CreateToken("ada@zatrano.test")
	if !errors.Is(err, auth.ErrResetThrottled) {
		t.Fatalf("expected throttle, got %v", err)
	}
}

func TestEmailVerificationHelpers(t *testing.T) {
	user := &auth.GenericUser{Attributes: map[string]any{"email": "ada@zatrano.test"}}
	if auth.HasVerifiedEmail(user) {
		t.Fatal("expected unverified")
	}
	auth.MarkEmailVerified(user.Attributes)
	if !auth.HasVerifiedEmail(user) {
		t.Fatal("expected verified")
	}
	if auth.EmailHash("ada@zatrano.test") == "" {
		t.Fatal("expected hash")
	}
}

func TestRegisterSendsEmailVerification(t *testing.T) {
	provider := newMemoryUserProvider()
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))

	var gotURL string
	var gotEmail string
	manager.SetVerificationURLGenerator(func(user auth.Authenticatable) (string, error) {
		return "https://example.test/verify/" + fmt.Sprint(user.AuthID()), nil
	})
	manager.SetEmailVerificationSender(func(user auth.Authenticatable, verifyURL string) error {
		gotEmail = auth.EmailForVerification(user)
		gotURL = verifyURL
		return nil
	})

	raw := httptest.NewRequest(stdhttp.MethodPost, "/register", nil)
	req := http.NewRequest(raw)
	req.SetSession(&memSession{data: map[string]any{}})

	_, err := manager.Register(req, map[string]any{
		"name": "Ada", "email": "ada@zatrano.test", "password": "secret1",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if gotEmail != "ada@zatrano.test" || gotURL != "https://example.test/verify/1" {
		t.Fatalf("verification not sent: email=%q url=%q", gotEmail, gotURL)
	}
}

func TestPasswordChangedSender(t *testing.T) {
	provider := newMemoryUserProvider()
	manager := auth.NewManager("web")
	manager.Extend("web", auth.NewGuard("web", provider))
	raw := httptest.NewRequest(stdhttp.MethodPost, "/register", nil)
	req := http.NewRequest(raw)
	req.SetSession(&memSession{data: map[string]any{}})
	user, err := manager.Register(req, map[string]any{
		"name": "Ada", "email": "ada@zatrano.test", "password": "secret1",
	})
	if err != nil {
		t.Fatal(err)
	}

	called := false
	manager.SetPasswordChangedSender(func(u auth.Authenticatable) error {
		called = true
		if fmt.Sprint(u.AuthID()) != fmt.Sprint(user.AuthID()) {
			t.Fatalf("unexpected user")
		}
		return nil
	})
	if err := manager.ChangePassword(req, "secret1", "secret2"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected password-changed notification")
	}
}

func TestPasswordResetTokenConstantTime(t *testing.T) {
	provider := &stubPasswordProvider{users: map[string]*auth.GenericUser{
		"ada@zatrano.test": {Attributes: map[string]any{"id": 1, "email": "ada@zatrano.test", "password": "old"}},
	}}
	tokens := auth.NewMemoryTokenRepository(time.Hour)
	broker := auth.NewPasswordBroker(tokens, provider, time.Hour)

	token, err := broker.CreateToken("ada@zatrano.test")
	if err != nil {
		t.Fatal(err)
	}
	if !broker.TokenValid("ada@zatrano.test", token) {
		t.Fatal("expected valid token")
	}
	if broker.TokenValid("ada@zatrano.test", "deadbeef") {
		t.Fatal("short invalid token must not match")
	}
	if broker.TokenValid("ada@zatrano.test", token+"x") {
		t.Fatal("mutated token must not match")
	}
	wrong := make([]byte, len(token))
	copy(wrong, token)
	wrong[0] ^= 0x01
	if broker.TokenValid("ada@zatrano.test", string(wrong)) {
		t.Fatal("same-length wrong token must not match")
	}
}
