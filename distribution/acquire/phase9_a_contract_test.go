package acquire

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed PHASE9.md
var phase9Spec []byte

func phase9SpecText(t *testing.T) string {
	t.Helper()
	spec := string(phase9Spec)
	if strings.TrimSpace(spec) == "" {
		t.Fatal("PHASE9.md must not be empty")
	}
	return spec
}

func TestPhase9SpecAcceptsContractA(t *testing.T) {
	spec := phase9SpecText(t)
	for _, want := range []string{
		"# Phase 9 — Bounding SPEC",
		"Acceptance:** ACCEPTED",
		"Contract A complete",
		"Contract A — Dry-run",
		"does not call the execution boundary",
		"MUST NOT run `go get`",
		"MUST NOT resolve `latest`",
		"installed / acquired / applied",
		"func Apply",
		"Acquisition ≠ Enablement",
	} {
		if !strings.Contains(spec, want) {
			t.Fatalf("PHASE9.md missing %q", want)
		}
	}
}

func TestPhase9SpecKeepsContractCClosed(t *testing.T) {
	spec := phase9SpecText(t)
	for _, want := range []string{
		"Contract B — CLI Acquisition",
		"Contract B complete",
		"Contract C — Acquisition ↔ Enablement",
		"Contract C closed",
		"package:install",
		"implicit transaction",
		"renamed equivalent",
	} {
		if !strings.Contains(spec, want) {
			t.Fatalf("PHASE9.md missing %q", want)
		}
	}
}
