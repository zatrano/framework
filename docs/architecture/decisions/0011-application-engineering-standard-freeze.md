# ADR-0011 — Application Engineering Standard freeze

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

Review, golden scenarios, generator alignment, and doctor work produced ADRs 0001–0010, `zatrano doctor`, and an adversarial enforcement boundary. Kernel, contracts, ORM, and public API were already frozen. Application architecture was still described as “evolving documentation.”

## Problem

A competent human or AI can still ask whether a second layout, UseCase-shaped service, or mixed Web/API controller is “also ZATRANO.” Undocumented alternatives become a second architecture.

## Decision

[`docs/architecture/STANDARD.md`](../STANDARD.md) is the **frozen** Application Engineering Standard.

- One canonical implementation path per concern, with explicit decision tables where a choice is intentionally valid (simple CRUD vs service; optional concrete repository).
- The six doctor-boundary doctor-PASS stacks remain **documented SEMANTIC boundaries**, not silent holes and not a mandate to add fragile AST rules.
- Kernel / contracts / ORM public API / ABI stay frozen. Runtime fail-open `unique`/`exists` stayed ADR-0010 until the validation-package amendment (completed fail-closed unique/exists; STANDARD freeze unchanged).

Supersedes informal “PROPOSED” language in STANDARD where the decision was already accepted (transaction helper, jobs after commit, bootstrapped `APP_ENV`).

## Why

Determinism for humans, AI agents, generators, doctor, and CI. Completeness without pretending static analysis is a theorem prover.

## Rejected alternatives

- Keep STANDARD as a living sketch until “zero bypasses.”
- Ban `app/core` and `Handler` type names (unacceptable false positives — doctor boundary).
- Reopen ADR-0001–0010 to chase fashion (fashionable extra layers, fragment views, mandatory repos).

## Consequences

Changes to application architecture require a new ADR that **supersedes** a named previous ADR. Additive package features do not create new application layers.

## Enforcement

Documentation freeze + existing doctor catalog + `go test ./...`. No new speculative doctor rules in this ADR.
