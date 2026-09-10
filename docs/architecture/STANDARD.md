# ZATRANO A–Z Application Engineering Standard

Evidence: framework v2.2.0, packages v1.7.2, `zatrano new` scaffolds, CLI generators, package tests.

Legend: `IMPLEMENTED` · `ACCEPTED` · `NOT SUPPORTED` · `INCONSISTENT` · `SEMANTIC`

**Status: FROZEN.** Kernel, contracts, ORM public API, and ABI stay frozen. This document is the authoritative specification for building ZATRANO applications. Additive architectural rules require an ADR that supersedes a previous decision ([ADR-0011](decisions/0011-application-engineering-standard-freeze.md)). Golden scenarios: [golden.md](golden.md). Machine subset: [rules.md](rules.md). Ambiguity audit: [no-second-way.md](no-second-way.md). Doctor boundary: [doctor-boundary.md](doctor-boundary.md).

There is one canonical way. “It depends” is not an answer unless this document names the exact decision table.

Enforcement classes (do not pretend AST proves semantics):

| Class | Meaning |
|---|---|
| PROVABLE | Compiler, import tests, or exact filesystem/AST |
| HIGH-CONFIDENCE STRUCTURAL | `zatrano doctor` error with bounded false positives |
| SEMANTIC | Tests, review, and this STANDARD. Doctor must not claim proof |

---

## A. Executive Architecture Summary

ZATRANO is a **two-module application platform**.

```text
Application (consumer)
    │  controllers, routes, providers, models, views
    │  Enabled ∩ Imported
    ▼
Packages (github.com/zatrano/packages)
    │  From(app) / app.Make
    ▼
Kernel (github.com/zatrano/framework/v2)
    contracts.App · container · config · router · HTTP · lifecycle · CLI
```

An application is not a DDD project and not a clone of another platform. It is a **provider-booted HTTP (and console) consumer**:

1. `bootstrap.App` selects addons (`EnabledAddons` ∩ blank-imports).
2. Providers `Register` then the kernel installs global middleware, then providers `Boot`, then the container **freezes**.
3. Lifecycle: `Created → Bootstrapping → Booted → Starting → Running → Stopping → Stopped`. Until Booted, HTTP is 503.
4. Request path: kernel middleware → route middleware → FormRequest (writes) → controller → optional service → ORM/package → View or JSON.

**Minimum canonical stack** (no extra layers):

```text
Route → Middleware → FormRequest (Authorize + Validate)
     → Controller
     → Service   (only if workflow / multi-write / reuse)
     → orm.* / pkg.From(app)
     → Response
```

Rejected as application layers: UseCase, Action, Domain Service, Entity (DDD), Value Object, DTO (separate from FormRequest), Handler (instead of Controller), mandatory Repository, UnitOfWork, WebService/ApiService.

Golden scenarios (exact files, verbs, tests): [golden.md](golden.md). Golden report: [golden-report.md](golden-report.md).

---

## B. Application Engineering Language

| Term | Meaning | Location |
|---|---|---|
| Application | Consumer module created by `zatrano new` | project root |
| Kernel | Frozen HTTP/boot ABI | `framework/v2` |
| Package / Addon | Optional service or library | `packages/<name>` |
| Enablement | Name in `EnabledAddons` **and** imported | `bootstrap/enabled.go`, `bootstrap/addons.go` |
| Provider | `Register` / `Boot` | `app/providers`, package `ServiceProvider` |
| Controller | HTTP entry | `app/http/controllers/{web,api,admin}` |
| FormRequest | Input + authorize + rules | `app/http/requests` |
| Service | Application workflow | `app/services` |
| Model | ORM struct | `app/models` |
| Repository | Optional ORM seam | `app/repositories` |
| Policy / Gate | Authorization | `app/policies` + `authorization.From` |
| Route file | HTTP map | `app/routes/web`, `app/routes/api` |
| View | Server-rendered HTML | `app/views` |
| Resource | Optional JSON transformer | `app/http/resources` — **not required** |
| Command | Console command | `app/console/commands` |
| Job | Queue payload | `app/jobs` |
| Notification | Mail/SMS/database channel | `app/notifications` |

**Canonical request names** (`make:request --store|--update|--index`):

| HTTP intent | Type name | Example |
|---|---|---|
| Create | `{Resource}StoreRequest` | `BlogStoreRequest` |
| Update (full) | `{Resource}UpdateRequest` | `BlogUpdateRequest` |
| Partial | `{Resource}UpdateRequest` | same type; no PatchRequest type |
| Destroy | none | route + policy only |
| Index filters | `{Resource}IndexRequest` | **required** when the index accepts any query (`page`, `q`, `sort`, filters) |
| Login / register | `{Action}Request` | `LoginRequest` |
| Upload | `{Resource}UploadRequest` | `FileUploadRequest` |
| Bulk | `{Resource}BulkRequest` | only if the endpoint is bulk |

Do **not** create `CreateRequest` as a global type. Do **not** create DTO packages.

`bus.Dispatch` exists (`packages/bus`). **ACCEPTED (ADR-0001):** it is not the CRUD path. Use it only when the application explicitly needs a command bus. Default is a named service verb (`Place`, `Publish`), not `Handle()`.

---

## C. Canonical Directory Structure

`IMPLEMENTED` — union of `kernel/dirs.CanonicalConsumerDirs()` (doctor-required) and generated scaffolds.

```text
.
├── app/
│   ├── console/
│   │   ├── commands/          # make:command
│   │   └── kernel.go
│   ├── database/
│   │   ├── factories/         # packages/factory
│   │   ├── migrations/
│   │   └── seeders/
│   ├── enums/                 # make:enum
│   ├── events/ listeners/ subscribers/
│   ├── exceptions/
│   ├── http/
│   │   ├── controllers/
│   │   │   ├── admin/
│   │   │   ├── api/
│   │   │   └── web/
│   │   ├── middleware/
│   │   ├── requests/          # make:request (validation package)
│   │   └── resources/         # optional make:resource
│   ├── jobs/
│   ├── localization/{en,tr}/
│   ├── models/
│   ├── notifications/
│   ├── observers/ scopes/ casts/ rules/
│   ├── policies/
│   ├── providers/
│   ├── repositories/          # optional
│   ├── routes/{api,web}/
│   ├── services/              # make:service
│   └── views/{layouts,components,partials}/
├── bootstrap/                 # addons.go, enabled.go, database_drivers.go
├── cmd/app/main.go
├── public/{css,js}/
├── storage/{app,framework,logs,backups}/
├── tests/
├── .env / .env.example
├── go.mod
└── Dockerfile / docker-compose.yml
```

