package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/zatrano/framework/core"
)

// TestCase wraps an application for HTTP feature tests.
type TestCase struct {
	App     *core.Application
	headers map[string]string
	cookies []*stdhttp.Cookie
}

// New creates a test case from an application.
func New(app *core.Application) (*TestCase, error) {
	if !appIsBooted(app) {
		if err := app.Bootstrap(); err != nil {
			return nil, err
		}
	}
	return &TestCase{
		App:     app,
		headers: map[string]string{},
		cookies: make([]*stdhttp.Cookie, 0),
	}, nil
}

func appIsBooted(app *core.Application) bool {
	// Bootstrap is safe to call only when needed; detect via logger presence.
	return app.Logger() != nil && app.Router() != nil
}

// WithHeader sets a request header for subsequent calls.
func (t *TestCase) WithHeader(key, value string) *TestCase {
	t.headers[key] = value
	return t
}

// WithToken sets a bearer token.
func (t *TestCase) WithToken(token string) *TestCase {
	return t.WithHeader("Authorization", "Bearer "+token)
}

// WithCookie adds a cookie for subsequent requests.
func (t *TestCase) WithCookie(name, value string) *TestCase {
	t.cookies = append(t.cookies, &stdhttp.Cookie{Name: name, Value: value})
	return t
}

// AcceptJSON sets Accept to application/json.
func (t *TestCase) AcceptJSON() *TestCase {
	return t.WithHeader("Accept", "application/json")
}

// Get performs a GET request.
func (t *TestCase) Get(uri string) *TestResponse {
	return t.call(stdhttp.MethodGet, uri, nil, "")
}

// Post performs a POST request with form or JSON body.
func (t *TestCase) Post(uri string, body any) *TestResponse {
	return t.call(stdhttp.MethodPost, uri, body, "")
}

// PostJSON performs a JSON POST request.
func (t *TestCase) PostJSON(uri string, body any) *TestResponse {
	return t.call(stdhttp.MethodPost, uri, body, "application/json")
}

// PutJSON performs a JSON PUT request.
func (t *TestCase) PutJSON(uri string, body any) *TestResponse {
	return t.call(stdhttp.MethodPut, uri, body, "application/json")
}

// Delete performs a DELETE request.
func (t *TestCase) Delete(uri string) *TestResponse {
	return t.call(stdhttp.MethodDelete, uri, nil, "")
}

func (t *TestCase) call(method, uri string, body any, contentType string) *TestResponse {
	var reader io.Reader
	switch v := body.(type) {
	case nil:
		reader = nil
	case string:
		reader = strings.NewReader(v)
	case []byte:
		reader = bytes.NewReader(v)
	case map[string]string:
		if contentType == "application/json" {
			raw, _ := json.Marshal(v)
			reader = bytes.NewReader(raw)
		} else {
			form := url.Values{}
			for key, value := range v {
				form.Set(key, value)
			}
			reader = strings.NewReader(form.Encode())
			if contentType == "" {
				contentType = "application/x-www-form-urlencoded"
			}
		}
	default:
		raw, _ := json.Marshal(v)
		reader = bytes.NewReader(raw)
		if contentType == "" {
			contentType = "application/json"
		}
	}

	req := httptest.NewRequest(method, uri, reader)
	for key, value := range t.headers {
		req.Header.Set(key, value)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, cookie := range t.cookies {
		req.AddCookie(cookie)
	}

	recorder := httptest.NewRecorder()
	t.App.ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	t.cookies = append(t.cookies, resp.Cookies()...)

	return &TestResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       raw,
	}
}

// TestResponse asserts against an HTTP response.
type TestResponse struct {
	StatusCode int
	Headers    stdhttp.Header
	Body       []byte
}

// AssertStatus asserts the status code.
func (r *TestResponse) AssertStatus(status int) *TestResponse {
	if r.StatusCode != status {
		panic(newAssertionError("expected status %d, got %d; body=%s", status, r.StatusCode, string(r.Body)))
	}
	return r
}

