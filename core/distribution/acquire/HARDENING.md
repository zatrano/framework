# Acquisition Production Hardening

**Status:** Accepted
**Prerequisite:** Acquisition/enablement contracts (`v2.0.27`) implemented and verified
**Acceptance:** ACCEPTED
**Implementation:** COMPLETE

---

## 1. Purpose

Acquisition hardening does not introduce a new acquisition architecture.

Its purpose is to harden and validate the existing package lifecycle against real operational conditions:

```text
Search
  ↓
Info
  ↓
Resolve
  ↓
Acquire
  ↓
optional --enable
  ↓
runtime / boot
```

This contract addresses the gaps identified in the `v2.0.27` repository audit:

* end-to-end lifecycle validation;
* CLI failure semantics;
* exit-code semantics;
* JSON output completeness;
* recovery reporting;
* timeout and cancellation;
* `package:enable` / `--enable` semantic consistency;
* official package ecosystem validation;
* heavy-package validation;
* `framework_min` validation;
* tagged-release / `main` acquisition validation;
* CI-level acquisition verification.

Acquisition hardening MUST NOT reopen the Apply contract or redesign orchestration contracts.

---

# 2. Frozen Boundaries

The following remain immutable:

### Apply contract

Apply contract (`v2.0.22`) remains frozen.

The existing acquisition surface remains authoritative:

```text
FromResult
    ↓
Targets
    ↓
GoGetArg
    ↓
Execute / ExecuteTargets
    ↓
Inspect
    ↓
ApplyResult
    ↓
SnapshotFiles / RecoverFiles
```

No hardening feature may create a replacement for this flow.

### Apply

The following remains forbidden:

```go
func Apply
```

A renamed, wrapped, disguised, or semantically equivalent `Apply` abstraction is also forbidden.

### Contract A

Dry-run remains frozen.

It continues to:

* consume the existing concrete acquisition plan;
* report the existing `GoGetArg`;
* avoid execution;
* avoid mutation;
* avoid recovery;
* avoid enablement;
* reject `latest`.

Acquisition hardening MUST NOT redesign `DryRun` or `DryRunTargets`.

### Contract B

`package:acquire` remains CLI orchestration.

The CLI must continue to delegate acquisition responsibilities to existing APIs.

No second resolver or acquisition engine may be introduced.

### Contract C

Explicit acquisition ↔ enablement semantics remain:

```text
package:acquire NAME
    → acquisition only

package:acquire NAME --enable
    → acquisition
    → explicit enablement after successful acquisition
```

`package:install` remains enablement.

No implicit acquisition → enablement transition may be introduced.

---

# 3. Acquisition hardening Scope

This hardening specification contains the following independent hardening areas:

### A. End-to-End Lifecycle

### B. Failure Matrix

### C. CLI Result and Exit Semantics

### D. Recovery Reporting

### E. Timeout and Cancellation

### F. Enablement Semantic Consistency

### G. Real Ecosystem Validation

### H. CI Acquisition Validation

These areas are related operationally but MUST NOT create a new central lifecycle abstraction.

---

# 4. Contract A — End-to-End Lifecycle Validation

## Objective

Prove the complete package lifecycle through the real CLI boundaries rather than only through isolated package tests.

The target lifecycle is:

```text
package:search
      ↓
package:info
      ↓
package:resolve
      ↓
package:acquire
      ↓
package:acquire --enable
      ↓
package boot
```

The tests MUST distinguish discovery from resolution and resolution from acquisition.

## MUST

At least one end-to-end scenario MUST use an official package/module path.

At least one scenario MUST exercise:

```text
catalog → Resolve → package:acquire
```

At least one scenario MUST exercise:

```text
package:acquire --enable → enablement → boot
```

Where practical, tests MUST cover both:

* shared official module packages;
* heavy packages with their own module.

The E2E layer MUST use the real CLI orchestration path.

It MUST NOT replace the complete acquisition path with an injected fake when the purpose of the test is real acquisition validation.

Existing unit tests with fakes remain valid and are not replaced.

---

# 5. Contract B — Failure Matrix

Acquisition hardening MUST explicitly validate CLI behavior for the following classes:

| Failure / State              | Required Result                                  |
| ---------------------------- | ------------------------------------------------ |
| Unknown package              | resolution failure                               |
| Incompatible package/version | resolution failure                               |
| Conflicting module pins      | planning failure                                 |
| Network / `go get` failure   | acquisition failure                              |
| Partial acquisition          | success/failed/unattempted distinction preserved |
| Snapshot failure             | recovery unavailable / explicit failure          |
| Recovery failure             | recovery failure explicitly reported             |
| Already-present module       | deterministic existing behavior                  |
| Already-enabled package      | deterministic existing behavior                  |
| Enablement failure           | acquisition remains successful                   |
| Cancelled acquisition        | cancellation explicitly reported                 |
| Timeout                      | timeout/cancellation explicitly reported         |