**Profiles** (`IMPLEMENTED`):

| Command | Profile | Presentation | Default EnabledAddons |
|---|---|---|---|
| `zatrano new myapp` | empty | kernel HTTP/CLI | `[]` |
| `zatrano new myapp --web` | web | HTML `http.View` | assets, health, localization, view |
| `zatrano new myapp --api` | api | JSON | health, validation |
| `zatrano new myapp --full` | full | Web Apply + API Overlay | web defaults + API overlay |

`--minimal` **fails**. `APP_BOOT=minimal` is legacy runtime only.

**Forbidden directories (HIGH-CONFIDENCE STRUCTURAL — APP-LAY-001):** `domain/`, `internal/usecase/`, `internal/entity/`, `handlers/` (instead of controllers), `dtos/`, `interactors/`, `app/application/` (invented application layer), `app/http/handlers`, `app/http/actions`, `entities/`, `actions/`.

**Not a forbidden *name*, still not a ZATRANO layer (SEMANTIC):** `app/core`, `app/workflows`, `app/processors`, `app/orchestration`. Doctor does not auto-ban these names (business words / false positives). Using them as a second application architecture is **non-compliant**. Put code in §C.1 directories.

Empty / web / api share this **tree**. Difference is file **content** and enablement, not a second layout.

### C.1 Directory classification

Every path is one of: **CANONICAL** (always present / required), **CONDITIONAL** (scaffold or package-owned; empty until used), **USER-DEFINED** (files inside a canonical/conditional dir that are not a new layer), **FORBIDDEN**, **NOT PROVIDED**.

| Path | Class | Purpose | Package name | Generator | Doctor |
|---|---|---|---|---|---|
| `cmd/app` | CANONICAL | Process entry | `main` | `zatrano new` | APP-LAY-004 |
| `bootstrap/` | CANONICAL | Enablement ∩ imports | `bootstrap` | `zatrano new` | layout |
| `app/providers` | CANONICAL | `Register`/`Boot` | `providers` | `make:provider` | APP-PROV-001 |
| `app/routes/web` | CANONICAL | Web HTTP map | `web` | `zatrano new` | APP-ROUTE-001 |
| `app/routes/api` | CANONICAL | API HTTP map | `api` | `zatrano new` | APP-ROUTE-001 |
| `app/http/controllers/web` | CANONICAL | HTML/redirect HTTP entry | `web` | `make:controller` | APP-CTL-* |
| `app/http/controllers/api` | CANONICAL | JSON HTTP entry | `api` | `make:controller --api` | APP-CTL-* |
| `app/http/controllers/admin` | CONDITIONAL | Admin HTTP entry | `admin` | `make:controller --admin` | APP-CTL-* |
| `app/http/middleware` | CONDITIONAL | App middleware | `middleware` | `make:middleware` | — |
| `app/http/requests` | CONDITIONAL | FormRequest types | `requests` | `make:request` | APP-REQ-* |
| `app/http/resources` | CONDITIONAL | Optional JSON transformers | `resources` | package `make:resource` | — |
| `app/models` | CONDITIONAL | ORM structs | `models` | `make:model` | — |
| `app/services` | CONDITIONAL | Workflows only when §H | `services` | `make:service` | APP-CTL-005 (TX not in controllers) |
| `app/repositories` | CONDITIONAL | Optional concrete ORM wrappers | `repositories` | `make:repository` | APP-REP-001 (no interfaces/generic) |
| `app/policies` | CONDITIONAL | Gate policies | `policies` | `make:policy` | — |
| `app/views` | CONDITIONAL | HTML (web) | templates | `make:view` | — |
| `app/jobs` | CONDITIONAL | Queue payloads | `jobs` | `make:job` | — |
| `app/notifications` | CONDITIONAL | Notification types | `notifications` | `make:notification` | — |
| `app/events` · `listeners` · `subscribers` | CONDITIONAL | App events | matching | `make:event` / listener / subscriber | — |
| `app/console/commands` | CONDITIONAL | CLI commands | `commands` | `make:command` | — |
| `app/database/{migrations,seeders,factories}` | CONDITIONAL | Schema / seed / factory | matching | `make:migration` / seeder / factory | — |
| `app/enums` · `casts` · `observers` · `scopes` · `rules` | CONDITIONAL | Package-owned helpers | matching | `make:enum` / cast / observer / scope / rule | — |
| `app/exceptions` | CONDITIONAL | App exception types | `exceptions` | `make:exception` | — |
| `app/localization` | CONDITIONAL | Lang files | dirs | `make:lang` / web scaffold | — |
| `app/broadcasting` | CONDITIONAL | Scaffold placeholder | — | none in kernel CLI | not a layer |
| `app/views/mail` | CONDITIONAL | Notification mail HTML | templates | notification views | not a mail package |
| `tests/` | CANONICAL (scaffold) | App tests | `tests` | `make:test` | — |
| `public/` · `storage/` | CANONICAL (scaffold) | Assets / local disks | — | `zatrano new` | — |
| `domain/` · `app/domain/` · `usecases/` · `interactors/` · `dtos/` · `handlers/` · `actions/` · `entities/` · `app/application/` | FORBIDDEN | Second architecture | — | none | APP-LAY-001/002 |
| `app/core` · `app/workflows` · `app/processors` · `app/orchestration` | NOT CANONICAL | SEMANTIC-forbidden as layers | — | none | limitation (doctor boundary) |
| fragment-view dirs, `app/usecases`, Outbox, UnitOfWork | NOT PROVIDED | Do not invent | — | none | — |

**Allowed contents** of a canonical dir: types that belong to that row. **Forbidden contents:** HTTP controllers outside `controllers/{web,api,admin}`; FormRequest `Rules()` outside `app/http/requests` (APP-REQ-001 still flags the type anywhere); `orm.Transaction` in controller files.

**Naming:** files `snake_case.go`; exported types `{Resource}Controller`, `{Resource}StoreRequest`, `{Name}Service`. Package name matches the leaf dir (`web`, `api`, `services`, `models`).

