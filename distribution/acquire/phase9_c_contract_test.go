package acquire

import (
	"strings"
	"testing"
)

func TestPhase9ContractCWorkflowIsExplicit(t *testing.T) {
	spec := phase9SpecText(t)
	for _, want := range []string{
		"Contract C OPEN",
		"package:acquire NAME --enable",
		"enablement **not requested**",
		"Acquisition: success / failed / not_executed",
		"Enablement:  not_requested / success / failed",
		"NO implicit transaction",
		"MUST NOT enable automatically",
	} {
		if !strings.Contains(spec, want) {
			t.Fatalf("PHASE9.md missing %q", want)
		}
	}
	if strings.Contains(spec, "Contract C remains CLOSED") {
		t.Fatal("PHASE9.md still treats Contract C as closed")
	}
}
