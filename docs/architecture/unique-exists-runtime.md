# Fail-closed unique/exists — validation runtime correctness

Date: 2026-09-10

STANDARD remains **frozen**. Kernel, contracts, ORM public API, ABI, generators, and the doctor catalog were not redesigned. No new doctor rule. This report hardens database-backed validation runtime in `github.com/zatrano/packages`.

**STATUS: COMPLETE**

---

## Objective

Database-backed `unique` and `exists` validation must fail closed on infrastructure/checker failures.

A database/checker/infrastructure failure must never produce a successful validation result.

---

## Result

Objective is **complete**. Runtime in `packages/validation` `checkPresence` is fail-closed. Successful lookups still pass or fail according to the rule. Historical fail-open is documented in ADR-0010, not current behavior.

---

## Verification

Executed 2026-09-10 on this Windows host. Only commands that were actually run are listed.

```text
go test ./...        PASS   (framework v2 and packages)
go vet ./...         PASS   (framework v2 and packages)
go test -race ./...  NOT EXECUTABLE — ENVIRONMENTAL LIMITATION
```

| Check | Result |
|---|---|
| `go test ./validation` unique/exists + FormRequest | PASS |
| `go test ./...` (`github.com/zatrano/packages`) | PASS |
| `go test ./...` (`github.com/zatrano/framework/v2`) | PASS |
| `go vet ./...` (both modules) | PASS |
| `go test -race ./validation` | NOT EXECUTABLE — ENVIRONMENTAL LIMITATION |
| `zatrano doctor` generated empty app | PASS (0 errors, 0 warnings) |
| `zatrano doctor --strict` generated empty app | PASS |
| `zatrano doctor .` (framework tree) | PASS with 1 layout warning (`no app/`; doctor inspects consumer apps) |
| `zatrano doctor . --strict` (framework tree) | FAIL expected: that warning is treated as error |
| Golden (`go test ./tests/compatibility`) | PASS |
| Generators (`go test ./console`, `zatrano new` empty) | PASS |

`console` tests include `TestNewEmptyApplication`, `TestNewWebApplication`, `TestNewAPIApplication`, `TestNewFullApplication` with `assertDoctorPass`.

---

## Race limitation

`go test -race ./...` could not be executed because GCC/CGO is unavailable in the current environment. This is an environmental/toolchain limitation and is not classified as a repository or test failure.

Exact evidence:

```text
CGO_ENABLED=0
CC=gcc
where gcc → INFO: Could not find files for the given pattern(s).
go test -race ./validation → go: -race requires cgo; enable cgo by setting CGO_ENABLED=1
```

This is **TEST NOT EXECUTABLE**, not **TEST FAILURE**. Race coverage was not claimed as PASS. No application code was changed to work around it.

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

`SetDefaultPresenceChecker` remains the existing package-level binding. This report does not add another global.

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

Race: **NOT EXECUTABLE — ENVIRONMENTAL LIMITATION** (see Race limitation above). No new shared mutable checker was added.

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

- Concatenated `"uni"+"que:…"` still bypasses APP-VAL-001 (doctor-boundary SEMANTIC). Runtime of a real `unique`/`exists` rule is fail-closed.
- `exists:` is still not authorization / IDOR protection.
- Ignore-ID / extra `WHERE` on `unique:users,email,id,5` remains unimplemented (extra CSV parts ignored). Not a fail-open hole.
- `SetDefaultPresenceChecker` is process-global (pre-existing). Tests must not mutate it under `t.Parallel()`.
- Nested object validation remains PARTIAL (unrelated).
- Query `context.Context` remains an ORM-package gap (G-M6).

---

## H. ADR status

**ADR-0010: RESOLVED.** Runtime is fail-closed. Historical fail-open remains in the ADR as context. APP-VAL-001 remains the structural doctor check; it does not prove SQL. This report is **COMPLETE**.
