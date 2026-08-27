package console_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDashboardStubsPresent(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Join(filepath.Dir(file), "stubs")
	required := []string{
		"layouts/dashboard.html",
		"partials/dashboard/nav.html",
		"dashboard/home.html",
		"public/css/dashboard.css",
		"public/js/dashboard-shell.js",
		"public/js/dashboard-analytics.js",
		"lang/en/dashboard.json",
		"lang/tr/dashboard.json",
		"go/routes_dashboard.go.stub",
		"go/dashboard_controller.go.stub",
		"go/dashboard_users_controller.go.stub",
		"go/dashboard_api_controller.go.stub",
		"go/migration_dashboard.go.stub",
		"go/impersonate_middleware.go.stub",
	}
	for _, rel := range required {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing stub %s: %v", rel, err)
		}
	}
}
