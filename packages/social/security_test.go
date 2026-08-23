package social

import (
	"errors"
	"strings"
	"testing"
)

func TestUserFromGoogleInfoRequiresVerifiedEmail(t *testing.T) {
	t.Run("verified", func(t *testing.T) {
		u, err := userFromGoogleInfo(map[string]any{
			"email":          "user@example.com",
			"email_verified": true,
			"sub":            "google-123",
			"name":           "User",
		}, "tok")
		if err != nil {
			t.Fatal(err)
		}
		if !u.EmailVerified || u.ID != "google-123" || u.Email != "user@example.com" {
			t.Fatalf("%+v", u)
		}
	})

	t.Run("unverified", func(t *testing.T) {
		_, err := userFromGoogleInfo(map[string]any{
			"email":          "user@example.com",
			"email_verified": false,
			"sub":            "google-123",
		}, "tok")
		if err == nil || !strings.Contains(err.Error(), "not verified") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("missing claim", func(t *testing.T) {
		_, err := userFromGoogleInfo(map[string]any{
			"email": "user@example.com",
			"sub":   "google-123",
		}, "tok")
		if err == nil || !strings.Contains(err.Error(), "email_verified claim missing") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("missing email", func(t *testing.T) {
		_, err := userFromGoogleInfo(map[string]any{
			"email_verified": true,
			"sub":            "google-123",
		}, "tok")
		if err == nil || !strings.Contains(err.Error(), "email missing") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("missing sub", func(t *testing.T) {
		_, err := userFromGoogleInfo(map[string]any{
			"email":          "user@example.com",
			"email_verified": true,
		}, "tok")
		if err == nil || !strings.Contains(err.Error(), "sub missing") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("string true claim", func(t *testing.T) {
		u, err := userFromGoogleInfo(map[string]any{
			"email":          "user@example.com",
			"email_verified": "true",
			"sub":            "google-xyz",
		}, "tok")
		if err != nil || !u.EmailVerified {
			t.Fatalf("u=%+v err=%v", u, err)
		}
	})
}

func TestStubProviderDevDemoUser(t *testing.T) {
	SetAllowStubProviders(true)
	t.Cleanup(func() { SetAllowStubProviders(true) })

	p := Google(Config{
		ClientID:     "google-client-id",
		ClientSecret: "google-client-secret",
		RedirectURL:  "http://localhost/callback",
	})
	if _, ok := p.(*StubProvider); !ok {
		t.Fatalf("expected StubProvider, got %T", p)
	}
	u, err := p.UserFromCode("demo")
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "demo@google.test" || !u.EmailVerified {
		t.Fatalf("%+v", u)
	}
}

func TestStubProviderRejectedWhenDisallowed(t *testing.T) {
	SetAllowStubProviders(false)
	t.Cleanup(func() { SetAllowStubProviders(true) })

	p := Google(Config{
		ClientID:     "google-client-id",
		ClientSecret: "google-client-secret",
		RedirectURL:  "http://localhost/callback",
	})
	_, err := p.UserFromCode("demo")
	if !errors.Is(err, ErrStubNotAllowedInProduction) {
		t.Fatalf("err=%v", err)
	}
	if !IsPlaceholder("google-client-id", "google-client-secret") {
		t.Fatal("expected placeholder detection")
	}
}