---

## D. Canonical Dependency Graph

```text
app/http/controllers  →  kernel/http, kernel/routing
                      →  app/http/requests
                      →  app/services          (optional)
                      →  pkg.From(app)         (auth, view, …)
                      →  orm.*                 (allowed; see §J)

app/http/requests     →  kernel/http, packages/validation
                      →  NOT orm writes
                      →  Gate only via Authorize(req) (read auth user, no queries that mutate)

app/services          →  orm.*, pkg.From(app), app/models
                      →  NOT kernel/http.Request coupling except as an explicit argument if needed
                      →  NOT view rendering

app/models            →  packages/orm tags and optional TableName/Fillable
                      →  NOT controllers, NOT validation package

app/repositories      →  orm.*, app/models only

app/policies          →  authorization, models
                      →  NOT HTTP responses

app/providers         →  contracts.App, routing.From, csrf, ApplyWeb/ApplyAPI

app/routes            →  routing.From, controllers, middleware

packages/*            →  contracts, kernel primitives
framework             →  NEVER github.com/zatrano/packages   (ENFORCED)
```

**Forbidden:** controller → `database/sql` directly when ORM is enabled; service → `http.View`; model → `http.Request`.

**Container:** factory `Bind` / `Singleton` / `Make`. No reflection autowiring. `IMPLEMENTED`.

---

## E. Canonical Request Taxonomy

`IMPLEMENTED` type: `validation.FormRequest` (`Rules`, `Authorize`, `Messages`). Optional: `AttributesAware`, `ErrorBagAware`, `PreparesValidation`.

`ACCEPTED` (ADR-0002): mutating HTTP uses FormRequest when `validation` is enabled. `make:auth` writes `ValidateForm`. Dashboard stubs may still call `validation.Make` (remaining gap). Canonical:

| Input class | Canonical type | Mandatory? |
|---|---|---|
| POST/PUT/PATCH body (web or API) | `{Resource}{Intent}Request` FormRequest | YES if `validation` enabled |
| Login / register / password | `{Action}Request` FormRequest | YES |
| File upload | `{Resource}UploadRequest` | YES — rules include file constraints |
| Index query (`page` / `q` / `sort` / filters) | `{Resource}IndexRequest` | YES whenever the index accepts a query string. Skip only a static list with no query params |

**IndexRequest is the only filter/sort/search API.** Pagination is `Query[T]().Paginate(page, perPage)` / `SimplePaginate` (ORM). There is no cursor-page application API. Do not invent `FilterRequest`, query-object packages, or fragment search endpoints.
| Path id only (show/destroy) | none | Policy + `req.Param` |
| Headers / cookies | `req` primitives | no FormRequest |
| Nested JSON objects | **weak today** — flatten keys (`address.city`) | GAP: nested object validation |

**PROHIBITED names (HIGH-CONFIDENCE when they declare `Rules() map[string]string` — APP-REQ-001):** `PostRequest`, `CreatePostRequest`, `PostCreateRequest`, `PostForm`, `PostDTO`, `PostInput`, `PostPayload`, `PostCommand`. Canonical names are the table above. `PostIndexRequest` / `LoginRequest` remain valid.

**PROHIBITED APIs:** `validation.Make` in controllers, services, models, or repositories (APP-REQ-002). Dashboard stubs may still call `Make` — do not copy that into resource CRUD. Wrappers under `internal/` are SEMANTIC (doctor boundary). Unused canonical FormRequests while Store persists without `ValidateForm` is APP-REQ-003 when persist is in the same controller method; unused types without persist are SEMANTIC.

**Binding:** `req.All()` feeds the validator. Normalization belongs in `PrepareForValidation`. Defaults belong there or in rules (`required` vs `nullable`).

**Controller interaction:**

```go
form := requests.BlogStoreRequest{}
validated, err := validation.ValidateForm(req, form)
if err != nil {
    return validation.ResponseFor(req, err)
}
// pass validated map or bind into service input — not the raw Request for writes
```

**AI decision:** mutating HTTP → FormRequest in `app/http/requests`. Read-only with only path id → no request type. Index with `page`/`q`/`sort`/filters → IndexRequest. PUT and PATCH → the same UpdateRequest. No PatchRequest. No DeleteRequest unless the body carries bulk ids.

---

## F. Canonical Validation Model

`IMPLEMENTED` — `packages/validation`.

- API: `Make(data, rules)`, `ValidateForm`, `ValidateRequest`.
- Rules: string map `field → "required|email|min:3"`. Custom rules via `make:rule` (`app/rules`).
- Cross-field: `confirmed`, `same`, `different` (package rules). Domain invariants are **not** validation rules — they belong in the service after validation.
- Nested: dotted keys. True nested-object graphs are **PARTIAL**.
- Database-aware: `unique`, `exists` require a PresenceChecker (ADR-0010). **Without a checker, or if the checker/query cannot complete, the rule MUST NOT pass (fail-closed).** A completed lookup still passes or fails according to the rule (unique: absent passes; exists: present passes). This is a **correctness** invariant — not a security control and not authorization. Applications that use these rules MUST enable `database`. Never use `exists:` as IDOR protection.
- Localization: validation messages via FormRequest `Messages()` and lang files when localization is enabled.
- Errors: `ValidationException` → API 422 JSON `{message, errors}` or web flash + `RedirectBack`.
- Precognitive: `IsPrecognitive` uses JSON 422.

**MUST happen:** HTTP boundary (`ValidateForm`) before controller business logic.

**MUST NOT happen:** re-implement the same rules in the ORM; validate inside the model; skip `Authorize` on FormRequest.

**Authorization vs validation:** `Authorize` runs **first**. Failed authorization is 403 (`FailedAuthorization`), not 422.

**Mass assignment:** ORM `Fillable` / `Guarded` is a persistence control, not validation. Both apply: validate input, then only fillable attributes reach `Create`.

---

## G. Canonical Controller Model

`IMPLEMENTED` shape from generators and home controllers.

```go
package web

type BlogController struct{}

func (c *BlogController) Index(req *http.Request) *http.Response { ... }
```

