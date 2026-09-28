package acquire

import (
	"strings"
	"testing"
)

func TestAcquireEnablementWorkflowIsExplicit(t *testing.T) {
	spec := orchestrationSpecText(t)
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
			t.Fatalf("ORCHESTRATION.md missing %q", want)
		}
	}
	if strings.Contains(spec, "Contract C remains CLOSED") {
		t.Fatal("ORCHESTRATION.md still treats Contract C as closed")
	}
}
