# Doctor boundary — adversarial architecture verification

Date: 2026-09-10

STANDARD was not redesigned. Kernel, contracts, ORM, public API, and ABI were not modified.

This report asks whether a competent developer or AI can ship a **second application architecture** while `zatrano doctor` still reports PASS.

```text
STANDARD → doctor → real application → adversarial mutation → PASS / FAIL → known boundary
```

---

## A. Attack matrix

| Rule | Attack | Expected | Actual | Result |
|---|---|---|---|---|
| APP-LAY-001 | `app/usecases/` | FAIL | FAIL | closed |
| APP-LAY-001 | `app/interactors/` | FAIL | FAIL | closed |
| APP-LAY-001 | `app/application/` (invented application layer) | FAIL | FAIL | closed |
| APP-LAY-001 | `app/handlers/`, `app/http/handlers/` | FAIL | FAIL | closed |
| APP-LAY-001 | `app/core/`, `app/workflows/`, `app/processors/`, `app/orchestration/` | not auto-forbidden | PASS | **acceptable limitation** (false-positive risk) |
| APP-LAY-002 | `package usecases` / `interactors` / `handlers` | FAIL | FAIL | closed |
| APP-LAY-003 | `CreatePostUseCase` / `*Interactor` / `*DTO` | FAIL | FAIL | closed |
| APP-LAY-003 | `type Handler` / `Action` / `Entity` as business names | PASS | PASS | correct |
| APP-LAY-003 | `CreatePostHandler` in `app/services` with `Handle()` (no HTTP) | FAIL if second layer | PASS | **acceptable limitation** |
| APP-CTL-001 | HTTP `(req) *Response` on `*Handler` / `*Action` | FAIL | FAIL | closed |
| APP-CTL-003 | View+JSON in one controller method | FAIL | FAIL | closed |
| APP-CTL-003 | mix in `author_controller.go` (substring `auth`) | FAIL | FAIL | closed (was false exception) |
| APP-CTL-003 | documented `AuthController` / `auth_controller.go` | PASS | PASS | correct |
| APP-CTL-003 | mix moved to `renderShow()` helper in same file | FAIL if detectable | PASS | **acceptable limitation** |
| APP-CTL-004 | `http.View` in `controllers/api` | FAIL | FAIL | closed |
| APP-CTL-004 | JSON-only method in `controllers/web` | PASS (scaffold) | PASS | documented (not dual-controller proof) |
| APP-CTL-005 | `orm.Transaction` on `*Controller` method | FAIL | FAIL | closed |
| APP-CTL-005 | `orm.Transaction` in controller-file helper | FAIL | FAIL | closed |
| APP-CTL-005 | `orm.Transaction` in `app/txutil` called by controller | FAIL if ownership | PASS | **acceptable limitation** |
| APP-CTL-005 | TX via interface / func var | semantic | PASS | **acceptable limitation** |
| APP-REQ-001 | `PostRequest` | FAIL | FAIL | closed |
| APP-REQ-001 | `CreatePostRequest` / `PostCreateRequest` | FAIL | FAIL | closed |
| APP-REQ-001 | `PostForm` / `PostDTO` / `PostInput` with `Rules()` | FAIL | FAIL | closed |
| APP-REQ-001 | unused `PostStoreRequest` while controller skips it | FAIL if detectable | PASS | **acceptable limitation** |
| APP-REQ-002 | `validation.Make` in controller | FAIL | FAIL | closed |
| APP-REQ-002 | `validation.Make` in service/model/repository | FAIL | FAIL | closed |
| APP-REQ-002 | `Make` wrapped in `internal/validate` | FAIL if detectable | PASS | **acceptable limitation** |
| APP-REQ-003 | Store + `orm.Create` without `ValidateForm` | FAIL | FAIL | closed |
| APP-REP-001 | `type PostRepository interface` | FAIL | FAIL | closed |
| APP-REP-001 | `BaseRepository` / `GenericRepository` / `*RepositoryFactory` | FAIL | FAIL | closed |
| APP-REP-001 | concrete `type PostRepository struct` + ORM | PASS | PASS | correct (ADR-0003) |
| APP-ORM-001 | `orm.Query[T]().With("comments")` | FAIL | FAIL | closed |
| APP-ORM-001 | `log.With("comments")` / `info("comments")` in an orm-importing file | PASS | PASS | closed false-positive |
| APP-ORM-001 | `q := orm.Query[T](); q.With("comments")` | FAIL if detectable | PASS | **acceptable limitation** |
| APP-ROUTE-001 | `router.Get` in `app/http/routes` or `app/services` | FAIL | FAIL | closed (call-site, not dir name alone) |
| APP-ROUTE-002 | `app.Router().Put` immediate | FAIL | FAIL | closed |
| APP-ROUTE-002 | `r := app.Router(); r.Put` | FAIL if detectable | PASS | **acceptable limitation** |
| APP-VAL-001 | literal `unique:` / `exists:` without database | FAIL | FAIL | closed (structural only) |
| APP-VAL-001 | `"uni"+"que:users,email"` | FAIL if detectable | PASS | **acceptable limitation** |
| FW-DEP-001 | direct / alias / dot import of `github.com/zatrano/packages` | FAIL | FAIL (path string) | closed; test files included |
| FW-DEP-002/003 | contracts → kernel / packages | FAIL | FAIL | production `.go` only (`*_test.go` skipped) |
| FW-DEP-* | go.mod require without import | n/a | not scanned | **acceptable limitation** |
| (none) | web route invoking API controller type | FAIL if graph | PASS | **acceptable limitation** |
| (none) | CSRF / Policy correctness / Fillable / IDOR | not AST | not claimed | documentation-only |
| (none) | fragment helpers | unsupported | not scanned | ADR-0005 |