| Rule | Canonical |
|---|---|
| Directory | `app/http/controllers/web` · `api` · `admin` (`--admin`) |
| Name | `{Resource}Controller` |
| Package name | `web` / `api` / `admin` |
| Constructor | empty struct **or** fields set at route registration (`BlogController{Posts: NewPostService()}`). No container autowire. |
| Methods | `Index Show Create Store Edit Update Destroy` as needed. Extra actions are extra methods, not a second controller. |
| Signature | `(req *http.Request) *http.Response` |
| Web vs API | **Two controller types** (ADR-0009): `controllers/web` returns View/Redirect; `controllers/api` returns JSON. Do not mix View and JSON in one **resource** method. Do not create WebService/ApiService. Share an application service only when §H requires one. |

**`make:auth` exception (exact, matches doctor APP-CTL-003):** files named `auth_controller.go` or `social_auth_controller.go`, or types named `AuthController` or `SocialAuthController`, may mix `http.View` and `http.JSON` (generated `WantsJSON()`). `author_controller.go` is **not** exempt. Do **not** copy this mix into Post/Product/Order/File CRUD.

| Concern | Web | API |
|---|---|---|
| Controller dir | `app/http/controllers/web` | `app/http/controllers/api` |
| Package | `web` | `api` |
| Response | `http.View` or `Redirect` (+ flash) | `http.JSON` |
| FormRequest | same types in `app/http/requests` | same types |
| Authorization | FormRequest `Authorize` then Gate; 403 via `http.Abort` | same Gate; 403 via `authorization.ResponseFor` (JSON-only) |
| Validation | `ValidateForm` → 422 redirect/flash | `ValidateForm` → 422 JSON |
| Views | yes | no (`http.View` in api is APP-CTL-004) |
| JSON | **not** resource CRUD. API-scaffold `controllers/web` home may return JSON (APP-CTL-004 does not forbid it). Resource HTML apps use View | yes |
| Redirect | yes | no (use JSON status) |
| CSRF | yes on mutating web routes | `csrf.Except("/api")` |
| Auth | session guard + `make:auth` | `apitoken` / `RegisterAuthAPI` exception |

**MUST NOT contain:** query builders beyond `Find`/`Query` one-liners delegated immediately; `Rules()` maps; `orm.Transaction` / `QueryTx` in the controller **file** (APP-CTL-005); template string building; Gate `Define`; password hashing loops; business validation (`validation.Make` or rule maps); business authorization via `role ==` (SEMANTIC — ADR-0006; doctor does not scan role strings).

**MAY:** call `ValidateForm`, `authorization` Allow/Deny, `auth.From`, one `orm.Find`, invoke a service, return View/JSON/Redirect.

**Fragments:** `NOT SUPPORTED`. No special controller branch.

**Errors:** `validation.ResponseFor` for 422; `http.Abort(404)` / FindOrFail mapped to 404; Gate deny → web `http.Abort(403, err.Error())`, API `authorization.ResponseFor(err)` (JSON-only helper — do not use it for HTML); unexpected errors return and let exception middleware report.

---

## H. Canonical Service / Application Model

`IMPLEMENTED` generator: `app/services/{name}_service.go` with `NewX()` and a named verb (no canonical `Handle()`).

**ACCEPTED (ADR-0001):** a service is **not** a ritual layer. Decision table:

| Question | Answer |
|---|---|
| When MUST a service exist? | More than one model write; explicit `orm.Transaction`; the same operation is used by HTTP **and** console/job; orchestration of several packages beyond one `From(app)` call |
| When MAY a service exist? | A single complex invariant after validation that is easier to test without HTTP (still one model write is usually a controller) |
| When MUST NOT a service exist? | Single-model Index/Show/Store/Update/Destroy; “the project is large”; to mimic UseCase/Interactor/Handler |

**Constructor:** `New{Name}()` returning `*{Name}`. Generator: `make:service` → `app/services/{name}_service.go`, type `{Name}Service`, `New{Name}Service()`. Method names are verbs (`Place`, `Publish`, `Register`). **No** canonical `Handle()`.

**Forbidden as services:** UseCase types, Action types, Interactor types, DomainService types, HTTP `*Handler` types, injecting `contracts.App` into every service (pass `From(app)` results or concrete deps at `New`). `CreatePostHandler.Handle` inside `app/services` is **SEMANTIC-forbidden** (doctor may PASS — doctor boundary).

Otherwise the controller calls ORM / `From(app)` directly after validation.

```go
package services

type OrderPlacementService struct{}

func NewOrderPlacementService() *OrderPlacementService { return &OrderPlacementService{} }

func (s *OrderPlacementService) Place(input OrderPlacementInput) (*models.Order, error) {
    var order *models.Order
    err := orm.Transaction(func(tx *sql.Tx) error {
        // QueryTx only inside
        ...
    })
    return order, err
}
```

**No interface** unless a test fake is required.

**Transaction ownership:** the service. See §M.

**Authorization:** already decided before the service runs (FormRequest + Policy). The service may still enforce invariants (e.g. stock) and return errors.

---

## I. Canonical Domain Model

ZATRANO domain objects are **ORM models** in `app/models`.

There is no separate Entity/VO layer. `NOT SUPPORTED` as application architecture.

Enums: `make:enum` → `app/enums`. Use for status fields. Persist as the documented column type (string/int) via `db` tags.

Invariants that are not input rules live in the service (and optionally ORM events `creating`/`updating` for persistence-side consistency). Do not hide business workflows in model hooks.

---

## J. Canonical Repository Model

`IMPLEMENTED`: `make:repository` emits a **concrete** struct wrapping `orm.All` / `Find` / `Create`. No interface. Kernel CLI does not register `make:repository` — it is a **package** generator when the ORM/database tooling is enabled.

**ACCEPTED (ADR-0003) + doctor boundary:**

| Pattern | Class |
|---|---|
| Controller/Service → `orm.Query[T]()` | CANONICAL default |
| Concrete `type PostRepository struct` wrapping ORM | OPTIONAL seam (PASS) |
| `type PostRepository interface` | FORBIDDEN (APP-REP-001) |
| `BaseRepository` / `GenericRepository` / `*RepositoryFactory` | FORBIDDEN (APP-REP-001) |
| Repository per model “because architecture” | MUST NOT |
| Validation or Gate inside a repository | MUST NOT |

Golden scenarios never require a repository. Using concrete repositories as the **default** access path for every model is SEMANTIC drift (doctor boundary bypass 6) — still optional, never mandatory.

