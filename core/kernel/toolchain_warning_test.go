package kernel

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/goversion"
)

func TestToolchainWarningOnlyForOldProductionBuilds(t *testing.T) {
	cases := []struct {
		production bool
		built      string
		warn       bool
	}{
		{true, "go1.25.0", true},
		{true, "go1.26.8", true},
		{true, "devel go1.27-abcdef", true},
		{true, goversion.RecommendedMinimum, false},
		{true, "go1.27.2", false},
		{false, "go1.25.0", false},
	}
	for _, c := range cases {
		msg := toolchainWarning(c.production, c.built)
		if (msg != "") != c.warn {
			t.Fatalf("production=%v built=%q: warning %q", c.production, c.built, msg)
		}
		if c.warn && (!strings.Contains(msg, c.built) || !strings.Contains(msg, goversion.RecommendedMinimum)) {
			t.Fatalf("warning does not name both versions: %q", msg)
		}
	}
}
