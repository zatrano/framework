# ADR-0010 — `unique` / `exists` fail-open without a PresenceChecker

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

`packages/validation` implements `unique` and `exists` via `PresenceChecker`. If neither the validator nor `SetDefaultPresenceChecker` has a checker, `checkPresence` **returns true** (the rule passes). If a checker is set and it returns an error, the rule **fails**.

This is existing package behavior. Phase 2 does **not** change the runtime.

## Problem

Applications can ship `unique:users,email` or `exists:posts,id` and believe they have uniqueness or IDOR protection while `database` is disabled. Duplicate rows insert. Client-supplied ids “exist.”

## Decision

Classify the behavior as a **documented limitation** and a **correctness problem**. It is **not** a substitute for authorization.

| Situation | Rule result | Application duty |
|---|---|---|
| No PresenceChecker | pass (fail-open) | Do not rely on unique/exists |
| Checker error | fail | Treat as validation failure |
| `database` enabled, checker bound | real lookup | Required for these rules |
| Ownership / IDOR | Policy + `Find` | **Never** `exists:` as AuthZ |

Applications that declare `unique` or `exists` **MUST** enable `database` (and therefore the checker the package binds). Doctor (Phase 3): **error** APP-VAL-001 when those rule names appear as string literals and `database` is not enabled. Concatenated strings are a documented SEMANTIC bypass.

Do **not** change `checkPresence` in this phase. A fail-closed default would be a validation-package ADR, not an application-layer workaround.

## Why

Fail-open is surprising but changing it is an ABI/behavior break for apps that validate without a database in unit tests. Documentation + doctor is the Phase 2/3 path.

## Rejected alternatives

- Silently treat this as “intentional and fine” — it is a correctness hole.
- Invent app-level uniqueness checks in services that duplicate the rule — two ways.
- Use `exists:posts,id` instead of Policy — IDOR-adjacent; forbidden.

## Consequences

Product and Order golden scenarios that use `exists:products,id` assume `database` is enabled. Tests that call `ValidateForm` without a checker must not assert uniqueness.

## Enforcement

`package:doctor` / architecture doctor: `unique`/`exists` in FormRequest `Rules()` ⇒ `database` in `EnabledAddons`.
