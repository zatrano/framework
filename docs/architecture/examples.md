# Canonical examples (sketches)

These are **not** demo applications. Exact file names, verbs, Policy API, and tests live in **[golden.md](golden.md)**. Golden report: **[golden-report.md](golden-report.md)**.

Copy the shape, not a Blog-only mindset. HTML fragments are omitted: **NOT SUPPORTED**.

---

## Pointers (do not invent a second pattern)

| Flow | Canonical home |
|---|---|
| Post CRUD (every verb) | golden.md §2 |
| Category ↔ Post relations | golden.md §3 + STANDARD §L |
| Product filters / unique | golden.md §4 + ADR-0010 |
| Order + items TX | golden.md §5 + STANDARD §H |
| File upload | golden.md §6 + STANDARD §V |
| Authentication | golden.md §7 — follow `make:auth` |
| Authorization / IDOR | golden.md §8 + STANDARD §Q |
| Notification | golden.md §9 |
| AI | golden.md §10 |
| Web + API | ADR-0009 — two controllers, same requests/policies |

Short sketches below remain for grepping. If they disagree with golden.md, **golden.md wins**.

---

## 1. CRUD — Blog / Post

**Enable:** `validation`, `database`, `orm`, `view` (web).

| Piece | Location |
|---|---|
| Model | `app/models/post.go` — embed `orm.Model`, `Fillable()` |
| IndexRequest | `app/http/requests/post_index_request.go` (includes `page`) |
| StoreRequest | `app/http/requests/post_store_request.go` |
| UpdateRequest | `app/http/requests/post_update_request.go` — PUT and PATCH |
| Web controller | `app/http/controllers/web/post_controller.go` |
| API controller | `app/http/controllers/api/post_controller.go` |
| Service | none (single-model writes) |
| Policy | `NewPostPolicy()`; `gate.Authorize(user, "post.update", post)` |
| ORM | `orm.Create[models.Post](validated)`, `Query[Post]().Paginate` |
| Routes | `app/routes/web/posts.go` and `app/routes/api/posts.go` |
| View | `app/views/posts/index.html`, `create.html`, `show.html`, `edit.html` |
| Test | `tests/http/post_test.go` |

Controller Store: `ValidateForm` → overwrite `user_id` from auth → `orm.Create` → `Redirect` named route.  
No repository. No UseCase. No `PostService`.

---

## 2. Relationship — Category has many Posts

| Piece | Canonical |
|---|---|
| Models | `Category`, `Post` with `category_id` |
| Relation | `HasMany` / `EagerHasMany[Category, Post]("Posts", "category_id")` |
| Nested create | `orm.Create[Post]` with FK — no association Create |
| Query | `orm.Query[Category]().With(orm.EagerHasMany[Category, Post]("Posts", "category_id"))` |
| AuthZ | Policy on Category; creating Post checks category visibility |
| Test | eager load count; FK filter |

Do not invent `With("posts")`.

---

## 3. Filtering — Product index

| Piece | Canonical |
|---|---|
| Request | `ProductIndexRequest` — `q`, `category_id`, `sort`, `page` |
| Normalize | `PrepareForValidation` (default sort `id`, page ≥ 1) |
| Validate | `nullable|string`; `exists:categories,id` **only if** database enabled (ADR-0010) |
| ORM | `Where` / `WhereLike` / `OrderBy` / `Paginate` |
| Response | View with paginator **or** JSON `{data, meta}` via a **second** API controller |
| Test | unknown sort rejected; empty q lists all |

---

## 4. Authentication — Login

| Piece | Canonical |
|---|---|
| Generator | `make:auth` — FormRequest + `ValidateForm` |
| Request | `LoginRequest` — email, password, remember |
| Auth | `auth.From(app)` session guard |
| API | same `AuthController` + `WantsJSON()` — **do not copy** for resources |
| Response | Redirect intended URL; JSON 401 for API |
| Security tests | wrong password 422/401; CSRF on web POST; no password in logs |

API clients: `apitoken` middleware, not this session login.

---

## 5. Authorization — ownership

| Piece | Canonical |
|---|---|
| Policy | `NewPostPolicy()` `update`/`delete`: `user.AuthID() == post.UserID` |
| Controller | `Find` then `gate.Authorize(user, "post.update", post)` then mutate |
| Denied | Web `http.Abort(403)`; API `authorization.ResponseFor` |
| Guest | 401 via auth middleware **before** controller |
| Test | owner 200; other user 403; guest 401 |

Do not use dashboard `roles` tables as the check.

---

## 6. Transactional workflow — Order

| Piece | Canonical |
|---|---|
| Request | `OrderStoreRequest` |
| AuthZ | `order.create` + product visibility |
| Service | `OrderPlacementService.Place` **mandatory** |
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
| Storage | `filesystem.From(app)` disk, generated path |
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
| Resolve | `ai.From(app)` — `Chat(ctx, ChatRequest)` |
| Service | optional persist transcript |
| Agent/RAG | import libraries; do not invent orchestration types |
| Response | JSON or View (two controllers if both) |
| Test | fake provider; do not hit live APIs in CI |

---

## Golden domain coverage

| Domain | Fits STANDARD without a new pattern? |
|---|---|
| Blog | YES |
| User | YES — make:auth + Policy |
| Product | YES — IndexRequest + Where + Paginate |
| Category | YES — HasMany / BelongsTo |
| Order | YES — Service + Transaction |
| File | YES — filesystem + metadata model |
| Authentication | YES — auth package |
| Authorization | YES — Gate/Policy (not RBAC package) |
| Notification | YES — notification package |
| AI | YES **if** addon enabled; otherwise out of scope |
| Fragment dashboards | **NO** — capability missing (ADR-0005) |

If a future domain needs a new layer, that is a STANDARD/ADR change, not a local invention.
