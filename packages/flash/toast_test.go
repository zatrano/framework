package flash_test

import (
	stdhttp "net/http"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/flash"
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

func TestToastHelpers(t *testing.T) {
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	req := http.NewRequest(raw)
	sess := &memSession{data: map[string]any{}}
	req.SetSession(sess)

	flash.ToastSuccess(req, "Saved")
	items := flash.Toasts(req)
	if len(items) != 1 || items[0].Type != "success" || items[0].Message != "Saved" {
		t.Fatalf("unexpected %#v", items)
	}
	html := flash.RenderToasts(req)
	if !strings.Contains(html, "Saved") {
		t.Fatalf("html=%q", html)
	}
}