The CLI MUST NOT convert distinct failure classes into misleading generic success.

The existing acquisition semantics remain authoritative.

---

# 6. Contract C — Exit Codes

Acquisition hardening may introduce classified CLI exit semantics.

The exit-code design MUST be deterministic and documented.

At minimum, the CLI MUST distinguish:

```text
success
usage / argument error
resolution failure
planning failure
acquisition failure
enablement failure
cancellation / timeout
```

The implementation MUST NOT introduce exit-code logic into the acquisition package merely for CLI presentation.

Exit-code interpretation belongs to the CLI boundary.

The existing successful acquisition semantics MUST remain unchanged.

A partial acquisition MUST NOT be reported as complete success if the selected workflow requires all targets to succeed.

---

# 7. Contract D — JSON Output

The existing:

```text
--format=json
```

surface must become operationally useful without creating a second state model.

JSON output MUST expose the same semantic result already available to the CLI.

Where applicable it MUST distinguish:

```text
acquisition
enablement
inspection
recovery
targets
errors
```

For acquisition targets, the output MUST preserve:

```text
success
failed
unattempted
```

distinction.

`Inspect` output MUST NOT be silently discarded when the CLI workflow requires it.

The JSON representation MUST be a presentation of existing state, not a new acquisition state machine.

---

# 8. Contract E — Recovery Reporting

Current behavior:

```text
rec, _ := c.runRecover
```

is insufficient because recovery failure is operationally significant.

Acquisition hardening MUST ensure that recovery errors are observable.

Required distinction:

```text
Acquisition failed
Recovery not attempted

Acquisition failed
Recovery attempted successfully

Acquisition failed
Recovery attempted and failed
```

A recovery failure MUST NOT be silently discarded.

Snapshot failure MUST remain distinguishable from recovery failure.

Recovery remains explicit.

There is still no transactional guarantee.

Acquisition hardening MUST NOT claim that restoring:

```text
go.mod
go.sum
```

rolls back all effects of Go tooling.

---

# 9. Contract F — Timeout and Cancellation

The current CLI use of:

```go
context.Background()
```

is insufficient for long-running acquisition operations.

Acquisition hardening MUST establish an explicit context propagation path from CLI invocation into acquisition execution.

The implementation MUST:

* preserve cancellation;
* allow timeout configuration where appropriate;
* terminate acquisition execution through the existing execution boundary;
* report cancellation/timeout distinctly;
* avoid creating a second process abstraction.

The existing `Invoke` / `Runner` boundary remains authoritative.

The CLI MUST NOT call `os/exec` directly.

---

# 10. Contract G — Enablement Consistency

Acquisition/enablement introduced:

```text
package:acquire NAME --enable
```

while:

```text
package:install NAME
```

remains enablement.

Acquisition hardening MUST verify and document whether the two commands intentionally share or differ in:

* wiring;
* environment application;
* addon activation;
* stub publication;
* already-enabled behavior;
* error handling.

The goal is semantic consistency, not forced implementation identity.

A wire failure MUST have a clearly defined meaning.

The CLI MUST NOT classify:

```text
acquisition success
```

as:

```text
enablement success
```

when enablement failed.

No implicit transaction may be introduced.

---

# 11. Contract H — Official Ecosystem Validation

The acquisition lifecycle MUST be validated against the actual ZATRANO package ecosystem.

Validation MUST include, where applicable:

### Official collection

```text
github.com/zatrano/packages
```

### Shared-module packages

At least multiple packages resolving to the same module must be exercised.

### Heavy packages

Heavy packages with independent module paths must be exercised.

### `framework_min`

Packages declaring `framework_min` must be validated against compatible and incompatible framework versions.

### `main`

The existing `@main` resolution path must be validated through the real acquisition workflow.

### Tagged releases

At least one concrete tagged module release must be acquired through the CLI.

The tests MUST verify that the concrete resolved version reaches the same existing `GoGetArg` path.

No new version-selection algorithm may be introduced.

---

# 12. Real Go Tooling

Acquisition hardening MUST distinguish:

```text
fake acquisition tests
```

from:

```text
real Go tooling validation
```

Both remain useful.

