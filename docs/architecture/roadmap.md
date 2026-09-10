# Implementation roadmap

**Boundary:** kernel, contracts, ORM runtime, public API, and ABI stay frozen unless a separate review authorizes a change.

---

## Phase 0 — Review

- ADRs 0001–0008 **Accepted**.
- HTMX remains out of scope.
- Repository optional / FormRequest mandatory for writes.

## Phase 1 — Generators match STANDARD

Done. See git history (`chore: align generators with application standard`).

## Phase 2 — Golden applications & STANDARD hardening

**Done (documentation only).** [golden.md](golden.md), [phase2.md](phase2.md), ADRs 0009–0010.

- Exact CRUD traces, service decision table, Policy fluent API, dual controllers, unique/exists classification.
- No kernel / contracts / ORM / generator changes.
- No consumer demo app in this repository.

## Phase 3 — Machine-enforced architecture

**Done.** Extend `zatrano doctor` (not a second CLI). Errors exit 1. `--json` / `--strict`. Catalog: [rules.md](rules.md). Report: [phase3.md](phase3.md).

## Phase 3.5 — Adversarial verification

**Done.** Prove bypasses vs false positives. Analyzer-only closures: [phase3.5.md](phase3.5.md). Kernel/ORM/API untouched. Known remaining PASS paths are documented limitations, not silent bugs.

## Phase 4 — Standard freeze

**Done.** [STANDARD.md](STANDARD.md) is frozen ([ADR-0011](decisions/0011-application-engineering-standard-freeze.md)). Completeness: [completeness-matrix.md](completeness-matrix.md). Ambiguity audit: [no-second-way.md](no-second-way.md). Report: [phase4.md](phase4.md).

**Still do not:**

- Nested transactions, cursor pagination, typed ErrModelNotFound, query context — those are **ORM package** ADRs, not app-standard patches.
- `func Apply` / acquire redesign / `contracts.App` growth / HTMX kernel helpers.
- Mandatory repositories or UseCase generators.
- Automatic enable after acquire.
- Speculative doctor heuristics to drive phase 3.5 bypasses to zero.

## Phase 4.5 — Validation runtime correctness

**Done.** `unique` / `exists` fail closed when the database fact cannot be established ([phase4.5.md](phase4.5.md), ADR-0010 amended). No new doctor rule. Kernel/ORM/API untouched.

## Phase 5 — Examples repo (optional)

- One golden app per flow in `github.com/zatrano/examples`, built **only** with accepted STANDARD + golden.md.
- Not a second architecture.

---

## Suggested review order

1. ADR-0001 layers  
2. ADR-0002 FormRequest  
3. ADR-0003 repositories  
4. ADR-0004 transactions  
5. ADR-0005 HTMX  
6. ADR-0006 authorization  
7. ADR-0007 controller generator  
8. ADR-0008 JSON vs jsonapi  
9. ADR-0009 dual controllers  
10. ADR-0010 unique/exists (fail-closed, Phase 4.5)  
11. ADR-0011 STANDARD freeze