---

## K. Canonical ORM Specification

Package: `github.com/zatrano/packages/orm`. Database service calls `orm.Configure`.

### Model definition — `IMPLEMENTED`

```go
type Post struct {
    orm.Model
    orm.SoftDeletes // optional
    Title  string `db:"title" json:"title"`
    UserID int64  `db:"user_id" json:"user_id"`
}

func (Post) TableName() string { return "posts" } // optional; default snake plural
func (Post) Fillable() []string { return []string{"title", "user_id"} }
```

| Feature | Status |
|---|---|
| struct + `db` / `json` tags | SUPPORTED |
| embed `orm.Model` (id, timestamps) | SUPPORTED |
| `SoftDeletes` / `deleted_at` | SUPPORTED |
| `TableName()` | SUPPORTED |
| `Fillable` / `Guarded` | SUPPORTED |
| `Connection()` named DB | SUPPORTED |
| composite primary keys | NOT SUPPORTED as first-class |
| optimistic locking / version column | NOT SUPPORTED |
| JSON columns | store as `[]byte` / string; no first-class JSON type API |
| arrays / PG arrays | NOT a dedicated API |
| enums | application `app/enums`, not ORM types |
| custom casts | `make:cast` + casts package |
| `Defaults() map[string]any` (`HasDefaults`) | SUPPORTED — applied on Create when missing |
| UUID / ULID keys | SUPPORTED via `UsesUUIDKeys` / `UsesULIDKeys` |
| unique / indexes on the struct | **NOT** — declare in **migrations** |

Nullable: pointer fields (`*time.Time`, `*string`). Unique constraints and indexes live in `app/database/migrations`, not on the model type.

### Lifecycle — `IMPLEMENTED`

Create / update / delete / restore (soft). Events via dispatcher: `creating`, `created`, `updating`, `updated`, `saving`, `saved`, `deleting`, `deleted`, `restoring`, `restored`. Not method hooks on the struct.

`FindOrFail` wraps `sql.ErrNoRows` as `fmt.Errorf("no query results for model [%s]", table)` — **not** a typed sentinel. Canonical handling: errors.Is `sql.ErrNoRows` on `Find`/`First`; treat FindOrFail error as 404 at the HTTP boundary.

### Query API — include only what exists

Entry: `orm.Query[T]()`, `orm.Where[T]`, `orm.Find`, `orm.FindOrFail`, `orm.All`, `orm.Create`, `orm.InsertMany`, `orm.Upsert`.

`Querier` includes: Where / OrWhere / WhereAny / WhereAll / WhereIn / WhereNotIn / WhereBetween / WhereNotBetween / WhereDate / WhereMonth / WhereYear / WhereDay / WhereTime / WhereDayOfWeek / WhereHour / WhereNull / WhereNotNull / WhereColumn / WhereRaw / WhereExists / WhereLike; OrderBy / OrderByDesc / Latest / Oldest / Reorder; Select / Distinct; GroupBy / Having; Join / LeftJoin / RightJoin / CrossJoin; Limit / Offset / Take / Skip / ForPage; Union; locks (`LockForUpdate`, `ForUpdate`, `SharedLock`, `SkipLocked`, `NoWait`); WithTrashed / OnlyTrashed; Get / First / Sole; Count / Sum / Avg / Min / Max; Value / Pluck; Paginate / SimplePaginate; With(loader funcs); Clone; **Update(attrs)**; **Delete()**.

**NOT SUPPORTED:** cursor pagination (there is a `Cursor` iterator, not keyset pages); `WhereNot` as a named method (use `Where` with operators / `WhereNotIn` / `WhereNotNull`); query `context.Context`; typed `ErrModelNotFound`.

Raw SQL: `WhereRaw`, `SelectRaw`, `OrderByRaw`, plus `database/query` builders. Prefer parameterized Where.

### Canonical usage

```go
post, err := orm.Find[models.Post](id)
posts, err := orm.Query[models.Post]().
    Where("status", "published").
    OrderByDesc("id").
    With(orm.EagerHasMany[models.Post, models.Comment]("Comments", "post_id")).
    Paginate(page, 15)
created, err := orm.Create[models.Post](map[string]any{"title": title, "user_id": uid})
```

Pagination: `Paginate` (length-aware) for web indexes; `SimplePaginate` when count is expensive.

---

## L. Canonical Relationship Specification

Relations are **functions**, not struct tags and not string names.

| Type | API | Status |
|---|---|---|
| has-many | `HasMany`, `EagerHasMany`, `LoadHasMany` | SUPPORTED |
| has-one | `HasOne`, eager/load equivalents | SUPPORTED |
| belongs-to | `BelongsTo` | SUPPORTED |
| belongs-to-many | `BelongsToMany`, `Attach`/`Detach`/`Sync`/`Toggle` | SUPPORTED |
| has-many-through | `HasManyThrough` | SUPPORTED |
| has-one-through | `HasOneThrough` | SUPPORTED |
| morph-to | `MorphTo`, `MorphToByTable` | SUPPORTED |
| morph-many / morph-to-many | `MorphToMany`, `MorphedByMany`, `AttachMorph`… | SUPPORTED |
| nested eager | `Nested`, `Then` | SUPPORTED |
| existence | `WhereHas`, `WhereDoesntHave`, morph/through variants | SUPPORTED |
| counts | `WithCount`, `CountRelated` | SUPPORTED |
| self-reference | same APIs with the same model type | SUPPORTED (no special API) |
| string eager (`With("comments")`) | — | NOT SUPPORTED |
| inferred FK without arguments | defaults exist (`id` local key) but **foreign key is explicit** in HasMany/BelongsTo calls | MUST pass FK |

**Canonical load:**

```go
orm.Query[models.Post]().With(
    orm.EagerHasMany[models.Post, models.Comment]("Comments", "post_id"),
)
```

FK is always explicit in application code. Do not invent magic `HasMany("comments")`.

**Create through a relationship:** there is no association `Create`. Set the foreign key and call `orm.Create`. BelongsToMany uses `Attach` / `Detach` / `Sync` / `Toggle` only.

Pivot extras: `Attach` `extra ...map[string]any`.

---

## M. Canonical Transaction Model

`IMPLEMENTED`: `orm.Transaction(fn func(tx *sql.Tx) error)`, `TransactionOn(connection, fn)`, `QueryTx` / `QueryOn`.