Fake tests validate deterministic failure and orchestration behavior.

Real integration tests validate:

* actual `go get`;
* actual `go.mod`;
* actual `go.sum`;
* actual module resolution;
* actual official package paths;
* actual CLI orchestration.

Real tests MUST use isolated temporary module roots.

They MUST NOT mutate the developer's repository dependency state.

---

# 13. CI Validation

Acquisition hardening SHOULD establish a dedicated acquisition E2E CI path.

The CI validation MUST cover the real CLI acquisition workflow without depending on a developer's local working tree.

At minimum:

```text
go test ./...
go vet ./...
staticcheck ./...
```

must remain green.

Acquisition-specific E2E validation should run against isolated temporary modules.

CI MUST NOT require a persistent `zatrano.lock`.

---

# 14. No New Architecture

Acquisition hardening is a production-hardening contract.

It MUST NOT introduce:

* a new acquisition engine;
* a new resolver;
* a new registry;
* a second process abstraction;
* a new transaction manager;
* a lockfile;
* automatic `go mod tidy`;
* automatic rollback semantics;
* a global package service locator;
* a replacement for `ExecuteTargets`;
* a replacement for `Inspect`;
* a replacement for existing recovery APIs.

The preferred implementation is the **smallest possible change at the existing boundary**.

---

# 15. Architecture Test Requirements

Architecture tests MUST enforce:

1. `package:acquire` remains orchestration.
2. No CLI `os/exec`.
3. No second resolver.
4. No second acquisition engine.
5. `package:install` remains enablement.
6. Contract A remains frozen.
7. Apply APIs remain unchanged.
8. `func Apply` remains absent.
9. No renamed Apply equivalent exists.
10. No implicit acquisition → enablement transition exists.
11. JSON does not introduce a second state model.
12. Exit-code logic remains at the CLI boundary.

---

# 16. Implementation Gate

This document is currently:

```text
Acquisition hardening
Status: ACCEPTED
SPEC Acceptance: ACCEPTED
Implementation: COMPLETE
```

Implementation proceeds in controlled increments:

```text
SPEC Acceptance
      ↓
A — E2E Lifecycle
      ↓
B — Failure Matrix
      ↓
C — Exit / JSON Result Semantics
      ↓
D — Recovery Reporting
      ↓
E — Timeout / Cancellation
      ↓
F — Enablement Consistency
      ↓
G — Ecosystem Validation
      ↓
H — CI Validation
```

Each increment follows:

```text
Contract
   ↓
Tests
   ↓
Implementation
   ↓
Verification
```

A later contract MUST NOT silently redefine an earlier completed contract.

---

# 17. Acceptance Checklist

Acquisition hardening SPEC is ready for acceptance only if the following are explicitly understood:

* [ ] The Apply contract remains frozen.
* [ ] Contract A remains frozen.
* [ ] Contract B remains orchestration-only.
* [ ] Contract C remains explicit `--enable`.
* [ ] `package:install` remains enablement.
* [ ] No implicit transaction exists.
* [ ] No `func Apply` exists or may be introduced.
* [ ] No renamed Apply equivalent is permitted.
* [ ] No second acquisition engine is permitted.
* [ ] No second resolver is permitted.
* [ ] Real CLI acquisition is validated, not only faked.
* [ ] Recovery failures are observable.
* [ ] Exit codes are classified at the CLI boundary.
* [ ] JSON exposes existing semantic state.
* [ ] Cancellation/timeout reaches the existing execution boundary.
* [ ] Official packages and heavy modules are validated.
* [ ] `framework_min` is validated.
* [ ] `main` and tagged releases are validated.
* [ ] CI validates acquisition E2E.
* [ ] No lockfile is introduced.
* [ ] No automatic `go mod tidy` is introduced.
* [ ] No automatic rollback transaction is introduced.

---

# 18. Current State

```text
Apply contract — v2.0.22
    FROZEN

Acquisition/enablement — v2.0.27
    COMPLETE

Contract A
    COMPLETE / FROZEN

Contract B
    IMPLEMENTED

Contract C
    IMPLEMENTED

Acquisition hardening
    ACCEPTED
    IMPLEMENTATION COMPLETE

Increments
    A E2E Lifecycle — COMPLETE
    B Failure Matrix — COMPLETE
    C Exit / JSON — COMPLETE
    D Recovery Reporting — COMPLETE
    E Timeout / Cancellation — COMPLETE
    F Enablement Consistency — COMPLETE
    G Ecosystem Validation — COMPLETE
    H CI Validation — COMPLETE
```
