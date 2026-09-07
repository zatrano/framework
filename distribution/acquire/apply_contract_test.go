package acquire

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed APPLY.md
var applySpec []byte

func applySpecText(t *testing.T) string {
	t.Helper()

	spec := string(applySpec)
	if strings.TrimSpace(spec) == "" {
		t.Fatal("APPLY.md must not be empty")
	}

	return spec
}

func requireSpecContains(t *testing.T, spec, clause string) {
	t.Helper()

	if !strings.Contains(spec, clause) {
		t.Fatalf("APPLY.md is missing required contract clause: %q", clause)
	}
}

func TestApplySpecDeclaresPhase8Boundary(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "# Phase 8 — Module Acquisition Apply")
	requireSpecContains(t, spec, "Previous phases: 1–7 frozen")
	requireSpecContains(t, spec, "Phase 7 boundary: `GoGetArg`")
	requireSpecContains(t, spec, "This phase: Apply contract only")
}

func TestApplySpecDoesNotRedefineResolution(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Apply MUST NOT become another resolver.")
	requireSpecContains(t, spec, "Apply treats Phase 7 as authoritative.")
	requireSpecContains(t, spec, "It MUST NOT call `Resolve`, `Search`, `Lookup`, `compareSemver`, or `latestCompatible`")
	requireSpecContains(t, spec, "`latest`")
	requireSpecContains(t, spec, "MUST NOT receive `latest`")
}

func TestApplySpecRequiresConcreteGoGetArguments(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "concrete `module@version` strings")
	requireSpecContains(t, spec, "module@v1.4.0")
	requireSpecContains(t, spec, "github.com/zatrano/packages@main")
	requireSpecContains(t, spec, "github.com/zatrano/framework/v2@main")
	requireSpecContains(t, spec, "MUST NOT replace a concrete version with `latest`")
}

func TestApplySpecDefinesGoToolingAsMutationBoundary(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Permitted surface: `go.mod` and `go.sum`")
	requireSpecContains(t, spec, "MUST NOT edit those files as arbitrary text")
	requireSpecContains(t, spec, "`go get <concrete-module>@<concrete-version>`")
	requireSpecContains(t, spec, "MUST NOT ship its own Go module parser/editor")
}

func TestApplySpecPreservesMainSemantics(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "`main` is a valid target")
	requireSpecContains(t, spec, "Apply passes `@main` to Go as-is.")
	requireSpecContains(t, spec, "MUST NOT invent a ZATRANO pseudo-version")
}

func TestApplySpecPreservesSharedModuleNormalization(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Apply operates on acquisition targets, not catalog names.")
	requireSpecContains(t, spec, "MUST NOT `go get` the same module twice")
	requireSpecContains(t, spec, "session@main")
	requireSpecContains(t, spec, "auth@main")
	requireSpecContains(t, spec, "github.com/zatrano/packages@main")
}

func TestApplySpecRejectsConflictingPins(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Phase 7 already rejects two pins for one module.")
	requireSpecContains(t, spec, "If a conflict reaches Apply, it is invalid input: no mutation.")
	requireSpecContains(t, spec, "MUST NOT invent last-write-wins")
	requireSpecContains(t, spec, "first-write-wins")
	requireSpecContains(t, spec, "highest-version-wins")
	requireSpecContains(t, spec, "main-wins")
}

func TestApplySpecDefinesFailFastMutationFailure(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Initial contract: fail-fast.")
	requireSpecContains(t, spec, "Stop on the first mutation failure.")
	requireSpecContains(t, spec, "MUST NOT continue and report success.")
	requireSpecContains(t, spec, "A failed mutation MUST NOT be reported as a successful acquisition.")
}

func TestApplySpecPreservesGoErrors(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Surface command failure, exit status, relevant stderr")
	requireSpecContains(t, spec, "Apply MUST NOT fabricate or repair checksums.")
	requireSpecContains(t, spec, "preserving the underlying reason")
}

func TestApplySpecDoesNotRunTidyAutomatically(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Apply MUST NOT treat `go mod tidy` as pinning or resolution.")
	requireSpecContains(t, spec, "No automatic tidy.")
}

func TestApplySpecDefinesConcurrencyBoundary(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "At most one Apply may mutate a given application's go.mod / go.sum at a time.")
	requireSpecContains(t, spec, "Concurrent Apply against different module roots MAY proceed independently.")
}

