package acquire

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed PHASE10.md
var phase10Spec []byte

func phase10SpecText(t *testing.T) string {
	t.Helper()
	spec := string(phase10Spec)
	if strings.TrimSpace(spec) == "" {
		t.Fatal("PHASE10.md must not be empty")
	}
	return spec
}

func TestPhase10SpecIsAccepted(t *testing.T) {
	spec := phase10SpecText(t)
	for _, want := range []string{
		"# Phase 10 — Production Hardening & Ecosystem Validation",
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
		"Phase 8 (`v2.0.22`) remains frozen",
		"Phase 10 MUST NOT redesign `DryRun` or `DryRunTargets`",
	} {
		if !strings.Contains(spec, want) {
			t.Fatalf("PHASE10.md missing %q", want)
		}
	}
	for _, ban := range []string{
		"**Status:** Draft",
		"NOT ACCEPTED",
		"Implementation: LOCKED",
		"No Phase 10 implementation is authorized",
	} {
		if strings.Contains(spec, ban) {
			t.Fatalf("PHASE10.md still contains %q — SPEC is accepted", ban)
		}
	}
}

func TestPhase10SpecKeepsFrozenBoundaries(t *testing.T) {
	spec := phase10SpecText(t)
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
			t.Fatalf("PHASE10.md missing frozen-boundary marker %q", want)
		}
	}
}
