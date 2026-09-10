# Phase 4 report — Application Engineering Standard freeze

Date: 2026-09-10

STANDARD is **frozen** ([ADR-0011](decisions/0011-application-engineering-standard-freeze.md)). Kernel, contracts, ORM public API, and ABI were not modified. No speculative doctor rules were added.

---

## A. Standard completeness

Concern inventory: [completeness.yaml](completeness.yaml) (frozen schema).

| Metric | Count |
|---|---|
| Concerns fully defined in STANDARD | 57 / 57 listed |
| With a canonical example (golden or generator) | 48 / 57 |
| Generator-supported | 30 / 57 |
| Statically enforceable (doctor or FW test) | 23 / 57 |
| CI-enforced (`go test` / doctor fixtures / architecture tests) | 28 / 57 |
| Semantic-only | 30 / 57 |
| Not provided | 12 / 57 |

“Fully defined” means STANDARD states CANONICAL / FORBIDDEN / SEMANTIC / NOT PROVIDED. It does not mean doctor proves the concern.

---

## B. No-second-way result

Document: [no-second-way.md](no-second-way.md).

| Class | Count |
|---|---|
| Intentional alternatives | 7 (optional repo, jsonapi/resources, bus, tenancy package, api-scaffold web JSON home, make:auth mix, service vs controller per §H) |
| Semantic bypasses | 6 phase-3.5 stacks + assigned router/With + unused FormRequest + dynamic unique + role-string AuthZ |
| Undocumented ambiguity | **0** |

---

## C. Doctor status

| Item | Value |
|---|---|
| Rule IDs | 24 (3 FW-DEP + 21 doctor) |
| Positive tests | generated empty/web/api/full + named PASS fixtures |
| Negative tests | every error rule (`console/doctor_arch_test.go`, `doctor_adversarial_test.go`, `doctor_test.go`) |
| False-positive tests | business names, `app/core`, concrete repo, AuthController, typed With, non-orm With |
| Phase 3.5 bypasses | 6 stacks preserved; classified SEMANTIC |

No new rules this phase.

---

## D. Generator status

| Command | Canonical? | Doctor on output | Notes |
|---|---|---|---|
| `zatrano new` (empty) | YES | PASS (`TestNewEmpty`) | Kernel HTTP/CLI |
| `zatrano new --web` | YES | PASS | HTML home |
| `zatrano new --api` | YES | PASS | JSON home under `controllers/web` (ADR-0009 scaffold exception) |
| `zatrano new --full` | YES | PASS | Both |
| `make:controller` | YES | stub PASSes | View if `view` enabled; JSON for `--api` / api scaffold |
| `make:request` | YES (validation enabled) | names match APP-REQ-001 | `--store/--update/--index` |
| `make:model` | YES (package) | — | ORM when enabled |
| `make:service` | YES | empty service PASSes | `NewX()`; comment forbids ritual layer |
| `make:repository` | YES if used | concrete PASS | Package CLI; interfaces FAIL APP-REP-001 |
| `make:policy` | YES | fluent `NewXPolicy` | Not `PostPolicy.Update` methods |
| `make:middleware` | YES | — | Group registration is SEMANTIC |
| `make:view` | YES (view enabled) | — | |
| `make:auth` | YES with exception | mix allowed on AuthController | Do not copy for resource CRUD |

Kernel CLI always: controller, middleware, provider, command, service, exception, test. Package `make:*` require enablement.

---

## E. Runtime hardening backlog

Documentation gaps are **not** listed here.

| ID | Problem | Impact | Current | Expected | Recommended fix |
|---|---|---|---|---|---|
| ADR-0010 | `unique`/`exists` pass when no PresenceChecker | Duplicate rows; fake existence | Fail-open | Fail-closed or hard error without checker | Validation-package ADR; do not fake in apps |
| G-M6 | ORM query has no `context.Context` | Cancelled requests may still hit SQL | No ctx | Package ADR | ORM package, not app workaround |
| Nested TX | `NOT SUPPORTED` | Panic / wrong rollback if nested | Documented | Keep unsupported or package ADR | Do not invent savepoints in apps |
| Jobs inside TX | No outbox | Side effects after rollback | SEMANTIC | Enqueue after nil return | App discipline; no UnitOfWork |
| Disk+DB file replace | No distributed TX | Disk orphans | Documented | Accept or compensating delete | STANDARD §V |
| G-H2 | `role ==` in controllers | Fake AuthZ | SEMANTIC | Policy only | Not a doctor rule (FP) |
| G-P2 | `authorization.ResponseFor` JSON-only | Wrong 403 body on web if misused | Documented | Web uses `http.Abort` | Do not change helper this freeze |
| EncryptCookies | Not default-global | Cookie payload not encrypted unless enabled | Documented | Opt-in | Session package |
| G-M1 | Empty `package:preset` vs scaffold enablement | CLI confusion | Presets empty | Align later | Not architecture freeze |

---

## F. Contradictions closed this phase

| Was | Now |
|---|---|
| ADR-0004 “TX only in services/console” vs doctor (controllers only) | STANDARD + ADR-0004 enforcement = controller **files**; elsewhere SEMANTIC |
| ADR-0006 “doctor role ==” vs no such rule | SEMANTIC; doctor does not scan |
| ADR-0007 “until generator implemented” | Implemented Phase 1 |
| ADR-0010 “doctor warns” | APP-VAL-001 is **error** |
| STANDARD “PROPOSED” orm.Transaction vs database TX | ACCEPTED: use orm.Transaction for models |
| STANDARD “PROPOSED” jobs after commit | ACCEPTED |
| “single environment truth” | RESOLVED at kernel boot snapshot |
| G-H2 doctor claim | Docs only |

Remaining tension (documented, not contradictory): JSON in `controllers/web` is allowed by doctor because the API scaffold home lives there; STANDARD forbids JSON for **resource HTML CRUD**.

---

## G. Recommendation

**FROZEN.**

Blockers for freeze: none that are documentation/architecture. Runtime fail-open unique/exists is classified, not a freeze blocker.

Next work (not this phase): validation-package ADR for fail-closed unique/exists; optional examples repo; do not grow doctor heuristics for the six SEMANTIC stacks.