| Rule | Canonical |
|---|---|
| Owner | Application service (or a single console command). **Never the controller file** (APP-CTL-005). Cross-package `txutil` called from a controller is SEMANTIC-forbidden (doctor boundary). |
| Start | `orm.Transaction` at the start of the multi-write operation |
| Inside | `orm.QueryTx[T](tx)` / `QueryOn` only — default `Query[T]()` will **not** see the tx |
| Commit | implicit on nil error |
| Rollback | implicit on error or panic (panic rethrown) |
| Nested / savepoints | **NOT SUPPORTED** |
| Isolation | driver default; no ORM isolation API |
| Retry | **NOT SUPPORTED** in ORM |
| Locking | `LockForUpdate` etc. on the querier **inside** the transaction |
| Single insert/update | no transaction required |
| Jobs / notifications | **after** `orm.Transaction` returns nil. Dispatch inside a TX that rolls back is a known gap (no outbox). Do not invent Outbox. |

`packages/database` also has transactions. Application model writes use `orm.Transaction`, not a second database TX helper. There is no UnitOfWork type.

---

## N. Canonical Routing Model

`IMPLEMENTED`.

- Files: `app/routes/web/*.go`, `app/routes/api/*.go`.
- Bind: `app/routes/bind.go` + `RouteServiceProvider` → `ApplyWeb` / `ApplyAPI`.
- Registrar: `routing.From(app)` (`*Router`) — Get Post Put Patch Delete Resource Group Use Name.
- `contracts.Router` is **narrow** (Get/Post/Use/Group/Name). Application code MUST use `routing.From`, not `app.Router()` for REST verbs.
- `routing.Controller(r, ctrl, func(r, c) { r.Get(..., c.Index) })` groups one controller. `RouteRegistrar` only has Get/Post — **INCONSISTENT** with Put/Patch. **PROPOSED:** register Put/Patch/Delete on `*Router` / group directly; use `Controller()` only when all routes are Get/Post, or pass the concrete router.

**Web vs API:**

- Web: HTML, session, CSRF (except paths listed).
- API: `/api` prefix, JSON, CSRF excluded via `csrf.Except("/api")` in `AppServiceProvider`.

**Canonical declaration:**

```go
func RegisterWeb(r *routing.Router) {
    r.Get("/posts", posts.Index).Name("posts.index")
    r.Post("/posts", posts.Store).Name("posts.store")
    r.Get("/posts/{id}", posts.Show).Name("posts.show")
    r.Put("/posts/{id}", posts.Update)
    r.Delete("/posts/{id}", posts.Destroy)
}
```

Named routes when a name will be used in redirects/views. Resource helper is allowed when it matches this verb map.

**Fragment routes:** `NOT SUPPORTED` as a class. Same web routes.

---

## O. Canonical Middleware Model

Kernel boot order `IMPLEMENTED` (`kernel/application.go`):

1. Trusted proxy (`trustedproxy.FromEnv`)
2. Exception handler
3. RequestID
4. SecurityHeaders (HSTS on HTTPS in production)
5. HTTP-bridge package middleware
6. TrimStrings
7. ConvertEmptyStringsToNull (except password fields)
8. CORS (`CORS_ENABLED`, default true)
9. Optional by container name: octane, maintenance, metrics-timing, inspector, audit
10. Provider Boot — application CSRF `Except("/api")`
11. Route-group middleware: auth, guest, throttle, authenticate stubs from `make:auth`

**Failure:** middleware returns a Response (401/403/419/429) or next(). Do not panic.

**Context mutation:** RequestID headers; auth user on the request/session; CSRF token in session.

**Application middleware** lives in `app/http/middleware` (`make:middleware`). Register on groups in route files, not globally unless it is truly global.

**EncryptCookies:** exists in kernel/session stack but is **not** default-global. Treat as opt-in until enabled by session package boot. Gap: document per session package.

---

## P. Authentication

`IMPLEMENTED` — `packages/auth`, session driver guards. PAT: `packages/apitoken` middleware. OAuth **server**: `packages/oauth`. Social login: `packages/social`. WebAuthn: heavy package.

User model: **application stub from `make:auth`**, not a package type.

Canonical:

- Resolve `auth.From(app)`.
- Web: session guard + `make:auth` routes/controllers/middleware.
- API tokens: `apitoken` middleware, not session.
- MFA/2FA: supported in auth package. Use generated/auth APIs; do not invent a second MFA.
- Password hashing: hashing package / auth registrar — never store plaintext.
- Events: `auth.login`, `auth.logout`, `auth.two_factor_authenticated`, etc.

**NOT a second auth path:** Basic auth exists as helper; do not use it for the application user session.

Logout / session invalidation: auth manager methods on the generated controllers.

Registration / password reset / MFA / profile: generated by `make:auth`. Follow those stubs (`RegisterAuthWeb`, `RegisterAuthAPI`, `web.AuthController`). FormRequests + `ValidateForm` are canonical after generator alignment.

**ADR-0009 exception:** `RegisterAuthAPI` reuses `web.AuthController` and `WantsJSON()`. Do **not** copy that mixing into Post/Product/Order/File CRUD — those use two controllers.

---

## Q. Authorization

`IMPLEMENTED` — `packages/authorization` Gate + Policy.

Gate is bound during **auth boot**, not by authorization’s empty provider.

`make:policy` emits a constructor, not methods on a `PostPolicy` struct:

```go
func NewPostPolicy() *authorization.Policy {
    return authorization.NewPolicy().
        Define("view", ...).
        Define("create", ...). // add; generator ships view/update/delete
        Define("update", ...).
        Define("delete", ...)
}

gate := authorization.From(app)
gate.Policy("post", policies.NewPostPolicy())
err := gate.Authorize(user, "post.update", post) // (user, ability, args...)
```

Ability names are `{policyName}.{ability}` (`splitAbility` on `.`). Do not write `PostPolicy.Update` as the application type — that is not what the generator or Gate API owns.

**ACCEPTED (ADR-0006):**

