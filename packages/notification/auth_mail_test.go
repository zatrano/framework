package notification_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/localization"
	"github.com/zatrano/framework/packages/notification"
)

func TestPasswordResetNotificationLocalized(t *testing.T) {
	tr := localization.New("", "en", "en")
	_ = tr.Load("en")
	_ = tr.Load("tr")
	notification.SetTranslator(tr)
	t.Cleanup(func() {
		notification.SetTranslator(nil)
		notification.SetMailDefaults("", "")
	})

	t.Run("en", func(t *testing.T) {
		notification.SetMailDefaults("en", "DemoApp")
		n := notification.PasswordResetNotification{
			Token:         "secret-token",
			ResetURL:      "https://example.com/reset",
			ExpireMinutes: 45,
		}
		msg := n.ToMail(notification.Recipient{Email: "a@example.com"})
		if msg == nil || msg.Subject == "" || msg.Text == "" || msg.HTML == "" {
			t.Fatalf("empty mail: %+v", msg)
		}
		if !strings.Contains(msg.Subject, "DemoApp") {
			t.Fatalf("subject missing app: %q", msg.Subject)
		}
		if !strings.Contains(msg.Text, "https://example.com/reset?") {
			t.Fatalf("text missing link: %q", msg.Text)
		}
		if strings.Contains(msg.Text, "secret-token") && !strings.Contains(msg.Text, "token=secret-token") {
			t.Fatalf("raw token should not appear outside URL query: %q", msg.Text)
		}
		if !strings.Contains(msg.HTML, "token=secret-token") {
			t.Fatalf("html missing token in href: %q", msg.HTML)
		}
		if strings.Contains(msg.HTML, ">secret-token<") {
			t.Fatalf("html should not show raw token as body text")
		}
		if !strings.Contains(msg.Text, "45") {
			t.Fatalf("text missing minutes: %q", msg.Text)
		}
	})

	t.Run("tr", func(t *testing.T) {
		notification.SetMailDefaults("tr", "DemoApp")
		n := notification.PasswordResetNotification{
			Token:    "tok",
			ResetURL: "https://example.com/reset",
		}
		msg := n.ToMail(notification.Recipient{Email: "a@example.com"})
		if !strings.Contains(strings.ToLower(msg.Subject), "parola") {
			t.Fatalf("expected Turkish subject, got %q", msg.Subject)
		}
		if !strings.Contains(msg.Text, "https://example.com/reset?") {
			t.Fatalf("text missing link: %q", msg.Text)
		}
		if msg.Subject == "auth.mail_reset_subject" {
			t.Fatal("subject unresolved")
		}
	})
}

func TestVerifyAndPasswordChangedLocalized(t *testing.T) {
	tr := localization.New("", "tr", "en")
	_ = tr.Load("tr")
	_ = tr.Load("en")
	notification.SetTranslator(tr)
	notification.SetMailDefaults("tr", "ZApp")
	t.Cleanup(func() {
		notification.SetTranslator(nil)
		notification.SetMailDefaults("", "")
	})

	verify := notification.VerifyEmailNotification{VerifyURL: "https://example.com/verify?sig=1"}
	vmail := verify.ToMail(notification.Recipient{Email: "a@example.com"})
	if vmail.Subject == "" || !strings.Contains(vmail.Text, "https://example.com/verify") {
		t.Fatalf("%+v", vmail)
	}
	if !strings.Contains(strings.ToLower(vmail.Subject), "doğrulama") && !strings.Contains(strings.ToLower(vmail.Subject), "e-posta") {
		t.Fatalf("expected Turkish verify subject: %q", vmail.Subject)
	}

	changed := notification.PasswordChangedNotification{}
	cmail := changed.ToMail(notification.Recipient{Email: "a@example.com"})
	if cmail.Subject == "" || cmail.Text == "" {
		t.Fatalf("%+v", cmail)
	}
	if !strings.Contains(cmail.Subject, "ZApp") {
		t.Fatalf("subject missing app: %q", cmail.Subject)
	}
}
