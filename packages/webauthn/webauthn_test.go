package webauthn_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/webauthn"
)

func TestMissingConfigErrors(t *testing.T) {
	m := webauthn.New("", "", "ZATRANO")
	_, err := m.BeginRegistration("1", "admin@zatrano.test", "Admin")
	if err == nil || !strings.Contains(err.Error(), "WEBAUTHN_RP_ID") {
		t.Fatalf("expected config error, got %v", err)
	}
	_, err = m.BeginLogin("1")
	if err == nil || !strings.Contains(err.Error(), "WEBAUTHN_RP_ID") {
		t.Fatalf("expected config error, got %v", err)
	}
}

func TestBeginRegistrationOptions(t *testing.T) {
	m := webauthn.New("localhost", "http://localhost:8080", "ZATRANO")
	opts, err := m.BeginRegistration("1", "admin@zatrano.test", "Admin")
	if err != nil {
		t.Fatal(err)
	}
	if opts.ChallengeID == "" || opts.Options == nil {
		t.Fatalf("unexpected %#v", opts)
	}
	if opts.Options.Response.RelyingParty.ID != "localhost" {
		t.Fatalf("rp id=%q", opts.Options.Response.RelyingParty.ID)
	}
	_, err = m.FinishRegistration(opts.ChallengeID, []byte(`{"not":"valid"}`))
	if err == nil {
		t.Fatal("expected invalid attestation to fail")
	}
}

func TestBeginLoginRequiresCredential(t *testing.T) {
	m := webauthn.New("localhost", "http://localhost:8080", "ZATRANO")
	_, err := m.BeginLogin("missing")
	if err == nil || !strings.Contains(err.Error(), "no credentials") {
		t.Fatalf("expected no credentials error, got %v", err)
	}
}