// AssertOK asserts 200.
func (r *TestResponse) AssertOK() *TestResponse {
	return r.AssertStatus(200)
}

// AssertCreated asserts 201.
func (r *TestResponse) AssertCreated() *TestResponse {
	return r.AssertStatus(201)
}

// AssertUnauthorized asserts 401.
func (r *TestResponse) AssertUnauthorized() *TestResponse {
	return r.AssertStatus(401)
}

// AssertNotFound asserts 404.
func (r *TestResponse) AssertNotFound() *TestResponse {
	return r.AssertStatus(404)
}

// AssertJSONContains asserts a key/value exists in JSON object body.
func (r *TestResponse) AssertJSONContains(key string, expected any) *TestResponse {
	var payload map[string]any
	if err := json.Unmarshal(r.Body, &payload); err != nil {
		panic(newAssertionError("invalid json: %v; body=%s", err, string(r.Body)))
	}
	actual, ok := payload[key]
	if !ok {
		panic(newAssertionError("missing json key %q; body=%s", key, string(r.Body)))
	}
	if stringify(actual) != stringify(expected) {
		panic(newAssertionError("json key %q expected %v, got %v", key, expected, actual))
	}
	return r
}

// AssertSee asserts the body contains needle.
func (r *TestResponse) AssertSee(needle string) *TestResponse {
	if !strings.Contains(string(r.Body), needle) {
		panic(newAssertionError("expected body to contain %q; body=%s", needle, string(r.Body)))
	}
	return r
}

// AssertDontSee asserts the body does not contain needle.
func (r *TestResponse) AssertDontSee(needle string) *TestResponse {
	if strings.Contains(string(r.Body), needle) {
		panic(newAssertionError("expected body not to contain %q; body=%s", needle, string(r.Body)))
	}
	return r
}

// AssertHeader asserts a response header equals value.
func (r *TestResponse) AssertHeader(key, value string) *TestResponse {
	actual := r.Headers.Get(key)
	if actual != value {
		panic(newAssertionError("header %q expected %q, got %q", key, value, actual))
	}
	return r
}

// AssertRedirect asserts a redirect Location.
func (r *TestResponse) AssertRedirect(to string) *TestResponse {
	if r.StatusCode < 300 || r.StatusCode >= 400 {
		panic(newAssertionError("expected redirect status, got %d", r.StatusCode))
	}
	loc := r.Headers.Get("Location")
	if loc != to {
		panic(newAssertionError("expected redirect to %q, got %q", to, loc))
	}
	return r
}

// AssertJSONPath asserts a dotted path in a JSON object.
func (r *TestResponse) AssertJSONPath(path string, expected any) *TestResponse {
	var payload any
	if err := json.Unmarshal(r.Body, &payload); err != nil {
		panic(newAssertionError("invalid json: %v; body=%s", err, string(r.Body)))
	}
	actual, ok := digJSON(payload, strings.Split(path, ".")...)
	if !ok {
		panic(newAssertionError("missing json path %q; body=%s", path, string(r.Body)))
	}
	if stringify(actual) != stringify(expected) {
		panic(newAssertionError("json path %q expected %v, got %v", path, expected, actual))
	}
	return r
}

// JSON decodes the body.
func (r *TestResponse) JSON(dest any) error {
	return json.Unmarshal(r.Body, dest)
}

// String returns the body as string.
func (r *TestResponse) String() string {
	return string(r.Body)
}

func digJSON(value any, parts ...string) (any, bool) {
	current := value
	for _, part := range parts {
		if part == "" {
			continue
		}
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := obj[part]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

type assertionError string

func (e assertionError) Error() string { return string(e) }

func newAssertionError(format string, args ...any) assertionError {
	return assertionError(fmt.Sprintf(format, args...))
}

func stringify(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(raw)
}
