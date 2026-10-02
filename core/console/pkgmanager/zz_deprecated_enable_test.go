package pkgmanager

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel"
)

func TestRejectEnableDeprecatedDatabase(t *testing.T) {
	_ = kernel.NewApplication(t.TempDir())
	err := rejectEnableTarget("database")
	if err == nil || !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("expected deprecated rejection, got %v", err)
	}
	err = rejectEnableTarget("orm")
	if err == nil || !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("expected orm deprecated rejection, got %v", err)
	}
	err = rejectEnableTarget("view")
	if err == nil || !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("expected view deprecated rejection, got %v", err)
	}
	err = rejectEnableTarget("factory")
	if err == nil || !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("expected factory deprecated rejection, got %v", err)
	}
	err = rejectEnableTarget("httpclient")
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected httpclient unknown rejection (gone from catalog), got %v", err)
	}
}
