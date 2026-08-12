package notification_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/notification"
)

func TestHTTPSmsSenderJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("auth=%q", r.Header.Get("Authorization"))
		}
		raw, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(raw), `"to":"+1"`) {
			t.Errorf("body=%s", raw)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	sender := &notification.HTTPSmsSender{
		Endpoint: srv.URL,
		Token:    "secret",
	}
	if err := sender.Send(&notification.SmsMessage{To: "+1", Body: "hi", From: "Z"}); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPSmsSenderForm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("To") != "+90" || r.Form.Get("Body") != "ping" {
			t.Errorf("form=%v", r.Form)
		}
		w.WriteHeader(202)
	}))
	defer srv.Close()

	sender := &notification.HTTPSmsSender{Endpoint: srv.URL, Form: true}
	if err := sender.Send(&notification.SmsMessage{To: "+90", Body: "ping"}); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPPushSender(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer push-secret" {
			t.Errorf("auth=%q", r.Header.Get("Authorization"))
		}
		raw, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(raw), `"token":"device-1"`) || !strings.Contains(string(raw), `"title":"Hi"`) {
			t.Errorf("body=%s", raw)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	sender := &notification.HTTPPushSender{Endpoint: srv.URL, Token: "push-secret"}
	if err := sender.Send("device-1", map[string]any{"title": "Hi"}); err != nil {
		t.Fatal(err)
	}
}
