package useragent_test

import (
	"testing"

	"github.com/zatrano/framework/packages/useragent"
)

func TestParseUserAgent(t *testing.T) {
	chrome := useragent.Parse("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if chrome.Browser != "Chrome" || chrome.Platform != "Windows" || chrome.IsMobile {
		t.Fatalf("%+v", chrome)
	}
	ios := useragent.Parse("Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Version/17.0 Mobile/15E148 Safari/604.1")
	if ios.Browser != "Safari" || !ios.IsMobile || ios.Platform != "iOS" {
		t.Fatalf("%+v", ios)
	}
	bot := useragent.Parse("Googlebot/2.1 (+http://www.google.com/bot.html)")
	if !bot.IsBot {
		t.Fatalf("%+v", bot)
	}
}
