# Architecture rule catalog

Machine-enforced subset of the Application Engineering STANDARD. If a rule is not listed here as **error** or **warning**, it is documentation/review only.

Analyzer: `zatrano doctor` (`console/doctor.go`, `console/doctor_arch.go`).
Framework package boundaries: `tests/architecture_test.go`, `contracts/import_test.go`.

HTMX is **not** analyzed (ADR-0005; no implementation). Repositories are **not** required (ADR-0003).

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
| APP-LAY-001 | No `domain/`, `usecases/`, `dtos/`, `handlers/`, `actions/`, `entities/` application layers | exact directory names | error | generated app | `app/usecases/` | does not match `DomainEvent` filenames |
| APP-LAY-002 | No `package usecase` / `dto` / `domain` under app | AST package name | error | `package models` | `package usecases` | `package domain` only under `app/` or `domain/` |
| APP-LAY-003 | No `UseCase`, `DTO`, `WebService`, `ApiService` types | struct type suffix | error | `OrderPlacementService` | `CreatePostUseCase` | does not ban `Handler`, `Action`, `Entity` types |
| APP-LAY-004 | Canonical dirs exist | filesystem | warning | `zatrano new` | missing `app/routes/web` | incomplete checkouts warn |
| APP-LAY-005 | No legacy `application/`, top-level `routes/`, `app/controllers` | filesystem | error (config: warning) | canonical tree | `application/` | — |
| APP-ROUTE-001 | HTTP route verbs in `app/routes/{web,api}` | AST call names | error | generated routes | `router.Get` in a service | heuristic on receiver names |
| APP-ROUTE-002 | No `app.Router().Put/Patch/Delete/Resource` | AST | error | `routing.From(app).Put` | `app.Router().Put` | — |
| APP-CON-001 | Avoid importing contract concretes | import path | warning | `routing` in routes/providers | `kernel/config` in a controller | not merge-blocking |
| APP-PROV-001 | Provider types have Register and Boot | AST | warning | generated providers | type `FooProvider` missing Boot | structs named `*Provider` without being kernel providers |
| APP-CTL-001 | `*Controller` types live under `app/http/controllers/{web,api,admin}` | AST + path | error | generated controllers | `app/services/PostController` | — |
| APP-CTL-002 | Exported HTTP methods are `(req *http.Request) *http.Response` | AST | error | generated Index | `Handle() error` on a controller when it takes `*Request` | helpers without Request are ignored |
| APP-CTL-003 | Do not mix `http.View` and `http.JSON` in one method | AST | error | web View-only | both in `Show` | **except** `*auth*` files / `AuthController` (ADR-0009) |
| APP-CTL-004 | API controllers must not call `http.View` | AST | error | api JSON home | `http.View` in `controllers/api` | JSON in `controllers/web` is allowed (api scaffold home) |
| APP-CTL-005 | Controllers must not call `orm.Transaction` / `QueryTx` | AST | error | TX in `app/services` | TX in a controller | does not prove a business op *needs* a TX |
| APP-REQ-001 | FormRequest with `Rules` is not a bare `{Resource}Request` | type name | error | `PostStoreRequest` | `PostRequest` | allowlist of action stems (Login, Store, …) |
| APP-REQ-002 | No `validation.Make` in controllers | AST | error | `ValidateForm` | inline Make | dashboard stubs still violate if copied |
| APP-REQ-003 | Store/Update that persist must `ValidateForm` when validation is enabled | AST + EnabledAddons | error | Index-only stubs | `orm.Create` in Store without ValidateForm | does not require FormRequest on Show/Destroy |
| APP-ORM-001 | No `With("relation")` string eager | AST string arg | error | `With(orm.EagerHasMany[...])` | `With("comments")` | other `With("…")` in a file that imports orm |
| APP-VAL-001 | `unique`/`exists` require `database` enabled | Rules strings + enabled.go | error | unique + database | unique without database | string scan, not full rule parser |

---

## Explicitly not machine-enforced

Authorization *correctness*, IDOR semantics, CSRF presence, Fillable completeness, “this workflow needs a transaction”, mandatory repositories, HTMX, relationship signature checking, nested validation graphs, business Policy abilities.

Those remain tests + review. See [enforcement.md](enforcement.md).
