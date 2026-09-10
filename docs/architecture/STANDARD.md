# ZATRANO A–Z Application Engineering Standard

Evidence: framework v2.1.0, packages v1.7.1, `zatrano new` scaffolds, CLI generators, package tests.

Legend: `IMPLEMENTED` · `ACCEPTED` · `NOT SUPPORTED` · `INCONSISTENT`

There is one canonical way. “It depends” is not an answer unless this document names the exact decision table.

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

An application is not a DDD project and not a Laravel clone. It is a **provider-booted HTTP (and console) consumer**:

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

Golden scenarios (exact files, verbs, tests): [golden.md](golden.md). Phase 2 report: [phase2.md](phase2.md).

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

**Forbidden directories:** `domain/`, `internal/usecase/`, `internal/entity/`, `handlers/` (instead of controllers), `dtos/`.

Empty / web / api share this **tree**. Difference is file **content** and enablement, not a second layout.

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
| Path id only (show/destroy) | none | Policy + `req.Param` |
| Headers / cookies | `req` primitives | no FormRequest |
| Nested JSON objects | **weak today** — flatten keys (`address.city`) | GAP: nested object validation |

**PROHIBITED types (do not create):** `DTO`, `PatchRequest` as a parallel type, `DeleteRequest` unless the body carries bulk ids (`BulkRequest`).

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
- Database-aware: `unique`, `exists` require a PresenceChecker (ADR-0010). **Without a checker they pass silently (fail-open).** If a checker returns an error, the rule fails. This is a **documented limitation** and a **correctness** problem — not a security control and not authorization. Applications that use these rules MUST enable `database`. Never use `exists:` as IDOR protection.
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
| Web vs API | **Two controller types** (ADR-0009): `controllers/web` returns View/Redirect; `controllers/api` returns JSON. Do not mix View and JSON in one **resource** method. Do not create WebService/ApiService. Share an application service only when §H requires one. `make:auth` may reuse `AuthController` + `WantsJSON()` — that exception is not the resource CRUD pattern. |

**MUST NOT contain:** query builders beyond `Find`/`Query` one-liners delegated immediately; `Rules()` maps; `orm.Transaction`; template string building; Gate `Define`; password hashing loops.

**MAY:** call `ValidateForm`, `authorization` Allow/Deny, `auth.From`, one `orm.Find`, invoke a service, return View/JSON/Redirect.

**HTMX:** `NOT SUPPORTED`. No special controller branch.

**Errors:** `validation.ResponseFor` for 422; `http.Abort(404)` / FindOrFail mapped to 404; Gate deny → web `http.Abort(403, err.Error())`, API `authorization.ResponseFor(err)` (JSON-only helper — do not use it for HTML); unexpected errors return and let exception middleware report.

---

## H. Canonical Service / Application Model

`IMPLEMENTED` generator: `app/services/{name}_service.go` with `NewX()` and a named verb (no canonical `Handle()`).

**ACCEPTED (ADR-0001):** a service is **mandatory** when any of these is true, and **forbidden as a ritual layer** otherwise:

| Condition | Service |
|---|---|
| More than one model write | **Mandatory** |
| Explicit `orm.Transaction` | **Mandatory** (controller MUST NOT start TX) |
| Same operation used by HTTP and console/job | **Mandatory** |
| Orchestration of several packages beyond one `From(app)` call | **Mandatory** |
| Single-model Index/Show/Store/Update/Destroy | **No** — controller → ORM |
| Project is “large” | **No** — size is not a criterion |

Golden: Post CRUD has no service. Order placement is `OrderPlacementService.Place`.

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

**No interface** unless a test fake is required. **No** `Handle()` without arguments in real code — rename to the verb (`Place`, `Publish`, `Register`).

**Forbidden:** UseCase types, Action types, DomainService types, injecting `contracts.App` into every service (pass `From(app)` results or concrete deps at `New`).

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

`IMPLEMENTED`: `make:repository` emits a **concrete** struct wrapping `orm.All` / `Find` / `Create`. No interface.

**ACCEPTED (ADR-0003):**

- Default: **no repository**. Controllers/services call `orm.Query[T]()`. Golden scenarios never require a repository.
- Add a repository only as a test seam or to hide a stable query that is used in many places.
- Do not create `BlogRepositoryInterface`.
- Do not put authorization or validation in repositories.

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
| Owner | Application service (or a single console command). **Never the controller.** |
| Start | `orm.Transaction` at the start of the multi-write operation |
| Inside | `orm.QueryTx[T](tx)` / `QueryOn` only — default `Query[T]()` will **not** see the tx |
| Commit | implicit on nil error |
| Rollback | implicit on error or panic (panic rethrown) |
| Nested / savepoints | **NOT SUPPORTED** |
| Isolation | driver default; no ORM isolation API |
| Retry | **NOT SUPPORTED** in ORM |
| Locking | `LockForUpdate` etc. on the querier **inside** the transaction |
| Single insert/update | no transaction required |

`packages/database` also has transactions. **PROPOSED:** application code uses `orm.Transaction`, not a second database TX helper, when working with models.

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

**HTMX routes:** `NOT SUPPORTED` as a class. Same web routes.

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

Registration / password reset / MFA / profile: generated by `make:auth`. Follow those stubs (`RegisterAuthWeb`, `RegisterAuthAPI`, `web.AuthController`). FormRequests + `ValidateForm` are canonical after Phase 1.

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

## S. Views / HTMX

`IMPLEMENTED`: `packages/view`, `http.View`, layouts `@yield`, components, localization in generated web welcome.

**HTMX: NOT SUPPORTED** as a framework or package API (zero matches in framework/packages source). Do not add `hx-` conventions to the standard. Progressive enhancement with plain forms + Redirect + flash is the web model.

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

Dispatch jobs **after** commit (from the service, post-transaction). Dispatching inside a TX that rolls back is a gap (no outbox). **PROPOSED:** enqueue after `orm.Transaction` returns nil.

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

- Kernel: `.env` / `.env.example`, `contracts.ConfigRepository`, `APP_ENV`, `APP_PORT`, secrets in production.
- Packages: each addon’s `.env.example` merged by `package:enable` / `install` / `preset`. **Do not invent a second env parser.**
- Generated apps have **no `VERSION` file** (intentional). Framework version = `go.mod` pin.
- Source of truth: environment → config repository. Code reads `app.Config().Get` / `env` helpers already used by kernel — application code should use `app.Config()`, not `os.Getenv`, **PROPOSED** for consistency.

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

UseCase, Action, DTO, Entity, ValueObject, Mail, HTMX fragment, FilterRequest as a distinct command (`make:request` is enough).

### AI MUST

Follow [AGENTS.md](../../AGENTS.md). Use generators. Match this STANDARD. Run doctor and tests. Report gaps.

### AI MUST NOT

Invent layers; copy `make:controller` JSON into web apps; treat HTMX as built-in; treat dashboard RBAC stubs as Gate; nest transactions; add `zatrano.lock`.

### Generator defects to treat as non-canonical (do not copy)

1. ~~`make:controller` always JSON~~ — **addressed** (ADR-0007 / Phase 1).
2. ~~`make:service` empty `Handle() error`~~ — **addressed** (Phase 1).
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
