package console_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/console"
)

func TestFilterDashboardModulesRemovesDisabled(t *testing.T) {
	src := `
core
// @module users
users-block
// @module impersonate
impersonate-block
// @endmodule impersonate
// @endmodule users
// @module api
api-block
// @endmodule api
`
	mods := map[string]bool{
		"users": true, "impersonate": false, "api": false,
		"notifications": true, "roles": true, "rbac": true, "settings": true, "analytics": true,
	}
	out := console.FilterDashboardModulesForTest(src, mods)
	if strings.Contains(out, "impersonate-block") {
		t.Fatalf("expected impersonate removed: %s", out)
	}
	if strings.Contains(out, "api-block") {
		t.Fatalf("expected api removed: %s", out)
	}
	if !strings.Contains(out, "users-block") {
		t.Fatalf("expected users kept: %s", out)
	}
	if strings.Contains(out, "@module") {
		t.Fatalf("expected markers stripped: %s", out)
	}
}
