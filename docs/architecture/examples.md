# Canonical examples (sketches)

These are **not** demo applications. They show the complete canonical flow. Copy the shape, not a Blog-only mindset.

HTMX fragments are omitted: **NOT SUPPORTED**.

---

## 1. CRUD — Blog / Post

**Enable:** `validation`, `database`, `orm`, `view` (web).

| Piece | Location |
|---|---|
| Model | `app/models/post.go` — embed `orm.Model`, `Fillable()` |
| StoreRequest | `app/http/requests/post_store_request.go` |
| UpdateRequest | `app/http/requests/post_update_request.go` |
| Controller | `app/http/controllers/web/post_controller.go` |
| Service | none (single-model writes) |
| ORM | `orm.Create[models.Post](validated)`, `Query[Post]().Paginate` |
| Routes | `app/routes/web/posts.go` |
| View | `app/views/posts/index.html`, `create.html`, `show.html` |
| Test | `tests/http/post_test.go` — TestCase Get/Post, 422, 200 |

Controller Store: `ValidateForm` → `orm.Create` → `Redirect` named route.  
No repository. No UseCase.

---

## 2. Relationship — Category has many Products

| Piece | Canonical |
|---|---|
| Models | `Category`, `Product` with `category_id` |
| Relation | `HasMany` / `EagerHasMany[Category, Product]("Products", "category_id")` |
| Nested create | Service only if both rows must commit together (`orm.Transaction` + `QueryTx`) |
| Query | `orm.Query[Category]().With(orm.EagerHasMany[Category, Product]("Products", "category_id"))` |
| AuthZ | Policy on Category; creating Product checks `category` ownership |
| Test | eager load count; FK constraint failure |

Do not invent `With("products")`.

---

## 3. Filtering — Product index

| Piece | Canonical |
|---|---|
| Request | `ProductIndexRequest` FormRequest — `q`, `category_id`, `sort`, `page` |
| Normalize | `PrepareForValidation` (default sort `id`, page ≥ 1) |
| Validate | `nullable|string`, `exists:categories,id` only if database enabled |
| ORM | `Where` / `WhereLike` / `OrderBy` / `Paginate` |
| Response | View with paginator **or** JSON `{data, meta}` |
| Test | unknown sort rejected; empty q lists all |

---

## 4. Authentication — Login

| Piece | Canonical |
|---|---|
| Generator | `make:auth` then **replace inline Make with LoginRequest** |
| Request | `LoginRequest` — email, password, remember |
| Auth | `auth.From(app)` session guard |
| Session | package session |
| Response | Redirect intended URL; JSON 401 for API |
| Security tests | wrong password 422/401; CSRF on web POST; no password in logs |

API clients: `apitoken` middleware, not this session login.

---

## 5. Authorization — ownership

| Piece | Canonical |
|---|---|
| Policy | `PostPolicy` Update/Delete: `user.AuthID() == post.UserID` |
| Controller | `Find` then `gate.Authorize` then mutate |
| Denied | 403 (`FailedAuthorization` or Gate deny) |
| Guest | 401 via auth middleware **before** controller |
| Test | owner 200; other user 403; guest 401 |

Do not use dashboard `roles` tables as the check.

---

## 6. Transactional workflow — Order

| Piece | Canonical |
|---|---|
| Request | `OrderStoreRequest` |
| AuthZ | `create` on Order + product visibility |
| Service | `OrderPlacementService.Place` |
| TX | `orm.Transaction`; `QueryTx` for order + items + stock |
| Rollback | return error from fn |
| After commit | `notification` or `queue` job |
| Test | stock failure rolls back; no order row |

Controller does not call `orm.Transaction`.

---

## 7. File upload

| Piece | Canonical |
|---|---|
| Request | `FileUploadRequest` — file required, mime, max |
| Storage | `filesystem` disk, generated path, never raw user filename |
| Model | metadata row (`path`, `disk`, `user_id`) |
| AuthZ | policy before store and before download |
| Response | Redirect / JSON id+url (public only if intended) |
| Security tests | path traversal rejected; unauthorized download 403 |

---

## 8. Async — notify after publish

| Piece | Canonical |
|---|---|
| Service | `PostPublisher.Publish` updates model then `notification` or `queue.Push` |
| Job | `make:job` for slow mail |
| Failure | queue retry config; do not retry in the HTTP request |
| Test | HTTP 302 and job payload asserted (fake queue if package supports) |

Mail: `Channels: ["mail"]`. No `packages/mail`.

---

## 9. AI workflow (only if `ai` enabled)

| Piece | Canonical |
|---|---|
| Request | `CompletionRequest` (prompt rules, max tokens) |
| Resolve | `ai.From(app)` — provider/profile from package config |
| Service | `CompletionService` — call provider, persist transcript if needed |
| Agent/RAG | `agent` / `rag` packages when those addons are enabled — do not invent orchestration types |
| Response | JSON or View |
| Test | fake provider; do not hit live APIs in CI |

If `ai` is not enabled, the feature is out of scope — do not copy HTTP clients into `app/`.

---

## Golden domain coverage

| Domain | Fits STANDARD without a new pattern? |
|---|---|
| Blog | YES — CRUD example |
| User | YES — make:auth + Policy |
| Product | YES — IndexRequest + Where + Paginate |
| Category | YES — HasMany / BelongsTo |
| Order | YES — Service + Transaction |
| File | YES — filesystem + metadata model |
| Authentication | YES — auth package |
| Authorization | YES — Gate/Policy (not RBAC package) |
| Notification | YES — notification package |
| AI | YES **if** addon enabled; otherwise out of scope |
| HTMX dashboards | **NO** — capability missing (ADR-0005), not an app exception |

If a future domain needs a new layer, that is a STANDARD/ADR change, not a local invention.