- Resource actions → Policy in `app/policies` (`make:policy`).
- Who / where / when: authenticated user; FormRequest `Authorize` then `gate.Authorize` **after Find** for ownership; before mutate and before leaking private rows.
- Default HTTP: **401** guest (auth middleware), **404** missing row, **403** authenticated but forbidden.
- Ownership: policy compares `user.AuthID()` to `post.UserID` / `order.UserID`. Never trust body `user_id`.
- **Dashboard role/permission stubs are not the authorization API.** Do not check `role == "admin"` in controllers.
- Web 403: `http.Abort(403, err.Error())`. API 403: `authorization.ResponseFor(err)` (always JSON).
- FormRequest `Authorize` is coarse HTTP access. Policy is the resource-layer check. Store: `post.create`. Update/Destroy: Find then `post.update` / `post.delete`.

Route middleware may call Gate for ability names; it does not replace Policy. There is no third authorization system.

---

## R. Security

| Control | Status | Canonical application duty |
|---|---|---|
| CSRF | kernel `csrf`, Except `/api` | Web POST/PUT/PATCH/DELETE stay on CSRF; API uses tokens |
| XSS | views escape by default (Go templates) | never `template.HTML` for user input |
| SQL injection | ORM parameterized Where | WhereRaw only with bound args |
| Path traversal | filesystem package | never join user paths unsanitized |
| Upload | filesystem + validation | mime/size rules on UploadRequest |
| Sessions / cookies | session package + production cookie policy | `APP_ENV=production` secrets required |
| Trusted proxies | kernel FromEnv | set in production |
| CORS | kernel CORSFromEnv | configure origins; do not `*` in production |
| HSTS | SecurityHeaders in production HTTPS | no extra app middleware |
| Mass assignment | Fillable/Guarded | every writable model implements Fillable |
| IDOR | Policy before mutate | never trust client user_id for ownership |
| Rate limit | `packages/ratelimit` | enable and group on login and API writes |
| Secrets | `.env`, never commit | `ensureProductionSecrets` |
| Error disclosure | exception handler | production hides stacks |
| unique/exists silent pass | ADR-0010 limitation | enable `database`; never as AuthZ |

**EncryptCookies** not default — do not assume cookie payload encryption unless the session/encrypt stack is enabled.

---

## S. Views

`IMPLEMENTED`: `packages/view`, `http.View`, layouts `@yield`, components, localization in generated web welcome.

**HTML fragments: NOT SUPPORTED** as a framework or package API (zero matches in framework/packages source). Do not add fragment-swap conventions to the standard. Progressive enhancement with plain forms + Redirect + flash is the web model.

Canonical web cycle:

```text
TrustedProxy → Exceptions → RequestID → SecurityHeaders → Trim/Null
→ CORS → CSRF (web) → Auth → AuthZ → FormRequest
→ Controller → (Service) → ORM → http.View | Redirect + flash
```

Assets: `public/css`, `public/js`, `assets` package when enabled. No mandated Tailwind utility API in kernel (welcome uses inline CSS).

Forms: POST to named routes; validation errors via flash + RedirectBack (`validation.ResponseFor`).

API cycle: same without CSRF/View; JSON 422/403/401.

---

## T. Errors

| Class | Identity | HTTP |
|---|---|---|
| Validation | `validation.ValidationException` | 422 |
| Form authz | `validation.FailedAuthorization` | 403 |
| Unauthenticated | auth middleware | 401 |
| Forbidden (Gate) | authorization deny | 403 |
| Not found | `sql.ErrNoRows` / FindOrFail wrap | 404 |
| CSRF | kernel CSRF failure | 419 (kernel behavior) |
| HTTP abort | `http.Abort` / `exceptions.Abort` | given status |
| Unexpected | any error | 500 + report |

Wrap with `%w` when adding context. Do not invent `DomainError` hierarchies.

Logging: exception middleware reporters (`app.Reports()`). Do not log passwords or tokens.

---

## U. Events / Jobs / Notifications

| Mechanism | Package | Canonical use |
|---|---|---|
| Domain/ORM events | orm dispatcher + `packages` events | persistence hooks, not workflows |
| Application events | `make:event` / listener / subscriber | side effects after a successful service |
| Command bus | `packages/bus` | optional; not CRUD |
| Queue / jobs | `packages/queue`, `make:job` | slow I/O, emails, fan-out |
| Schedule | schedule provider | periodic commands |
| Notifications | `packages/notification` | user-facing mail/SMS/database |
| Mail | **no mail package** | `Channels: ["mail"]` |
| SMS / push | via notification channels if enabled | do not invent SMTP in app |

Retry / dead-letter: follow `queue` package configuration — do not reimplement workers.

Dispatch jobs **after** commit (from the service, post-transaction). Dispatching inside a TX that rolls back is a documented limitation (no outbox). **ACCEPTED:** enqueue after `orm.Transaction` returns nil. Do not invent Outbox architecture.

### AI (when `ai` is enabled)

```text
Application → ai.From(app) → Manager.Chat(ctx, ChatRequest)
```

`rag` and `agent` are import-only libraries (no addon `Register`). Do not invent workflow/orchestration types. Tests use the package fake provider. If `ai` is not enabled, the feature is out of scope — do not copy HTTP clients into `app/`. See [golden.md](golden.md) §10.

---

## V. Storage / Files

`packages/filesystem` (and `storage` layout under `storage/app`).

Canonical upload ([golden.md](golden.md) §6):

1. `{Resource}UploadRequest` / `FileUploadRequest` validates size/mime.
2. Policy `file.create` (owner from `auth`, not the body).
3. `filesystem.From(app)` — `Put` / `PutFile` on disk `local` / `public` / `s3`; kernel `safepath` (do not concatenate user paths).
4. Persist metadata on `models.File` (`path`, `disk`, `mime`, `size`, `user_id`). Generated object name, never the raw user filename as the disk path.
5. Download: Policy then `Get`; private disks are not served from `public/`.
6. Delete: Policy then disk `Delete` then ORM Delete.
7. Replace: two writes; no distributed TX — document possible disk orphans; do not invent UnitOfWork.

Image processing: **NOT SUPPORTED**. Do not invent extra storage abstractions.

---

## W. Configuration

