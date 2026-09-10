# Implementation status

**Boundary:** kernel, contracts, ORM runtime, public API, and ABI stay frozen unless a separate review authorizes a change.

---

## Review (done)

- ADRs 0001–0008 **Accepted**.
- HTML fragment / partial-swap clients remain out of scope.
- Repository optional / FormRequest mandatory for writes.

## Generators match STANDARD (done)

See git history (`chore: align generators with application standard`).

## Golden applications and STANDARD hardening (done)

Documentation only. [golden.md](golden.md), [golden-report.md](golden-report.md), ADRs 0009–0010.

- Exact CRUD traces, service decision table, Policy fluent API, dual controllers, unique/exists classification.
- No kernel / contracts / ORM / generator changes.
- No consumer demo app in this repository.

## Machine-enforced architecture (done)

Extend `zatrano doctor` (not a second CLI). Errors exit 1. `--json` / `--strict`. Catalog: [rules.md](rules.md). Report: [doctor-report.md](doctor-report.md).

## Adversarial verification (done)

Prove bypasses vs false positives. Analyzer-only closures: [doctor-boundary.md](doctor-boundary.md). Kernel/ORM/API untouched. Known remaining PASS paths are documented limitations, not silent bugs.

## Standard freeze (done)

[STANDARD.md](STANDARD.md) is frozen ([ADR-0011](decisions/0011-application-engineering-standard-freeze.md)). Completeness: [completeness-matrix.md](completeness-matrix.md). Ambiguity audit: [no-second-way.md](no-second-way.md). Report: [freeze-report.md](freeze-report.md).

**Still do not:**

- Nested transactions, cursor pagination, typed ErrModelNotFound, query context — those are **ORM package** ADRs, not app-standard patches.
- `func Apply` / acquire redesign / `contracts.App` growth / fragment kernel helpers.
- Mandatory repositories or UseCase generators.
- Automatic enable after acquire.
- Speculative doctor heuristics to drive semantic bypasses to zero.

## Validation runtime correctness (done)

`unique` / `exists` fail closed when the database fact cannot be established ([unique-exists-runtime.md](unique-exists-runtime.md), ADR-0010 amended). No new doctor rule. Kernel/ORM/API untouched.

## Platform conformance and release audit (done)

Forensic audit of framework + packages HEAD: [platform-audit.md](platform-audit.md). Recommendation **GO WITH CONDITIONS** (next public tag must not reuse `v2.1.0`; packages fail-closed must be tagged; dirty packages WIP must not ship). Architecture not redesigned.

## Examples repo (optional)

- One golden app per flow in `github.com/zatrano/examples`, built **only** with accepted STANDARD + golden.md.
- Not a second architecture.

---

## Suggested review order

1. ADR-0001 layers
2. ADR-0002 FormRequest
3. ADR-0003 repositories
4. ADR-0004 transactions
5. ADR-0005 fragment views
6. ADR-0006 authorization
7. ADR-0007 controller generator
8. ADR-0008 JSON vs jsonapi
9. ADR-0009 dual controllers
10. ADR-0010 unique/exists (fail-closed)
11. ADR-0011 STANDARD freeze