---

## B. Bypasses (after analyzer fixes)

Successful bypasses that still doctor-PASS. Classification:

### Acceptable limitation (static analysis boundary)

1. **Unbanned extra directories** — `app/core`, `app/workflows`, `app/processors`, `app/orchestration` can host a second layer if types avoid `UseCase`/`Interactor`/`DTO` suffixes.
2. **UseCase-shaped types without banned names** — `CreatePostHandler.Handle` / `CreatePostAction.Execute` / bare `CreatePost` in `app/services`.
3. **Cross-package transaction** — controller calls `txutil.Run` which calls `orm.Transaction`.
4. **Indirect View/JSON mix** — mix lives in a helper or other package; controller only `return render(...)`.
5. **Assigned `app.Router()`** — `r := app.Router(); r.Put`.
6. **Assigned ORM query** — `q := orm.Query[T](); q.With("relation")`.
7. **Interface / function-variable indirection** for TX, View, `Make`, `With`.
8. **Unused FormRequest** — canonical type exists; mutating method never calls `ValidateForm` *and* does not persist in the same function (or persists only through a service).
9. **Dynamic `unique`/`exists` strings** — concatenation, fmt, helpers; APP-VAL-001 is a literal scan.
10. **`validation.Make` in an unscanned helper package** (`internal/validate`).
11. **Web↔API controller wiring** — no route→controller type graph.
12. **JSON-only web controllers** — allowed because the API scaffold home lives under `controllers/web`.
13. **Framework test-only contracts imports** — `contracts/import_test.go` skips `*_test.go`.
14. **Indirect module dependency** without a `.go` import.

### Analyzer bug (closed this report)

- ORM `With("…")` flagged any `With` in a file that imported orm → now requires `ormCall`.
- Controller TX only on `*Controller` methods → now any `orm.Transaction`/`QueryTx` in a controller file.
- Auth mix exception matched substring `auth` (`author_controller.go`) → now `auth_controller.go` / `AuthController` / social variants only.
- `PostForm`/`PostDTO` with `Rules()` ignored because the name did not end with `Request`.
- `app/interactors`, `app/application`, HTTP `*Handler` entries, repository *interfaces* / `BaseRepository` were not rejected.

### Architecture gap

None that require kernel/ORM/API changes. ADR-0010 runtime fail-open is unchanged (correctness, not this report).

### STANDARD ambiguity

- **One canonical CRUD form** is still a decision table: Controller→ORM *or* Controller→Service→ORM *or* optional concrete repository. Doctor correctly PASSes all three. That is not a second architecture.
- JSON in `controllers/web` is an accepted scaffold exception, not proof of ADR-0009 dual controllers.

---

## C. False positives (must PASS)

| Construct | Result |
|---|---|
| `DomainEvent`, `LegalEntity`, `ActionLog`, `NotifyHandler` | PASS |
| `app/core` + `Workflow` / `Processor` types | PASS |
| `type Domain struct` in `app/services` | PASS |
| concrete `PostRepository` wrapping ORM | PASS |
| Controller → ORM (simple Product CRUD) | PASS |
| Controller → Service → ORM | PASS |
| Controller → concrete Repository → ORM | PASS |
| `AuthController` View+JSON mix | PASS |
| typed `With(orm.EagerHasMany[...])` | PASS |
| `log.With("comments")` / `info("comments")` next to orm import | PASS |
| generated empty/web/api/full | PASS |

No rule was widened in a way that forbids repositories or requires services.

---

## D. Rule quality

Catalog after analyzer fixes: **3** framework tests + **21** doctor IDs (**17** error + **4** warning), including **APP-REP-001** and documented **APP-PROV-002**.

