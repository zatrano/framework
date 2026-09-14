package http_test

import (
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
)

func TestParseUserAgent(t *testing.T) {
	chrome := http.ParseUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if chrome.Browser != "Chrome" || chrome.Platform != "Windows" || chrome.IsMobile {
		t.Fatalf("%+v", chrome)
	}
	ios := http.ParseUserAgent("Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Version/17.0 Mobile/15E148 Safari/604.1")
	if ios.Browser != "Safari" || !ios.IsMobile || ios.Platform != "iOS" {
		t.Fatalf("%+v", ios)
	}
	bot := http.ParseUserAgent("Googlebot/2.1 (+http://www.google.com/bot.html)")
	if !bot.IsBot {
		t.Fatalf("%+v", bot)
	}
	edge := http.ParseUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
	if edge.Browser != "Edge" {
		t.Fatalf("%+v", edge)
	}
	ff := http.ParseUserAgent("Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0")
	if ff.Browser != "Firefox" || ff.Platform != "Linux" {
		t.Fatalf("%+v", ff)
	}
	ie := http.ParseUserAgent("Mozilla/5.0 (Windows NT 6.1; Trident/7.0; rv:11.0) like Gecko")
	if ie.Browser != "IE" {
		t.Fatalf("%+v", ie)
	}
	ipad := http.ParseUserAgent("Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Version/17.0 Safari/604.1")
	if ipad.Device != "Tablet" || ipad.Platform != "iOS" {
		t.Fatalf("%+v", ipad)
	}
	mac := http.ParseUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/17.0 Safari/605.1.15")
	if mac.Platform != "macOS" || mac.Browser != "Safari" {
		t.Fatalf("%+v", mac)
	}
	tablet := http.ParseUserAgent("Mozilla/5.0 (Linux; Android 13; Tablet) AppleWebKit/537.36 Chrome/120.0.0.0 Mobile Safari/537.36")
	if !tablet.IsMobile {
		t.Fatalf("%+v", tablet)
	}
}
