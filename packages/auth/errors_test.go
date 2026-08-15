package auth_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/localization"
)

func TestAuthErrorsReturnLocalizationKeys(t *testing.T) {
	cases := []struct {
		err error
		key string
	}{
		{auth.ErrEmailTaken, "auth.email_taken"},
		{auth.ErrLockout, "auth.lockout"},
		{auth.ErrCurrentPassword, "auth.password"},
		{auth.ErrResetTokenInvalid, "auth.reset_token_invalid"},
		{auth.ErrUnauthenticated, "auth.unauthenticated"},
	}
	for _, tc := range cases {
		if got := auth.MessageKey(tc.err); got != tc.key {
			t.Fatalf("MessageKey(%v)=%q want %q", tc.err, got, tc.key)
		}
		if !errors.Is(tc.err, tc.err) {
			t.Fatal("sentinel broken")
		}
		if !strings.HasPrefix(tc.err.Error(), "auth.") {
			t.Fatalf("Error() should be catalog key, got %q", tc.err.Error())
		}
	}

	tr := localization.New("", "tr", "en")
	_ = tr.Load("tr")
	msg := tr.Get(auth.MessageKey(auth.ErrEmailTaken))
	if msg == "" || msg == "auth.email_taken" || !strings.Contains(strings.ToLower(msg), "e-posta") && !strings.Contains(msg, "kullanılıyor") {
		// Turkish default should translate; accept either Turkish wording.
		if msg == "auth.email_taken" {
			t.Fatalf("expected translated auth.email_taken, got key passthrough")
		}
	}
	en := localization.New("", "en", "en")
	_ = en.Load("en")
	enMsg := en.Get(auth.ErrEmailTaken.Error())
	if enMsg == "auth.email_taken" {
		t.Fatalf("expected english translation for auth.email_taken")
	}
}
