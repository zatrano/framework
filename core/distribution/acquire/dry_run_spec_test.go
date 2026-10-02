package acquire

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed ORCHESTRATION.md
var orchestrationSpec []byte

func orchestrationSpecText(t *testing.T) string {
	t.Helper()
	spec := string(orchestrationSpec)
	if strings.TrimSpace(spec) == "" {
		t.Fatal("ORCHESTRATION.md must not be empty")
	}
	return spec
}

func TestDryRunSpecAcceptsContractA(t *testing.T) {
	spec := orchestrationSpecText(t)
	for _, want := range []string{
		"# Acquisition / Enablement Contracts",
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
			t.Fatalf("ORCHESTRATION.md missing %q", want)
		}
	}
}

func TestAcquireEnablementStaysExplicit(t *testing.T) {
	spec := orchestrationSpecText(t)
	for _, want := range []string{
		"Contract B — CLI Acquisition",
		"Contract B complete",
		"Contract C — Acquisition ↔ Enablement",
		"Contract C OPEN",
		"MUST NOT enable automatically",
		"--enable",
		"not_requested",
		"package:install",
		"implicit transaction",
		"renamed equivalent",
	} {
		if !strings.Contains(spec, want) {
			t.Fatalf("ORCHESTRATION.md missing %q", want)
		}
	}
}
