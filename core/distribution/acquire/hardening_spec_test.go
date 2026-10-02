package acquire

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed HARDENING.md
var hardeningSpec []byte

func hardeningSpecText(t *testing.T) string {
	t.Helper()
	spec := string(hardeningSpec)
	if strings.TrimSpace(spec) == "" {
		t.Fatal("HARDENING.md must not be empty")
	}
	return spec
}

func TestAcquisitionHardeningSpecIsAccepted(t *testing.T) {
	spec := hardeningSpecText(t)
	for _, want := range []string{
		"# Acquisition Production Hardening",
		"**Status:** Accepted",
		"**Acceptance:** ACCEPTED",
		"Implementation: COMPLETE",
		"func Apply",
		"package:install",
		"smallest possible change",
		"DryRun",
		"package:acquire",
		"Exit-code interpretation belongs to the CLI boundary",
		"No implicit acquisition → enablement",
		"Apply contract (`v2.0.22`) remains frozen",
		"Acquisition hardening MUST NOT redesign `DryRun` or `DryRunTargets`",
	} {
		if !strings.Contains(spec, want) {
			t.Fatalf("HARDENING.md missing %q", want)
		}
	}
	for _, ban := range []string{
		"**Status:** Draft",
		"NOT ACCEPTED",
		"Implementation: LOCKED",
		"No hardening implementation is authorized",
	} {
		if strings.Contains(spec, ban) {
			t.Fatalf("HARDENING.md still contains %q — SPEC is accepted", ban)
		}
	}
}

func TestAcquisitionHardeningSpecKeepsFrozenBoundaries(t *testing.T) {
	spec := hardeningSpecText(t)
	for _, want := range []string{
		"FromResult",
		"Execute / ExecuteTargets",
		"Inspect",
		"SnapshotFiles / RecoverFiles",
		"Contract B",
		"Contract C",
		"github.com/zatrano/packages",
		"zatrano.lock",
		"go mod tidy",
		"os/exec",
		"context.Background()",
		"rec, _ := c.runRecover",
	} {
		if !strings.Contains(spec, want) {
			t.Fatalf("HARDENING.md missing frozen-boundary marker %q", want)
		}
	}
}
