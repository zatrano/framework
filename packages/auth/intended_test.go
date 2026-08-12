package auth_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/http"
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
func (s *memSession) ID() string        { return "test" }

func TestIntendedURLHelpers(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodGet, "/dashboard?tab=1", nil)
	req := http.NewRequest(raw)
	sess := &memSession{data: map[string]any{}}
	req.SetSession(sess)

	auth.CaptureIntendedFromRequest(req)
	if got := auth.IntendedURL(req, "/"); got != "/dashboard?tab=1" {
		t.Fatalf("intended=%q", got)
	}
	if got := auth.PullIntendedURL(req, "/"); got != "/dashboard?tab=1" {
		t.Fatalf("pull=%q", got)
	}
	if got := auth.PullIntendedURL(req, "/home"); got != "/home" {
		t.Fatalf("fallback=%q", got)
	}
}
