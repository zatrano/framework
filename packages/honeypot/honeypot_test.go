package honeypot_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/honeypot"
	"github.com/zatrano/framework/packages/http"
)

func TestHoneypotRejectsFilledField(t *testing.T) {
	mw := honeypot.Middleware()
	handler := mw(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})

	form := url.Values{"website": {"http://spam.test"}, "name": {"bot"}}
	r := httptest.NewRequest(stdhttp.MethodPost, "/contact", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := handler(http.NewRequest(r))
	if resp.StatusCode() != 422 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func TestHoneypotAllowsClean(t *testing.T) {
	mw := honeypot.Middleware(honeypot.Config{Field: "website", Timestamp: "_hp_ts", MinDelay: time.Millisecond})
	handler := mw(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	form := url.Values{
		"website": {""},
		"_hp_ts":  {time.Now().UTC().Add(-time.Second).Format(time.RFC3339)},
		"name":    {"Ada"},
	}
	r := httptest.NewRequest(stdhttp.MethodPost, "/contact", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := handler(http.NewRequest(r))
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d body=%s", resp.StatusCode(), string(resp.Content()))
	}
	if !strings.Contains(honeypot.Fields(), "website") {
		t.Fatal("fields html")
	}
}
