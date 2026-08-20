package fuzz_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/middleware/csrf"
	"github.com/zatrano/framework/packages/session"
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
func (s *memSession) Put(key string, value any) { s.data[key] = value }
func (s *memSession) Flash(key string, value any) {
	s.data[key] = value
}
func (s *memSession) Pull(key string, fallback ...any) any {
	v := s.Get(key, fallback...)
	delete(s.data, key)
	return v
}
func (s *memSession) Forget(key string) { delete(s.data, key) }
func (s *memSession) Regenerate() error { return nil }
func (s *memSession) ID() string        { return "fuzz" }

// FuzzSessionStart fuzzes session ID acceptance / path containment.
func FuzzSessionStart(f *testing.F) {
	for _, s := range []string{
		"",
		"0123456789abcdef0123456789abcdef",
		"../etc/passwd",
		"..\\windows",
		"/abs",
		"not-hex",
		string([]byte{0}),
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, id string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("session Start panic id=%q: %v", id, r)
			}
		}()
		if len(id) > 512 {
			return
		}
		dir := t.TempDir()
		mgr := session.NewManager(dir, 60)
		bag, err := mgr.Start(id)
		if err != nil {
			return
		}
		_ = bag.ID()
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if filepath.Base(e.Name()) != e.Name() {
				t.Errorf("unexpected entry name %q", e.Name())
			}
		}
	})
}

// FuzzCSRFMiddleware fuzzes token headers/cookies on unsafe methods.
func FuzzCSRFMiddleware(f *testing.F) {
	for _, s := range []string{"", "abc", "deadbeef", "<script>"} {
		f.Add(s, "https://app.example", "same-origin")
	}
	f.Fuzz(func(t *testing.T, token, origin, secFetch string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("csrf panic: %v", r)
			}
		}()
		if len(token) > 4096 || len(origin) > 1024 || len(secFetch) > 64 {
			return
		}
		handler := csrf.Middleware(func(req *http.Request) *http.Response {
			return http.Text("ok")
		})
		raw := httptest.NewRequest(stdhttp.MethodPost, "https://app.example/form", nil)
		req := http.NewRequest(raw)
		if strings.HasPrefix(origin, "https://") {
			req.Set("_forwarded_proto", "https")
		}
		sess := &memSession{data: map[string]any{}}
		req.SetSession(sess)
		_ = csrf.Token(req)
		raw.Header.Set("X-CSRF-TOKEN", token)
		if origin != "" {
			raw.Header.Set("Origin", origin)
		}
		if secFetch != "" {
			raw.Header.Set("Sec-Fetch-Site", secFetch)
		}
		_ = handler(req)
	})
}