func TestApplySpecDefinesPartialApplyVisibility(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "A success, B success, C failure, D unattempted")
	requireSpecContains(t, spec, "Successful, failed, and unattempted targets MUST be observable")
	requireSpecContains(t, spec, "MUST NOT claim “all targets acquired”")
}

func TestApplySpecSeparatesRollbackFromRecovery(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Rollback MUST NOT be assumed transactional.")
	requireSpecContains(t, spec, "Distinguish rollback guaranteed from rollback unavailable / recovery required.")
	requireSpecContains(t, spec, "Best-effort cleanup MUST NOT be presented as transactional rollback.")
}

func TestApplySpecRestrictsFilesystemScope(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "only in the explicitly supplied application/module root")
	requireSpecContains(t, spec, "MUST NOT scan parent trees")
	requireSpecContains(t, spec, "MUST NOT mutate unrelated repos")
	requireSpecContains(t, spec, "MUST NOT write those locations itself")
}

func TestApplySpecKeepsEnablementSeparate(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Apply MUST NOT modify `bootstrap/enabled.go`")
	requireSpecContains(t, spec, "Acquire module ≠ Enable package")
	requireSpecContains(t, spec, "A successful Apply does not enable a package.")
	requireSpecContains(t, spec, "Existing `package:install` stays enablement")
	requireSpecContains(t, spec, "MUST NOT silently redefine it.")
}

func TestApplySpecForbidsZatranoLockfile(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Apply MUST NOT create `zatrano.lock`.")
	requireSpecContains(t, spec, "Authoritative state remains go.mod + go.sum.")
}

func TestApplySpecRequiresStructuredProcessInvocation(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "MUST NOT build a shell string")
	requireSpecContains(t, spec, `exec.Command("go", "get", concreteGoGetArg)`)
	requireSpecContains(t, spec, "No shell interpolation.")
}

func TestApplySpecRequiresContextCancellation(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "Apply MUST accept `context.Context`.")
	requireSpecContains(t, spec, "Cancellation MUST reach the `go` process.")
	requireSpecContains(t, spec, "MUST NOT leave an orphaned `go` process")
}

func TestApplySpecDefinesObservabilityContract(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "module root")
	requireSpecContains(t, spec, "target module")
	requireSpecContains(t, spec, "requested version")
	requireSpecContains(t, spec, "exit status")
	requireSpecContains(t, spec, "stdout/stderr")
	requireSpecContains(t, spec, "mutation result")
	requireSpecContains(t, spec, "MUST NOT log credentials, tokens, or unrelated environment.")
}

func TestApplySpecFreezesImplementationOrder(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "1. Apply contract tests")
	requireSpecContains(t, spec, "2. Process invocation boundary")
	requireSpecContains(t, spec, "3. `go get` execution")
	requireSpecContains(t, spec, "4. go.mod / go.sum inspection")
	requireSpecContains(t, spec, "5. Concurrency enforcement")
	requireSpecContains(t, spec, "6. Partial-apply reporting")
	requireSpecContains(t, spec, "7. Rollback / recovery behavior")
	requireSpecContains(t, spec, "8. Integration tests")
}

func TestApplySpecFreezesPreviousPhaseAPIs(t *testing.T) {
	spec := applySpecText(t)

	requireSpecContains(t, spec, "MUST NOT modify `package:install`, `Resolve`, `FromResult`, `Plan`, `Targets`, or `GoGetArg`.")
}

func TestApplySpecArchitectureInvariants(t *testing.T) {
	spec := applySpecText(t)

	invariants := []string{
		"CLI does not resolve packages.",
		"Apply does not resolve packages.",
		"`FromResult` does not resolve packages.",
		"Plan holds concrete acquisition targets.",
		"`GoGetArg` contains no `latest`.",
		"`Targets` does not select pins.",
		"Conflicting pins are errors.",
		"Apply does not modify enablement.",
		"Apply does not create a ZATRANO lockfile.",
		"Apply does not implement its own Go module resolver.",
		"Apply does not automatically run `go mod tidy`.",
		"Phase 7 remains filesystem-free.",
	}

	for _, invariant := range invariants {
		requireSpecContains(t, spec, invariant)
	}
}
