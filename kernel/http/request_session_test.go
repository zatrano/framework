package http_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
)

type attrSession struct {
	id   string
	data map[string]any
}

func (s *attrSession) Get(key string, fallback ...any) any {
	if v, ok := s.data[key]; ok {
		return v
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return nil
}
func (s *attrSession) Put(key string, value any)            { s.data[key] = value }
func (s *attrSession) Flash(key string, value any)          { s.Put(key, value) }
func (s *attrSession) Pull(key string, fallback ...any) any { return s.Get(key, fallback...) }
func (s *attrSession) Forget(key string)                    { delete(s.data, key) }
func (s *attrSession) Regenerate() error                    { return nil }
func (s *attrSession) ID() string                           { return s.id }

func TestSessionLivesOnRequestAttribute(t *testing.T) {
	a := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	b := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	store := &attrSession{id: "s1", data: map[string]any{"k": "v"}}
	a.SetSession(store)
	if a.Session() == nil || a.Session().ID() != "s1" {
		t.Fatal("expected session on request A")
	}
	if a.Get("session") == nil {
		t.Fatal("session must be stored as the session attribute")
	}
	if b.Session() != nil {
		t.Fatal("session must not leak across requests")
	}
	a.SetSession(nil)
	if a.Session() != nil {
		t.Fatal("clearing session must drop the store")
	}
}

func TestGetIsNilSafe(t *testing.T) {
	var req *http.Request
	if req.Get("x") != nil {
		t.Fatal("nil request Get")
	}
	empty := &http.Request{}
	if empty.Get("x") != nil {
		t.Fatal("empty attrs Get")
	}
	empty.Set("x", 1)
	if empty.Get("x") != 1 {
		t.Fatal("Set must initialize attrs")
	}
}
