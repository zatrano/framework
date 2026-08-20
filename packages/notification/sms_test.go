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

func TestSmsManagerExtendUse(t *testing.T) {
	memA := &notification.MemorySmsSender{}
	memB := &notification.MemorySmsSender{}
	sms := notification.NewSmsManager("Z")
	sms.Extend("alpha", memA)
	sms.Extend("beta", memB)
	sms.Use("alpha")

	mgr := notification.NewManager()
	mgr.SetSms(sms)

	if err := mgr.SendNow(notification.Recipient{Phone: "+1"}, notification.Message{
		Channels: []string{"sms"},
		Body:     "via-default",
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok := memA.Last(); !ok {
		t.Fatal("expected alpha driver")
	}
	if _, ok := memB.Last(); ok {
		t.Fatal("beta should be unused")
	}

	if err := mgr.SendNow(notification.Recipient{Phone: "+2"}, notification.Message{
		Channels: []string{"sms.beta"},
		Body:     "via-beta",
	}); err != nil {
		t.Fatal(err)
	}
	if last, ok := memB.Last(); !ok || last.Body != "via-beta" {
		t.Fatalf("expected beta delivery, got %#v", last)
	}
}

func TestSmsMetaDriverOverride(t *testing.T) {
	memA := &notification.MemorySmsSender{}
	memB := &notification.MemorySmsSender{}
	sms := notification.NewSmsManager("Z")
	sms.Extend("alpha", memA)
	sms.Extend("beta", memB)
	sms.Use("alpha")
	mgr := notification.NewManager()
	mgr.SetSms(sms)

	n := driverSMS{body: "forced", driver: "beta"}
	if err := mgr.SendNow(notification.Recipient{Phone: "+3"}, n); err != nil {
		t.Fatal(err)
	}
	if _, ok := memB.Last(); !ok {
		t.Fatal("expected meta driver beta")
	}
}

type driverSMS struct {
	notification.Base
	body   string
	driver string
}

func (d driverSMS) Via() []string { return []string{"sms"} }

func (d driverSMS) ToSms(notification.Notifiable) *notification.SmsMessage {
	return &notification.SmsMessage{
		Body: d.body,
		Meta: map[string]any{"driver": d.driver},
	}
}
