package otp_test

import (
	"testing"
	"time"

	"github.com/zatrano/framework/packages/otp"
)

func TestGenerateVerify(t *testing.T) {
	m := otp.New(otp.NewMemoryStore()).WithTTL(time.Minute)
	code, err := m.Generate("admin@zatrano.test")
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 {
		t.Fatalf("code=%q", code)
	}
	if !m.Verify("admin@zatrano.test", code) {
		t.Fatal("expected verify success")
	}
	if m.Verify("admin@zatrano.test", code) {
		t.Fatal("code should be consumed")
	}
}

func TestVerifyFail(t *testing.T) {
	m := otp.New(nil)
	_, _ = m.Generate("phone:+15551212")
	if m.Verify("phone:+15551212", "000000") {
		t.Fatal("expected failure")
	}
}
