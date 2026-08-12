package features_test

import (
	"testing"

	"github.com/zatrano/framework/packages/features"
)

func TestFeatureFlags(t *testing.T) {
	m := features.New()
	m.Activate("a")
	m.Deactivate("b")
	m.Rollout("c", 100)
	m.Define("d", func(ctx features.Context) bool {
		return ctx["role"] == "admin"
	})

	if !m.Active("a") || m.Active("b") {
		t.Fatal("boolean flags failed")
	}
	if !m.Active("c", features.Context{"key": "user-1"}) {
		t.Fatal("expected 100% rollout")
	}
	if !m.Active("d", features.Context{"role": "admin"}) {
		t.Fatal("expected gate pass")
	}
	if m.Active("d", features.Context{"role": "user"}) {
		t.Fatal("expected gate fail")
	}
}