| Class | Count |
|---|---|
| Total catalog IDs | 24 |
| Strong (PASS + FAIL + bounded FP) | 18 |
| Weak (heuristic / scaffold exception) | 4 (ROUTE-001 receiver names, ROUTE-002 immediate call, CTL-004 web JSON allowed, REQ-003 same-function persist) |
| Needs revision | 0 after analyzer closures |
| Documentation-only | AuthZ/IDOR/CSRF/Fillable completeness, TX *necessity*, fragment views, unused FormRequest pairing |

Every error rule has a negative test. APP-REP-001, APP-LAY-001 interactors, APP-CTL-005 file-level TX, APP-REQ-001 Form/Create names, APP-ORM-001 non-orm `With` have explicit false-positive or limitation tests.

---

## E. Recommended changes

### Analyzer-only (done this report)

- Forbidden dirs/packages: interactors, `app/application`, `app/http/handlers|actions`.
- `*Interactor`; HTTP methods on non-`*Controller` types (APP-CTL-001).
- APP-REP-001: exported `*Repository` **interfaces**, `BaseRepository`, `GenericRepository`, `*RepositoryFactory`. Concrete structs remain legal.
- APP-CTL-005: controller **file** (helpers included).
- APP-CTL-003 auth exception: exact filenames/types.
- APP-REQ-001: any `Rules() map[string]string` whose type is not a canonical `*Request`.
- APP-REQ-002: `validation.Make` in services/models/repositories.
- APP-ORM-001: `ormCall` chain, not “file imports orm”.

### Documentation-only (done this report)

- This report, rules catalog, enforcement limits, completeness matrix, gaps G-C1 remainder.

### Runtime/API required (not done)

- Fail-closed `unique`/`exists` (ADR-0010).
- Nested transactions, query `context`, typed not-found (ORM ADRs).
- Route→controller graph, whole-program TX ownership, CSRF completeness.

---

## F. AI Product CRUD (STANDARD vs doctor)

| Implementation | STANDARD | Doctor |
|---|---|---|
| Controller → ORM | PASS (simple CRUD) | PASS |
| Controller → Service → ORM | PASS (when §H) | PASS |
| Controller → concrete Repository → ORM | PASS (optional) | PASS |
| Controller → HTTP `*Handler` → ORM | FAIL | FAIL APP-CTL-001 |
| Controller → `*UseCase` → ORM | FAIL | FAIL APP-LAY-003 |
| Controller → `app/application` / `app/interactors` | FAIL | FAIL APP-LAY-001 |
| Controller → `*Handler` in services (`Handle()`, no HTTP) | FAIL (non-canonical) | **PASS — limitation** |
| Controller → domain dir → repository interface | FAIL | FAIL LAY-001 + REP-001 |

Canonical PASS is the first two rows (and the third when the author opts into a thin wrapper). There is still only one *language*; optional repository is not a second required layer.

---

## G. Golden mutation

Replacing Order `OrderPlacementService` with `PlaceOrderUseCase` fails **APP-LAY-003**. Other golden mutations (bare `PostRequest`, controller TX, string `With`, API `View`, unique without database) remain FAIL as in the doctor report.

---

## H. Scope

| Surface | Changed |
|---|---|
| Kernel / contracts / ORM / ABI | NO |
| `zatrano doctor` analyzer | YES |
| Fixtures / tests | YES |
| Architecture docs | YES |
| Runtime fail-open unique/exists | NO |

---

## I. Enforcement boundary (the number)

If a highly competent developer or AI **intentionally** implements a second application architecture without changing kernel/ORM/public API, **`zatrano doctor` still reports PASS in 6 realistic ways**:

1. Invented-layer **directories doctor does not ban** (`app/core`, `app/workflows`, `app/processors`, `app/orchestration`) filled with verb types that are not `*UseCase`/`*Interactor`/`*DTO`.
2. **UseCase/Handler/Action behavior** inside `app/services` (or similar canonical dirs) without HTTP signatures and without banned suffixes.
3. **Ownership hiding** — transactions, View/JSON mixing, or `validation.Make` moved to another package or helper the checker does not attribute to the controller.
4. **Transport collapse** — JSON-only web controllers and/or web routes pointing at API controller types (no call graph).
5. **FormRequest theater** — unused canonical requests, concatenated `unique`/`exists`, or Make wrappers outside services/models/controllers.
6. **DDD-lite inside legal folders** — `*Entity` types in `app/models` plus optional concrete repositories used as the default access path (interfaces/generic bases now FAIL).

The goal is not zero. These six are **semantic / whole-program / naming** limits. Obvious second architectures (UseCase dirs, HTTP Handlers, repository interfaces, controller-file TX, `PostForm`, string eager on `orm.Query().With`) now FAIL.
