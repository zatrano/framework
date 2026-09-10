# Architecture rule catalog

Machine-enforced subset of the Application Engineering STANDARD. If a rule is not listed here as **error** or **warning**, it is documentation/review only.

Analyzer: `zatrano doctor` (`console/doctor.go`, `console/doctor_arch.go`).
Framework package boundaries: `tests/architecture_test.go`, `contracts/import_test.go`.

HTMX is **not** analyzed (ADR-0005; no implementation). Repositories are **not** required (ADR-0003). Adversarial boundary: [phase3.5.md](phase3.5.md). Freeze: [STANDARD.md](STANDARD.md) · [ADR-0011](decisions/0011-application-engineering-standard-freeze.md).

---

## CLI

```text
zatrano doctor [path] [--json] [--strict]
```

| Exit | Meaning |
|---|---|
| 0 | No **error** findings (warnings may exist) |
| 1 | One or more **error** findings, or `--strict` with warnings |

`--json` prints `{root,status,errors,warnings,findings[]}`.

---

## Framework (this repository)

| ID | Canonical | Detection | PASS | FAIL | Limitations |
|---|---|---|---|---|---|
| FW-DEP-001 | Framework MUST NOT import `github.com/zatrano/packages` | AST import walk in `tests/architecture_test.go` | this module | adding that import | none for import form |
| FW-DEP-002 | `contracts` MUST NOT import `kernel/*` | `contracts/import_test.go` | contracts tree | `import ".../kernel"` | none |
| FW-DEP-003 | `contracts` MUST NOT import packages | same | contracts tree | packages import | none |

These are **test/CI**, not `zatrano doctor` (doctor inspects consumer apps).

---

## Application (`zatrano doctor`)

| ID | Canonical | Detection | Severity | PASS | FAIL | False positives / limits |
|---|---|---|---|---|---|---|
| APP-LAY-001 | No `domain/`, `usecases/`, `interactors/`, `app/application/`, `dtos/`, `handlers/`, `actions/`, `entities/` application layers | exact directory names | error | generated app | `app/usecases/`, `app/interactors/` | does not forbid `app/core`, `app/workflows`, `app/processors`, `app/orchestration` |
| APP-LAY-002 | No `package usecase` / `interactor` / `dto` / `handlers` / `actions` / `entities` / `domain` under app | AST package name | error | `package models` | `package usecases` | `package domain` only under `app/` or `domain/` |
| APP-LAY-003 | No `UseCase`, `Interactor`, `DTO`, `WebService`, `ApiService` types | struct type suffix | error | `OrderPlacementService` | `CreatePostUseCase` | does not ban `Handler`, `Action`, `Entity` **names** without HTTP/`UseCase` shape |
| APP-LAY-004 | Canonical dirs exist | filesystem | warning | `zatrano new` | missing `app/routes/web` | incomplete checkouts warn |
| APP-LAY-005 | No legacy `application/`, top-level `routes/`, `app/controllers` | filesystem | error (config: warning) | canonical tree | `application/` | `application/` at repo root ≠ `app/application/` (both rejected, different rules) |
| APP-ROUTE-001 | HTTP route verbs in `app/routes/{web,api}` | AST call names | error | generated routes | `router.Get` in a service or `app/http/routes` | heuristic on receiver names |
| APP-ROUTE-002 | No `app.Router().Put/Patch/Delete/Resource` | AST | error | `routing.From(app).Put` | `app.Router().Put` | `r := app.Router(); r.Put` is a known bypass |
| APP-CON-001 | Avoid importing contract concretes | import path | warning | `routing` in routes/providers | `kernel/config` in a controller | not merge-blocking |
| APP-PROV-001 | Provider types have Register and Boot | AST | warning | generated providers | type `FooProvider` missing Boot | structs named `*Provider` without being kernel providers |
| APP-PROV-002 | Apps do not `Load` addon config maps | AST | warning | generated apps | `Load("view")` in consumer code | string addon names only |
| APP-CTL-001 | `*Controller` types live under `app/http/controllers/{web,api,admin}`; HTTP `(req)*Response` is not a Handler layer | AST + path | error | generated controllers | `PostHandler.Store(req) *Response` | business `Handler` types without HTTP signature pass |
| APP-CTL-002 | Exported HTTP methods are `(req *http.Request) *http.Response` | AST | error | generated Index | `Handle() error` on a controller when it takes `*Request` | helpers without Request are ignored |
| APP-CTL-003 | Do not mix `http.View` and `http.JSON` in one method | AST | error | web View-only | both in `Show` | **except** `auth_controller.go` / `AuthController` (ADR-0009). Mix in a helper is a known bypass |
| APP-CTL-004 | API controllers must not call `http.View` | AST | error | api JSON home | `http.View` in `controllers/api` | JSON in `controllers/web` is allowed (api scaffold home) |
| APP-CTL-005 | Controller **files** must not call `orm.Transaction` / `QueryTx` | AST | error | TX in `app/services` | TX in a controller helper | TX in another package called by the controller is a known bypass |
| APP-REQ-001 | FormRequest `Rules() map[string]string` uses canonical `*Request` names | type name | error | `PostStoreRequest` | `PostRequest`, `PostForm`, `CreatePostRequest` | unused canonical requests are not detected |
| APP-REQ-002 | No `validation.Make` in controllers, services, models, or repositories | AST | error | `ValidateForm` | inline Make | wrappers in `internal/` are a known bypass |
| APP-REQ-003 | Store/Update that persist must `ValidateForm` when validation is enabled | AST + EnabledAddons | error | Index-only stubs | `orm.Create` in Store without ValidateForm | persist via another package is not attributed |
| APP-REP-001 | No repository *interfaces* or generic repository bases | AST | error | concrete `PostRepository` struct | `type PostRepository interface` | does **not** require repositories |
| APP-ORM-001 | No `orm…With("relation")` string eager | AST string arg on orm call chain | error | `With(orm.EagerHasMany[...])` | `Query[T]().With("comments")` | `q.With("…")` after assignment is a known bypass; non-orm `With` is ignored |
| APP-VAL-001 | `unique`/`exists` require `database` enabled | Rules strings + enabled.go | error | unique + database | unique without database | string scan, not full rule parser; concatenated strings bypass. Runtime is fail-closed (Phase 4.5); doctor does not prove SQL |

---

## Explicitly not machine-enforced

Authorization *correctness*, IDOR semantics, CSRF presence, Fillable completeness, “this workflow needs a transaction”, mandatory repositories, HTMX, relationship signature checking, nested validation graphs, business Policy abilities.

Those remain tests + review. See [enforcement.md](enforcement.md).
