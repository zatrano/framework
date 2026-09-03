package totp_test

import (
	"testing"
	"time"

	"github.com/zatrano/framework/packages/auth/totp"
)

func TestTOTPRoundTrip(t *testing.T) {
	secret, err := totp.GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	code, err := totp.Code(secret, time.Now())
	if err != nil || len(code) != 6 {
		t.Fatalf("code=%q err=%v", code, err)
	}
	if !totp.Verify(secret, code) {
		t.Fatal("verify failed")
	}
	url := totp.OTPAuthURL("ZATRANO", "admin@zatrano.test", secret)
	if url == "" || url[:10] != "otpauth://" {
		t.Fatal(url)
	}
}
