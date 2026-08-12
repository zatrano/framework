package validation_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/validation"
)

type memSession struct {
	data map[string]any
}

func (s *memSession) Get(key string, fallback ...any) any {
	if v, ok := s.data[key]; ok {
		return v
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return nil
}
func (s *memSession) Put(key string, value any)   { s.data[key] = value }
func (s *memSession) Flash(key string, value any) { s.data[key] = value }
func (s *memSession) Pull(key string, fallback ...any) any {
	v := s.Get(key, fallback...)
	delete(s.data, key)
	return v
}
func (s *memSession) Forget(key string) { delete(s.data, key) }
func (s *memSession) Regenerate() error { return nil }
func (s *memSession) ID() string        { return "test" }

type sampleRequest struct {
	validation.Base
}

func (sampleRequest) Rules() map[string]string {
	return map[string]string{
		"title": "required|min:3",
		"body":  "required",
	}
}

type namedBagRequest struct {
	validation.Base
}

func (namedBagRequest) Rules() map[string]string {
	return map[string]string{"email": "required|email"}
}

func (namedBagRequest) ErrorBag() string { return "login" }

func (namedBagRequest) Attributes() map[string]string {
	return map[string]string{"email": "email address"}
}

func (namedBagRequest) Messages() map[string]string {
	return map[string]string{"email.required": "The :attribute is required."}
}

func (namedBagRequest) PrepareForValidation(req *http.Request) {
	req.Merge(map[string]string{"email": strings.TrimSpace(req.Input("email"))})
}

func TestValidateRequest(t *testing.T) {
	form := url.Values{}
	form.Set("title", "Hi")
	form.Set("body", "World")
	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodPost, "/posts", strings.NewReader(form.Encode())))
	req.Raw().Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := validation.ValidateRequest(req, map[string]string{
		"title": "required|min:3",
		"body":  "required",
	})
	if err == nil {
		t.Fatal("expected validation failure for short title")
	}
	ve, ok := err.(validation.ValidationException)
	if !ok || !ve.Errors.Has("title") {
		t.Fatalf("expected ValidationException with title, got %T %v", err, err)
	}
	form.Set("title", "Hello")
	req = http.NewRequest(httptest.NewRequest(stdhttp.MethodPost, "/posts", strings.NewReader(form.Encode())))
	req.Raw().Header.Set("Content-Type", "application/x-www-form-urlencoded")
	data, err := validation.ValidateRequest(req, map[string]string{
		"title": "required|min:3",
		"body":  "required",
	})
	if err != nil || data["title"] != "Hello" || data["body"] != "World" {
		t.Fatalf("data=%v err=%v", data, err)
	}
}

func TestPrecognitionSuccess(t *testing.T) {
	form := url.Values{}
	form.Set("title", "Hello")
	form.Set("body", "World")
	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodPost, "/posts", strings.NewReader(form.Encode())))
	req.Raw().Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Raw().Header.Set("Precognition", "true")

	resp := validation.WithForm(sampleRequest{}, func(req *http.Request, data map[string]string) *http.Response {
		t.Fatal("handler should not run for precognition")
		return nil
	})(req)

	if resp.StatusCode() != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode())
	}
}

func TestFormRequestValidation(t *testing.T) {
	form := url.Values{}
	form.Set("title", "Hi")
	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodPost, "/posts", strings.NewReader(form.Encode())))
	req.Raw().Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Raw().Header.Set("Accept", "application/json")

	resp := validation.WithForm(sampleRequest{}, func(req *http.Request, data map[string]string) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})(req)

	if resp.StatusCode() != 422 {
		t.Fatalf("expected 422, got %d body=%s", resp.StatusCode(), string(resp.Content()))
	}
}

func TestResponseForHTMLRedirect(t *testing.T) {
	form := url.Values{}
	form.Set("title", "Hi")
	raw := httptest.NewRequest(stdhttp.MethodPost, "http://example.test/posts", strings.NewReader(form.Encode()))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	raw.Header.Set("Accept", "text/html")
	raw.Header.Set("Referer", "http://example.test/demo/profile")
	req := http.NewRequest(raw)
	sess := &memSession{data: map[string]any{}}
	req.SetSession(sess)

	resp := validation.WithForm(sampleRequest{}, func(req *http.Request, data map[string]string) *http.Response {
		t.Fatal("handler should not run")
		return nil
	})(req)

	if !resp.IsRedirect() || resp.RedirectURL() != "/demo/profile" {
		t.Fatalf("expected redirect to referer, status=%d url=%q", resp.StatusCode(), resp.RedirectURL())
	}
	bag := validation.ErrorsFromSession(req)
	if !bag.Has("title") && !bag.Has("body") {
		t.Fatalf("expected flashed default errors, got %#v sess=%v", bag.All(), sess.data)
	}
}

func TestNamedErrorBagFlashAndForm(t *testing.T) {
	form := url.Values{}
	form.Set("email", "  ")
	raw := httptest.NewRequest(stdhttp.MethodPost, "http://example.test/login", strings.NewReader(form.Encode()))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	raw.Header.Set("Accept", "text/html")
	raw.Header.Set("Referer", "http://example.test/demo/validation/login")
	req := http.NewRequest(raw)
	sess := &memSession{data: map[string]any{}}
	req.SetSession(sess)

	resp := validation.WithForm(namedBagRequest{}, func(req *http.Request, data map[string]string) *http.Response {
		t.Fatal("handler should not run")
		return nil
	})(req)
	if !resp.IsRedirect() {
		t.Fatalf("expected html redirect, got %d", resp.StatusCode())
	}

	if validation.ErrorsFromSession(req).Any() {
		t.Fatal("default bag should be empty for named bag failures")
	}
	login := validation.ErrorsBagFromSession(req, "login")
	if !login.Has("email") {
		t.Fatalf("login bag missing email: %#v sess=%v", login.All(), sess.data)
	}
	if !strings.Contains(login.First("email"), "email address") {
		t.Fatalf("attribute placeholder missing: %q", login.First("email"))
	}

	bags := validation.ErrorBagsFromSession(req)
	if _, ok := bags["login"]; !ok {
		t.Fatalf("errorBags missing login: %#v", bags)
	}
}

func TestAttributesInMessages(t *testing.T) {
	v := validation.Make(map[string]string{"user_name": ""}, map[string]string{"user_name": "required"})
	v.SetAttributes(map[string]string{"user_name": "username"})
	v.SetMessages(map[string]string{"user_name.required": "Please provide the :attribute."})
	if !v.Fails() {
		t.Fatal("expected fail")
	}
	if got := v.Errors().First("user_name"); got != "Please provide the username." {
		t.Fatalf("got %q", got)
	}
}
