# Phase 9 — Bounding SPEC

**Status:** SPEC accepted — Contract A complete / FROZEN; Contract B complete (IMPLEMENTED); Contract C closed  
**Prerequisite:** Phase 8 (`v2.0.22`) FROZEN  
**Acceptance:** ACCEPTED  
**Implementation:** Contract A (Dry-run) COMPLETE / FROZEN. Contract B (CLI acquisition) IMPLEMENTED. Contract C (Acquisition ↔ enablement) CLOSED.

```text
Contract A — Dry-run              COMPLETE / FROZEN
Contract B — CLI Acquisition      IMPLEMENTED
Contract C — Acquisition ↔ Enablement  CLOSED
Phase 8 — v2.0.22                FROZEN
```

The next step is not automatically Contract C. No Contract C implementation until Contract C is explicitly opened.

---

## 1. Scope

Phase 9 defines three independent contracts:

1. **Contract A — Dry-run**
2. **Contract B — CLI Acquisition**
3. **Contract C — Acquisition ↔ Enablement**

The SPEC is accepted. Contract A is complete / FROZEN. Contract B is implemented. Contract C remains closed. The next step is not automatically Contract C.

---

# Contract A — Dry-run

## Purpose

Show what an acquisition would do without performing any acquisition mutation.

## MUST

Dry-run:

* operates on an existing concrete acquisition plan/target.
* shows the module path.
* shows the concrete version/ref.
* shows the `GoGetArg` value.
* shows the target module root.
* shows the form of the acquisition command that would run.
* does not call the execution boundary.
* does not mutate the filesystem.
* does not change `go.mod`.
* does not change `go.sum`.

## MUST NOT

Dry-run:

* MUST NOT run `go get`.
* MUST NOT run `go mod tidy`.
* MUST NOT run recovery.
* MUST NOT perform enablement.
* MUST NOT run `package:install`.
* MUST NOT perform package registry resolution.
* MUST NOT resolve `latest`.
* MUST NOT apply a semver algorithm.
* MUST NOT re-implement `GoGetArg` production.
* MUST NOT report the acquisition result as installed / acquired / applied.

## Principle

Dry-run is not a second version of execution.

It consumes the same acquisition plan:

```text
Resolve
  ↓
FromResult
  ↓
Targets
  ↓
Plan
  ├── Dry-run
  └── Execute
```

The difference between dry-run and execution is the mutation boundary, not the plan.

## Example

Concrete plan:

```text
module: github.com/zatrano/packages
version: main
arg: github.com/zatrano/packages@main
root: /project
```

Dry-run may report this; it MUST NOT run `go get`.

---

# Contract B — CLI Acquisition

## Purpose

A user-invoked acquisition workflow that orchestrates existing acquisition APIs.

The CLI does not own a second acquisition engine.

## CLI MUST

The CLI:

* uses the existing registry resolution result.
* uses the existing `FromResult` / `Targets` / `Plan` flow.
* uses the existing `GoGetArg` value.
* uses the existing execution boundary.
* reports the acquisition outcome to the user.
* reports a partial acquisition outcome explicitly.
* MAY report the recovery outcome explicitly.
* if dry-run is supported, uses the same acquisition plan.

## CLI MUST NOT

The CLI:

* MUST NOT contain a second resolver.
* MUST NOT perform semver resolution.
* MUST NOT copy the registry algorithm.
* MUST NOT invent a module-normalization algorithm.
* MUST NOT invent a conflict-resolution algorithm.
* MUST NOT introduce a second abstraction that executes `go get` itself.
* MUST NOT edit `go.mod` / `go.sum` directly.
* MUST NOT invent a second acquisition state model.

## Separation

```text
User
 ↓
CLI
 ↓
Existing Resolution / Planning APIs
 ↓
Existing Acquisition Execution Boundary
 ↓
Inspect / Result
```

The CLI's job is orchestration + presentation.

---

# Contract C — Acquisition ↔ Enablement

This contract separates two state transitions.

**Acquisition ≠ Enablement**

## Acquisition

Changes Go module dependency state:

```text
module dependency
       ↓
go get
       ↓
go.mod / go.sum
```

## Enablement

Changes package runtime state:

```text
package
  ↓
import / registration / enabled state
```

## MUST

A successful acquisition MUST NOT enable automatically.

An acquisition failure MUST NOT start enablement.

An enablement failure is not acquisition rollback.

There is no implicit transaction between acquisition and enablement.

## `package:install`

`package:install` does not become an acquisition command in this phase.

Its meaning remains **enablement**.

## State isolation

| Step | Acquisition | Enablement |
|------|-------------|------------|
| Start | Not acquired | Not enabled |
| Acquisition succeeded | Acquired | Not enabled |
| Enablement succeeded | Acquired | Enabled |
| Acquisition failed | Not acquired | Not enabled |
| Enablement failed | Acquired | Not enabled |

This SPEC does not guarantee automatic rollback of acquisition in the last case.

## Future combined workflow

A combined user workflow such as `Acquire + Enable` MAY be designed later. That requires a separate contract. Phase 9 does not define it.

Contract C remains CLOSED. The next step is not automatically Contract C. No Contract C implementation until Contract C is explicitly opened.

---

# Phase 8 Freeze Boundary

Phase 9 MUST NOT touch the following Phase 8 surface:

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

The following MUST NOT change:

* Phase 7 APIs
* `GoGetArg`
* `Execute`
* `ExecuteTargets`
* `Inspect`
* recovery semantics
* fail-fast behavior
* the partial-result model
* concurrency / lock behavior
* the Go tooling mutation boundary

In particular, **`func Apply`** is forbidden.

A new Apply variant or renamed equivalent is also not accepted under this SPEC.

---

# Phase 9 Global Invariants

A Phase 9 implementation:

* MUST NOT create a new resolver.
* MUST NOT create a second process abstraction.
* MUST NOT add `func Apply`.
* MUST NOT change `package:install` semantics.
* MUST NOT add `zatrano.lock`.
* MUST NOT add automatic `go mod tidy`.
* MUST NOT change the Phase 7 acquisition planning API.
* MUST NOT change Phase 8 execution semantics.
* MUST NOT turn acquisition and enablement into an implicit transaction.
* MUST NOT collapse A/B/C into one combined contract.

---

# Implementation Gate

The SPEC is accepted. Implementation order:

```text
Contract A — Dry-run              COMPLETE / FROZEN
        ↓
Contract B — CLI Acquisition      IMPLEMENTED
        ↓
Contract C — Acquisition ↔ Enablement  CLOSED
```

Phase 8 (`v2.0.22`) stays FROZEN.

Each contract: contract tests → implementation → verification.

Contract A is complete / FROZEN. Contract B is complete (IMPLEMENTED). Contract C remains closed. The next step is not automatically Contract C. Do not start acquisition ↔ enablement wiring until Contract C is explicitly opened.

## Current status

```text
Phase 9
Status: SPEC ACCEPTED
Implementation: Contract A COMPLETE / FROZEN; Contract B IMPLEMENTED; Contract C CLOSED
Acceptance: ACCEPTED
Next: not automatically Contract C
```
