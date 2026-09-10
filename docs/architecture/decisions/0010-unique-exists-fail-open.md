# ADR-0010 — Database-backed `unique` / `exists` must fail closed

- **Status:** Accepted (amended Phase 4.5)
- **Date:** 2026-09-10
- **Amendment:** 2026-09-10 (Phase 4.5)

The filename `0010-unique-exists-fail-open.md` is historical. Fail-open is **not** current behavior.

## Historical context (Phase 2)

`packages/validation` implements `unique` and `exists` via `PresenceChecker`. Through Phase 4, if neither the validator nor `SetDefaultPresenceChecker` had a checker, `checkPresence` **returned true** (the rule passed). Malformed `unique:users` (no column) also returned true. If a checker was set and it returned an error, the rule already **failed**.

Phase 2 classified that as a documented limitation and a correctness problem, not a security control and not authorization. Doctor (Phase 3) added **error** APP-VAL-001 when those rule names appear as string literals and `database` is not enabled. Concatenated strings remain a documented SEMANTIC bypass of the doctor check. The runtime was intentionally left unchanged until this validation-package amendment.

## Problem

Applications could ship `unique:users,email` or `exists:posts,id` and believe they had uniqueness or existence protection while the required database fact could not be established. Duplicate rows could insert. Client-supplied ids could “exist.” A database error is not the same as “record not found.”

## Root cause

Fail-open was introduced in `packages/validation` `checkPresence`:

1. `checker == nil` (instance and default) → `return true`
2. `len(parts) < 2` (missing table or column) → `return true`

A second path existed in `packages/database` `boot()`: empty table or column returned `(false, nil)`, which made `unique` succeed (`!exists`). Checker errors already failed the rule.

## Decision

Database-backed validation is fail-closed.

A database/checker/infrastructure failure must never
produce a successful validation result.

Invariant:

> A `unique` or `exists` validation rule MUST NEVER report success when its required database check could not be reliably completed.

This includes a missing checker, a checker error, an ORM/query/connection failure, and an incomplete `table,column` rule.

No new public validation API. `RuleFunc` still returns `bool`. Infrastructure failure is represented as a validation failure (`Fails()` / `ValidationException`), using the existing unique/exists messages. That is fail-closed, not a new error hierarchy.

| Situation | Rule result |
|---|---|
| Lookup succeeds, unique value absent | PASS |
| Lookup succeeds, unique value present | VALIDATION FAILURE |
| Lookup succeeds, exists value present | PASS |
| Lookup succeeds, exists value absent | VALIDATION FAILURE |
| No PresenceChecker | MUST NOT PASS |
| Checker / query / connection error | MUST NOT PASS |
| Malformed rule (no column, empty table/column) | MUST NOT PASS |
| Empty value without `required` | skip (unchanged; combine with `required`) |
| Extra CSV parts (`unique:users,email,id,5`) | still table+column+value only (ignore-ID was not implemented) |

`database` still binds `SetDefaultPresenceChecker` at boot. Applications that declare `unique` or `exists` MUST enable `database`. Empty table/column in that checker now returns an error, not `(false, nil)`.

Never use `exists:` as IDOR protection. Ownership remains Policy + `Find`.

## Fail-closed semantics

```text
                 Database fact
                      │
          ┌───────────┴───────────┐
          │                       │
       established             unknown
          │                       │
      evaluate rule            REJECT
```

Uncertain → reject. Not: everything → reject. Successful lookups still pass or fail according to the rule.

## Error propagation

`checkPresence` does not ignore checker errors, default a missing checker to pass, or treat an incomplete rule as pass. `ValidateForm` → `Make` → `Fails()` uses the same path. `sql.ErrNoRows` in the database checker remains `(false, nil)`: that is an established “not found,” not an infrastructure failure.

## Security / correctness rationale

Fail-open converted “we could not look this up” into “this value is unique / this id exists.” Fail-closed converts uncertainty into a 422 validation error. That is conservative under uncertainty. It is still not authorization.

## Compatibility

- Kernel, contracts, ORM public API, FormRequest taxonomy, doctor catalog: unchanged.
- `PresenceChecker` / `SetPresenceChecker` / `SetDefaultPresenceChecker`: unchanged signatures.
- Behavior change: apps or tests that asserted `unique`/`exists` **pass** with no checker will now fail. That was the hole.
- APP-VAL-001 remains a **structural** CI check (literals + `database` enabled). Doctor does not prove runtime SQL. Without `database`, those rules now fail closed at runtime (always-fail), which is why the doctor rule stays useful.

## Tests

`packages/validation` `presence_test.go` and FormRequest `ValidateForm` tests cover unique/exists present, absent, checker error, checker unavailable, malformed rules, extra CSV parts (table+column only; ignore-ID was not implemented), and the HTTP FormRequest path.

Closure verification: [phase4.5.md](../phase4.5.md). `go test -race` is **NOT EXECUTABLE** in the recorded environment (GCC/CGO unavailable); that is not a repository failure.

## Consequences

Product and Order golden scenarios that use `exists:products,id` still require `database` enabled and a working checker. Tests that call `ValidateForm` without a checker must not expect uniqueness or existence to pass.

## Enforcement

- Runtime: `checkPresence` fail-closed; database default checker errors on empty table/column.
- Doctor: APP-VAL-001 unchanged structurally (`unique`/`exists` literals ⇒ `database` in `EnabledAddons`).
- Report: [phase4.5.md](../phase4.5.md).
