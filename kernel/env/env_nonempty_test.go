package env_test

import (
	"os"
	"testing"

	"github.com/zatrano/framework/v2/kernel/env"
)

func TestGetNonEmptyFallsBackOnBlank(t *testing.T) {
	t.Setenv("ZATRANO_TEST_USER", "")
	if got := env.GetNonEmpty("ZATRANO_TEST_USER", "postgres"); got != "postgres" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("ZATRANO_TEST_USER", "  ")
	if got := env.GetNonEmpty("ZATRANO_TEST_USER", "postgres"); got != "postgres" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("ZATRANO_TEST_USER", "alice")
	if got := env.GetNonEmpty("ZATRANO_TEST_USER", "postgres"); got != "alice" {
		t.Fatalf("got %q", got)
	}
	_ = os.Unsetenv("ZATRANO_TEST_USER")
	if got := env.GetNonEmpty("ZATRANO_TEST_USER", "postgres"); got != "postgres" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeAppEnv(t *testing.T) {
	if got := env.NormalizeAppEnv(" PRODUCTION "); got != "production" {
		t.Fatalf("got %q", got)
	}
	if got := env.NormalizeAppEnv("Staging"); got != "staging" {
		t.Fatalf("got %q", got)
	}
}
