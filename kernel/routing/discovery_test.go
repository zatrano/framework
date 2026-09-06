package routing

import (
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
)

func dummyOK(req *http.Request) *http.Response {
	return http.JSON(map[string]any{"ok": true})
}

func TestApplyWebRunsRegistrarsFromSeparateCalls(t *testing.T) {
	discoveryMu.Lock()
	prevWeb, prevAPI := webFns, apiFns
	webFns, apiFns = nil, nil
	discoveryMu.Unlock()
	t.Cleanup(func() {
		discoveryMu.Lock()
		webFns, apiFns = prevWeb, prevAPI
		discoveryMu.Unlock()
	})

	// Two files / two init() sites:
	RegisterWeb(func(r *Router) {
		r.Get("/from-a", dummyOK).As("from.a")
	})
	RegisterWeb(func(r *Router) {
		r.Get("/from-b", dummyOK).As("from.b")
	})

	r := New()
	ApplyWeb(r)

	seen := map[string]bool{}
	for _, route := range r.Routes() {
		seen[route.Path] = true
	}
	if !seen["/from-a"] || !seen["/from-b"] {
		t.Fatalf("ApplyWeb missing paths, got %#v", seen)
	}
}
