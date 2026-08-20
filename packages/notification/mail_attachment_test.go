package notification_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/notification"
)

func TestAttachAndReplyTo(t *testing.T) {
	msg := (&notification.MailMessage{
		From:    "app@zatrano.test",
		To:      []string{"user@zatrano.test"},
		Subject: "Report",
		Text:    "See attached",
		HTML:    "<p>See attached</p>",
	}).SetReplyTo("support@zatrano.test").
		Attach("report.txt", []byte("hello"), "text/plain")

	if len(msg.ReplyTo) != 1 || msg.ReplyTo[0] != "support@zatrano.test" {
		t.Fatalf("reply-to=%v", msg.ReplyTo)
	}
	if len(msg.Attachments) != 1 || msg.Attachments[0].Name != "report.txt" {
		t.Fatalf("attachments=%v", msg.Attachments)
	}

	raw := string(notification.BuildMIME(msg))
	if !strings.Contains(raw, "Reply-To: support@zatrano.test") {
		t.Fatalf("missing reply-to: %s", raw)
	}
	if !strings.Contains(raw, "multipart/mixed") || !strings.Contains(raw, "report.txt") {
		t.Fatalf("missing attachment mime: %s", raw)
	}
	if !strings.Contains(raw, "Content-Transfer-Encoding: base64") {
		t.Fatalf("missing base64: %s", raw)
	}
}
