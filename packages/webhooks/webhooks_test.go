package webhooks_test

import (
	"testing"

	"github.com/zatrano/framework/packages/webhooks"
)

func TestWebhookSignVerify(t *testing.T) {
	body := []byte(`{"ok":true}`)
	sig := webhooks.Sign("secret", body)
	if !webhooks.Verify("secret", sig, body) {
		t.Fatal("expected verify pass")
	}
	if webhooks.Verify("other", sig, body) {
		t.Fatal("expected verify fail")
	}
}
