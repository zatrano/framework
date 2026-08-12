package consent_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/consent"
)

func TestEncodeParse(t *testing.T) {
	raw := consent.Encode(consent.Preferences{Analytics: true, Marketing: false})
	p := consent.Parse(raw)
	if !p.Necessary || !p.Analytics || p.Marketing {
		t.Fatalf("unexpected %#v", p)
	}
	if !consent.Allowed(p, "analytics") {
		t.Fatal("analytics should be allowed")
	}
	if consent.Allowed(p, "marketing") {
		t.Fatal("marketing should be denied")
	}
}

func TestParseInvalid(t *testing.T) {
	p := consent.Parse("not-json")
	if !p.Necessary || p.Analytics {
		t.Fatalf("expected defaults, got %#v", p)
	}
}

func TestBannerHTML(t *testing.T) {
	html := consent.BannerHTML()
	if html == "" || !strings.Contains(html, "zatrano-consent") {
		t.Fatalf("unexpected banner %q", html)
	}
}
