# Doctor report — machine-enforced application architecture

Date: 2026-09-10

STANDARD was not redesigned. Kernel, contracts, ORM, public API, and ABI were not modified.

Canonical tool: **extend `zatrano doctor`** (already the consumer architecture CLI). No second command.

---

## A. Rules

| Class | Count |
|---|---|
| Total catalog IDs | 22 (3 framework tests + 19 doctor) |
| Compiler-enforced | import/type as usual (not counted as STANDARD rules) |
| AST-enforced (doctor) | 16 |
| Filesystem-enforced | 3 (LAY-001 dirs, LAY-004, LAY-005) |
| Package/test-enforced (framework) | 3 (FW-DEP-*) |
| Generator-enforced | scaffolds must doctor-PASS (tests) |
| Documentation-only | AuthZ semantics, TX necessity, Fillable completeness, fragment views, repositories-as-optional, CSRF completeness |

---

## B. Rule matrix

| Rule | Enforcement | PASS test | FAIL test | CI |
|---|---|---|---|---|
| FW-DEP-001 | TEST | architecture_test | adding packages import | go test |
| FW-DEP-002/003 | TEST | contracts/import_test.go | kernel/packages import | go test |
| APP-LAY-001..003 | doctor error | TestDoctorFalsePositiveBusinessNames | TestDoctorForbiddenUseCaseDirectory | go test |
| APP-LAY-004 | doctor warning | generated apps | incomplete tree | go test |
| APP-LAY-005 | doctor error | generated apps | TestDoctorLegacyLayout | go test |
| APP-ROUTE-001 | doctor error | TestDoctorAllowsRoutePrimitives | TestDoctorMisplacedRoute | go test |
| APP-ROUTE-002 | doctor error | routing.From | (fixture in doctor_arch) | go test |
| APP-CON-001 | doctor warning | starters | TestDoctorConcreteLeak | go test (`--strict` optional) |
| APP-CTL-003..005 | doctor error | Auth mix exception; service TX | mix / TX / API View tests | go test |
| APP-REQ-001..003 | doctor error | PostStoreRequest | PostRequest; Make; Create without ValidateForm | go test |
| APP-ORM-001 | doctor error | typed EagerHasMany | With("comments") | go test |
| APP-VAL-001 | doctor error | unique + database | unique without database | go test |
| Generated empty/web/api/full | doctor PASS | TestNew* + TestDoctorCleanStarter | — | go test + starter-smoke |

---

## C. False positives / limits

- `Handler`, `Action`, `Entity`, `DomainEvent` **types** are allowed (business words).
- `internal/support` is allowed; `internal/usecase` is not.
- JSON in `controllers/web` is allowed (api scaffold home). Mixing View+JSON in one method is not (except AuthController).
- `With("…")` in a file that also imports orm may flag non-ORM helpers.
- Route detection still uses receiver-name heuristics (`r`, `router`, …).
- Doctor does not prove authorization or that a workflow needs a transaction.
- Missing canonical dirs are **warnings**, so a partial tree can still surface the real error.

---

## D. Remaining gaps

**Critical:** G-C1 is **reduced**, not gone. Doctor fails merge-blocking errors, but Go still compiles a UseCase if the author never runs doctor/CI.

**High:** unique/exists runtime fail-open unchanged (ADR-0010; APP-VAL-001 only when rules are present). Dashboard `validation.Make` still exists in stubs — doctor flags copies into controllers.

**Medium:** G-M1 presets; G-M4 RouteRegistrar; G-M6 query context; G-M9 tenancy undocumented; APP-CON-001 still warning.

**Low:** EncryptCookies; no E2E; doctor `--fix` still absent.

---

## E. Scope

| Surface | Changed |
|---|---|
| Kernel | NO |
| Contracts | NO |
| ORM | NO |
| Public API / ABI | NO |
| CLI | YES (`zatrano doctor` exit codes, `--json`, `--strict`, architecture checks) |
| Analyzer | YES (doctor) |
| Generator | NO (tests only) |
| Tests | YES |
| CI | YES (starter-smoke doctor; unit tests already cover fixtures) |
| Documentation | YES |

---

## F. Architecture proof

```text
zatrano new (empty|web|api|full)
        → zatrano doctor
        → PASS (errors: 0)

app/usecases + CreatePostUseCase
        → zatrano doctor
        → FAIL APP-LAY-001 / APP-LAY-002 / APP-LAY-003 (exit 1)
```
