# Golden report — applications and architecture conformance

Date: 2026-09-10

Scope: documentation and golden specification only. Kernel, contracts, ORM runtime, public API, and ABI were **not** modified. Generators were **not** modified in this report.

---

## 1. Golden scenario matrix

| Scenario | Model | Request | Validation | Controller | Service | Repository | Relations | TX | Auth | AuthZ | View | API | Tests |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| User | `User` + Fillable | Register/Profile/UserIndex | FormRequest; unique needs DB | auth stubs + optional admin UserController | no (CRUD); hashing via auth | no | owns Post/Order/File | no | session / PAT | `user.*` Policy | profile views | `/api/v1/auth` exception | `auth_test` / `user_test` |
| Post | `Post` | Index/Store/Update | FormRequest only | web + api `PostController` | **no** | no | BelongsTo Category/User | no | auth on writes | `post.*` after Find | posts/* | JSON CRUD | `tests/http/post_test.go` |
| Category | `Category` | Index/Store/Update | FormRequest | web + api | no | no | HasMany Posts | no | optional | `category.*` | category views | JSON | `category_test.go` |
| Product | `Product` | **Index always** + Store/Update | numeric/unique/exists | web + api | no | no | optional Category | no | optional public read | `product.*` | product views | JSON | filter 422 tests |
| Order | Order + OrderItem | `OrderStoreRequest` | dotted items | web + api Store | **`OrderPlacementService.Place`** | no | User, Items, Product | **yes** in service | required | `order.create` + ownership | order views | JSON 201 | service TX + HTTP |
| File | `File` metadata | `FileUploadRequest` | mime/size | web + api | only if disk+DB must be coordinated | no | owner User | no distributed TX | required | before store and download | optional | JSON id | traversal/403 |
| Authentication | generated User | Login/Register/password/MFA | FormRequest (`make:auth`) | `web.AuthController` | auth package | no | session | n/a | **is** auth | guest/auth MW | auth views | same controller + `WantsJSON` | `auth_test.go` |
| Authorization | — | FormRequest Authorize | 403 vs 422 | after Find | n/a | no | resource FKs | n/a | user present | Gate/Policy **only** | Abort 403 | `ResponseFor` JSON | policy tests |
| Notification | — | n/a | n/a | not in HTTP usually | after-commit from service | no | Notifiable user | after TX | n/a | n/a | n/a | n/a | fake channels |
| AI | optional transcript | `CompletionRequest` | prompt rules | thin controller | optional persist | no | none required | if persist+side effects | if endpoint protected | optional | View or JSON | `ai.From(app).Chat` | fake provider |

Exact file/class/method names: [golden.md](golden.md).

---

## 2. Architecture gaps

### Critical

None newly opened by golden scenarios. Applications still compile if they invent UseCases (**G-C1**) until doctor. Directory audit: **PASS** — no undocumented folder is required.

### High

| ID | Gap |
|---|---|
| G-H3 / ADR-0010 | `unique`/`exists` fail-open without PresenceChecker — correctness; not AuthZ |
| G-H2 remaining | dashboard stubs still exist; STANDARD forbids using them as AuthZ |
| G-P2 | `authorization.ResponseFor` is JSON-only — web must `http.Abort(403)` |

### Medium

| ID | Gap |
|---|---|
| G-M1 | Empty `PresetAPI`/`PresetWeb` vs scaffold enablement (kernel — out of the golden report) |
| G-M4 | `routing.Controller` registrar Get/Post only |
| G-M6 | ORM query has no `context.Context` |
| Nested validation | dotted keys only — Order items are PARTIAL |
| Disk+DB file replace | no distributed transaction; orphans possible |
| Notification outbox | jobs inside a rolled-back TX are a known hole; enqueue after commit |

### Low

EncryptCookies not default-global; no browser E2E; `make:resource` still optional; dashboard `validation.Make` leftovers.

---

## 3. Ambiguities (closed or remaining)

| Concern | Possible interpretations | Why ambiguous | Canonical rule | Files |
|---|---|---|---|---|
| Policy type | `PostPolicy.Update` methods vs `NewPostPolicy().Define` | examples said methods; generator emits fluent Policy | **Fluent `NewXPolicy()` + `gate.Policy("post", …)` + `Authorize(user, "post.update", post)`** | STANDARD §Q, golden.md |
| Authorize argument order | `Authorize("update", user, post)` vs `(user, ability, args)` | STANDARD sketch was wrong | **`Authorize(user, "post.update", post)`** | STANDARD §Q, gate.go |
| Web/API share | one controller + WantsJSON vs two controllers vs two services | auth stubs mix JSON/View | **Resources: two controllers (ADR-0009). Auth generator: exception** | ADR-0009 |
| IndexRequest | optional if only `page`/`q` vs always | STANDARD E said “more than page/q” | **Always when any query param including page** | STANDARD E, golden.md |
| Service for CRUD | always vs never vs “large project” | size is not a criterion | **Decision table §H — Order yes, Post no** | STANDARD H |
| unique/exists | security feature vs optional vs bug | silent pass | **Documented limitation + correctness (ADR-0010)** | validator.go |
| 403 helper | always ResponseFor | JSON-only helper | **API: ResponseFor; Web: Abort(403)** | gate.go |
| Create through HasMany | association Create vs FK Create | foreign ORM habit | **`orm.Create` with FK** | STANDARD L |
| Repository | needed for “real” apps | optional generator | **None in golden; optional concrete wrapper only** | ADR-0003 |

---

## 4. Unsupported features

### Not supported by ORM

Cursor/keyset pagination as a page API; query `context.Context`; nested TX/savepoints; typed `ErrModelNotFound`; string `With("comments")`; inferred-FK-only style as the documented app pattern.

### Not supported by kernel

`contracts.App` package methods; fragment views; `contracts.Router` Put/Patch/Delete/Resource (use `routing.From`).

### Not supported by packages

`packages/mail` as a standalone addon (mail lives under notification); first-party RBAC service; image-processing; generic workflow engine; `rag`/`agent` as boot addons (libraries).

### Not required by STANDARD

UseCase, Action, DTO, Entity, DomainService, Handler, UnitOfWork, mandatory Repository, PatchRequest, SearchRequest, WebService/ApiService, jsonapi by default.

---

## 5. AI determinism score

| Golden requirement | Deterministic |
|---|---|
| Create CRUD for Product | **YES** |
| Create a Category/Post relationship | **YES** |
| Create an Order that writes Order + OrderItems atomically | **YES** |
| Create an authenticated user’s Post | **YES** |
| Create a file upload with ownership | **YES** |
| Create the API equivalent of Post CRUD | **YES** |

Every former NO (Policy shape, IndexRequest, dual controllers, unique fail-open, auth exception) is now an explicit rule in STANDARD / golden.md / ADR-0009 / ADR-0010. Remaining non-determinism is **G-C1** (compiler does not yet reject a second way) — doctor.

---

## 6. STANDARD changes (this report)

- `docs/architecture/golden.md` — exact placement, CRUD traces, taxonomies
- `docs/architecture/golden-report.md` — this report
- `docs/architecture/STANDARD.md` — §E IndexRequest, §F unique/exists, §G dual controllers + 403, §H service table, §K query extras, §P auth exception, §Q Policy API, §V files, §Y tests, §Z generator defects
- `docs/architecture/examples.md` — pointer to golden.md
- `docs/architecture/gaps.md` / `conflicts.md` / `completeness-*` / `scans.md` / `README.md` / `roadmap.md`
- ADRs **0009**, **0010** Accepted
- `AGENTS.md` — reading order + lookup + outdated generator line

---

## 7. Implementation changes

| Surface | Changed? |
|---|---|
| Kernel | **NO** |
| Contracts | **NO** |
| ORM | **NO** |
| Public API | **NO** |
| ABI | **NO** |
| Generators | **NO** |
| Documentation | **YES** |
| Tests | **NO** (no consumer app in this repo) |

If fail-closed `unique`/`exists` or JSON/HTML `ResponseFor` is desired, that is a **package** change — stop and review separately; not done here.