- Kernel: `.env` / `.env.example`, `contracts.ConfigRepository`, `APP_ENV`, `APP_PORT`, secrets in production (`ensureProductionSecrets`).
- **Single environment truth (IMPLEMENTED):** bootstrap stores `app.environment = env.NormalizeAppEnv(env.Get("APP_ENV", "local"))` once. `IsProduction()` is that snapshot. CORS and cookie secure policy use the bootstrapped value, **not** a live `APP_ENV` re-parse (`kernel/env` comment; `kernel/application_env_test.go`). Mutating `APP_ENV` after boot must not change production security policy.
- `APP_DEBUG` is separate from `APP_ENV`. Production + debug is a deploy smell (`zatrano deploy` warns).
- Packages: each addon’s `.env.example` merged by `package:enable` / `install`. **Do not invent a second env parser.**
- Generated apps have **no `VERSION` file** (intentional). Framework version = `go.mod` pin.
- Application code SHOULD read `app.Config()`, not `os.Getenv` (SEMANTIC consistency). Kernel primitives may still use `env.Get` at boot.

---

## X. Observability

| Signal | Source |
|---|---|
| Logs | kernel logger `storage/logs/zatrano.log`, `LOG_LEVEL` |
| Request ID | `X-Request-ID` / `X-Correlation-ID` / traceparent |
| Metrics | optional `metrics-timing` middleware |
| Tracing | inspector middleware if enabled |
| Health | `/up` kernel; `health` package when enabled |
| Reports | `app.Reports()` exception events |
| Audit | optional audit middleware |

Application convention: log action + resource id + request id; never secrets.

---

## Y. Testing

| Kind | Location | Tool |
|---|---|---|
| HTTP feature | `tests/http/{resource}_test.go` | `packages/testing.TestCase` (Get/Post/JSON asserts) |
| Workflow / TX | `tests/services/{name}_test.go` | only when a service exists |
| Policy | `tests/policies/{resource}_policy_test.go` | `Allows` / `Denies` |
| FormRequest | same HTTP tests or request unit | ValidateForm |
| Package unit | next to package | `testing` |
| ORM | packages/orm tests; app sqlite | Configure test DB |
| Factories | `app/database/factories` + `packages/factory` | not ORM |

Scaffold still ships `tests/feature_test.go` — keep it; add `tests/http/` rather than a second top-level `test/` tree. Golden map: [golden.md](golden.md) §17.

Mocks: hand-written fakes. No mandated mock codegen.

E2E browser: **NOT SUPPORTED** as a framework package.

---

## Z. CLI / Generators / AI

### Kernel `make:*` (always)

controller, middleware, provider, command, service, exception, test.

### Package `make:*` (when enabled)

request, rule, model, migration, seeder, repository, observer, scope, cast, job, view, component, auth, dashboard, policy, notification, lang, factory, event, listener, subscriber, enum, channel.

### Missing generators (do not fake types)

UseCase, Action, DTO, Entity, ValueObject, Mail, HTML fragment, FilterRequest as a distinct command (`make:request` is enough).

### AI MUST

Follow [AGENTS.md](../../AGENTS.md). Use generators. Match this STANDARD. Run doctor and tests. Report gaps.

### AI MUST NOT

Invent layers; copy `make:controller` JSON into web apps; treat fragment views as built-in; treat dashboard RBAC stubs as Gate; nest transactions; add `zatrano.lock`.

### Generator defects to treat as non-canonical (do not copy)

1. ~~`make:controller` always JSON~~ — **addressed** (ADR-0007 / generator alignment).
2. ~~`make:service` empty `Handle() error`~~ — **addressed**.
3. `RouteRegistrar` Get/Post only vs REST verbs — register Put/Patch/Delete on `*Router`.
4. `factory` `make:resource` may be unregistered — verify before teaching AI to run it.
5. ~~Web `addons.go` blank-import gap~~ — **addressed**.
6. `make:auth` shared `AuthController` + `WantsJSON()` — **do not copy** for resource CRUD (ADR-0009).
7. Dashboard stubs may still call `validation.Make` — do not spread that.

### Versioning of this standard

Machine enforcement of the high-confidence subset: `zatrano doctor` and [rules.md](rules.md). Doctor does not replace this STANDARD; it fails CI when structural drift is detectable.

- Standard version tracks framework minor when rules change (`docs/architecture` + CHANGELOG entry).
- Additive rules: new section, not silent reinterpretation.
- Breaking architectural changes require an ADR that supersedes the previous one.
- Generated apps pin `framework/v2` in `go.mod`; they do not carry a copy of this spec until `agents:generate` links here (roadmap).

---

## Freeze, NOT PROVIDED, and semantic boundaries

This STANDARD is **frozen**. Completeness: [completeness-matrix.md](completeness-matrix.md). No-second-way audit: [no-second-way.md](no-second-way.md).

### NOT PROVIDED BY ZATRANO (do not invent)

Fragment architecture · browser E2E package · first-party `mail` package (notifications use `Channels: ["mail"]`) · Outbox · UnitOfWork · nested transactions / savepoints · query `context.Context` · typed `ErrModelNotFound` · cursor/keyset pagination as a page API · `App.AI()` / `AIService` / `AIManager` · `app.Auth()` as a kernel method · reflection autowiring · mandatory repository interfaces · UseCase/Action/Interactor/DTO/Domain Entity layers · third-party web/ORM stacks as application architecture · tenancy application directories (optional `tenancy.From(app)` only).

### Semantic doctor-PASS stacks (STANDARD still forbids as architecture)

| # | Bypass | Class | Canonical prevention |
|---|---|---|---|
| 1 | `app/core` / `workflows` / `processors` / `orchestration` as layers | SEMANTIC | Use §C.1 dirs only |
| 2 | UseCase-shaped `*Handler`/`*Action`/`Handle()` in `app/services` | SEMANTIC | Named service verb; no Handle |
| 3 | TX / View+JSON / `Make` hidden in another package | SEMANTIC | Owner stays service / FormRequest / controller method |
| 4 | JSON-only web resource CRUD; web route → API controller | SEMANTIC (+ scaffold JSON home exception) | ADR-0009 two controllers; register the matching type |
| 5 | Unused FormRequest; concatenated `unique`; `internal` Make wrapper | SEMANTIC | Call `ValidateForm`; literal rules; enable `database` |
| 6 | `*Entity` in models + concrete repos as default access | SEMANTIC (repos optional) | ORM models; `orm.Query[T]()` default |

Doctor must not grow fragile heuristics merely to drive this table to zero. Runtime `unique`/`exists` is fail-closed (fail-closed unique/exists, ADR-0010). Doctor still does not prove SQL.
