# Phase 4.5 report — Validation runtime correctness (fail-closed unique/exists)

Date: 2026-09-10

STANDARD remains **frozen**. Kernel, contracts, ORM public API, ABI, generators, and the doctor catalog were not redesigned. No new doctor rule. This phase hardens database-backed validation runtime in `github.com/zatrano/packages`.

---

## A. Current root cause

Fail-open lived in `packages/validation` `checkPresence` (`validator.go`).

| Path | Historical behavior | Why it passed |
|---|---|---|
| No `PresenceChecker` (instance and `defaultPresenceChecker` both nil) | `return true` | Missing checker treated as “lookup succeeded with no row” for unique, or “row exists” for exists |
| Malformed rule (`unique:users`, no column) | `return true` | Incomplete fact treated as success |
| Empty table/column after trim (`unique:,email`) | checker called; database boot returned `(false, nil)` | `unique` became `!false` → PASS |
| Checker / query error | `return false` | Already fail-closed |

`ValidateForm` → `Make` → `Fails()` → `checkPresence`. Instance checker is unset, so FormRequest used `SetDefaultPresenceChecker` (wired in `packages/database` `boot()`). Unit tests without that boot hit the nil-checker fail-open.

`sql.ErrNoRows` → `(false, nil)` is **not** fail-open: the query completed and established absence.

`database/query` `Where exists` is SQL EXISTS, not the validation rule. It was not changed.

There is **one** validation `unique`/`exists` implementation. No second fail-open package path was found after the fix.

---

## B. Runtime contract

No new public enum. Existing `RuleFunc bool` + `Fails()` / `ValidationException`:

| Logical state | Representation |
|---|---|
| PASS | `checkPresence` returns true → `Passes()` |
| VALIDATION FAILURE | lookup completed; unique hit or exists miss → field error (`The :attribute has already been taken.` / `The selected :attribute is invalid.`) |
| INFRASTRUCTURE FAILURE | nil checker, checker `err`, malformed/empty table or column → same `Fails()` / `ValidationException` (must not pass) |

Empty value without `required` still **skips** the lookup (combine `required|unique`). Extra CSV parts (`unique:users,email,id,5`) still use only table, column, and value; ignore-ID was not implemented and was not added.

`SetDefaultPresenceChecker` remains the existing package-level binding. This phase does not add another global.

---

## C. UNIQUE matrix

| Condition | Expected | Actual |
|---|---|---|
| Record absent | PASS | PASS |
| Record exists | FAIL validation | FAIL validation |
| DB error | MUST NOT PASS | MUST NOT PASS |
| Checker unavailable | MUST NOT PASS | MUST NOT PASS |
| Malformed / empty table or column | MUST NOT PASS | MUST NOT PASS |

---

## D. EXISTS matrix

| Condition | Expected | Actual |
|---|---|---|
| Record exists | PASS | PASS |
| Record absent | FAIL validation | FAIL validation |
| DB error | MUST NOT PASS | MUST NOT PASS |
| Checker unavailable | MUST NOT PASS | MUST NOT PASS |

---

## E. Tests

`packages/validation/presence_test.go`:

- `TestUniqueAbsentRecordPasses`
- `TestUniqueExistingRecordFails`
- `TestUniqueDatabaseErrorMustNotPass`
- `TestUniqueCheckerUnavailableMustNotPass`
- `TestExistsExistingRecordPasses`
- `TestExistsAbsentRecordFails`
- `TestExistsDatabaseErrorMustNotPass`
- `TestExistsCheckerUnavailableMustNotPass`
- `TestPresenceMalformedRuleMustNotPass`
- `TestPresenceEmptyTableOrColumnMustNotPass`
- `TestPresenceEmptyValueStillSkips`
- `TestUniqueExtraCSVPartsStillUseTableAndColumn`

`packages/validation/form_request_test.go` (FormRequest → `ValidateForm` → unique/exists → default checker):

- `TestValidateFormUniqueCheckerUnavailableMustNotPass`
- `TestValidateFormUniqueDatabaseErrorMustNotPass`
- `TestValidateFormUniqueAbsentPassesExistingFails`
- `TestValidateFormExistsExistingPassesAbsentFails`

`go test -race` was not runnable here (Windows host, no gcc / cgo). Focused `go test ./validation` passed. No new shared mutable checker was added.

---

## F. API compatibility

| Surface | Changed? |
|---|---|
| Kernel | NO |
| contracts ABI | NO |
| ORM public API | NO |
| FormRequest taxonomy | NO |
| Doctor catalog (APP-VAL-001) | NO (Why copy only) |
| `PresenceChecker` / `SetPresenceChecker` / `SetDefaultPresenceChecker` signatures | NO |

Behavior change only: missing checker and incomplete rules no longer pass.

---

## G. Remaining validation risks

- Concatenated `"uni"+"que:…"` still bypasses APP-VAL-001 (Phase 3.5 SEMANTIC). Runtime of a real `unique`/`exists` rule is fail-closed.
- `exists:` is still not authorization / IDOR protection.
- Ignore-ID / extra `WHERE` on `unique:users,email,id,5` remains unimplemented (extra CSV parts ignored). Not a fail-open hole.
- `SetDefaultPresenceChecker` is process-global (pre-existing). Tests must not mutate it under `t.Parallel()`.
- Nested object validation remains PARTIAL (unrelated).
- Query `context.Context` remains an ORM-package gap (G-M6).

---

## H. ADR status

**ADR-0010: RESOLVED** at runtime (fail-closed). Historical fail-open is retained as context in the ADR. APP-VAL-001 remains the structural doctor check; it does not prove SQL.
