# Conflict report

Two patterns exist in the ecosystem. STANDARD picks **one**. Copying the loser is a violation even if it compiles.

Choice criteria: architecture, public API stability, safety, simplicity, compiler enforceability, tooling, consistency, maintainability — not popularity, not Laravel, not generic Go.

---

## C1 — Controller JSON vs View

| Variant A | Variant B |
|---|---|
| Web `HomeController` returns `http.View` | `make:controller` always returns `http.JSON` |

**Winner:** A for web, B for `--api` / `controllers/api`.  
**Loser:** JSON-in-web from the generator.  
**Why:** Scaffold presentation is the product contract (`--web` HTML, `--api` JSON, `--full` both).  
**ADR:** 0007.

---

## C2 — FormRequest vs `validation.Make` in controllers

| A | B |
|---|---|
| `ValidateForm` + FormRequest | Inline `Make(req.All(), rules)` in controller |

**Winner:** A for mutating endpoints when `validation` is enabled.  
**Loser:** B except one-off throwaway scripts (not application code).  
**Why:** Authorize + Prepare + Messages + 422/redirect mapping already live on FormRequest.  
**ADR:** 0002.

---

## C3 — Direct ORM vs Repository vs mandatory interfaces

| A | B | C |
|---|---|---|
| `orm.Query[T]()` in service/controller | `make:repository` concrete wrapper | `Repository` interface per model |

**Winner:** A as default; B optional seam.  
**Loser:** C.  
**Why:** Generator emits concrete structs; no consumer needs interfaces.  
**ADR:** 0003.

---

## C4 — Service vs Bus vs UseCase

| A | B | C |
|---|---|---|
| `app/services` verb methods | `bus.Dispatch` | UseCase/Action types |

**Winner:** A.  
**Loser:** C always. B only when the bus package is enabled **and** the feature is a command pipeline.  
**Why:** Bus is an optional package; UseCase is not generated.  
**ADR:** 0001.

---

## C5 — `contracts.Router` vs `routing.From`

| A | B |
|---|---|
| `app.Router()` | `routing.From(app)` |

**Winner:** B for application routes.  
**Why:** Put/Patch/Delete/Resource exist only on the typed router.

---

## C6 — Gate/Policy vs dashboard roles

| A | B |
|---|---|
| `authorization` Gate/Policy | Role strings from dashboard stubs |

**Winner:** A.  
**Loser:** B as authorization. Stubs may render UI.  
**ADR:** 0006.

---

## C7 — Session auth vs API tokens vs OAuth vs Basic

| Web users | API clients | Third-party apps | Rare |
|---|---|---|---|
| `auth` session guard | `apitoken` | `oauth` server | Basic helper |

**Winner:** pick by client type; do not mix session cookies into token APIs as a second undocumented guard.  
**Loser:** inventing JWT in the application.

---

## C8 — `http.JSON` vs jsonapi vs Resource classes

| A | B | C |
|---|---|---|
| `http.JSON(map or struct)` | `packages/jsonapi` | `app/http/resources` |

**Winner:** A as default. B/C opt-in when the API must be JSON:API or a transformer is reused.  
**ADR:** 0008.

---

## C9 — CSRF on API

| A | B |
|---|---|
| `csrf.Except("/api")` in generated AppServiceProvider | CSRF on all routes |

**Winner:** A. API authenticates with tokens. Web keeps CSRF.

---

## C10 — Transaction in controller vs service vs ORM callback

**Winner:** service + `orm.Transaction` + `QueryTx`.  
**Loser:** controller TX; nested TX; `database` TX helpers for model writes.  
**ADR:** 0004.

---

## C11 — Eager load strings vs loader funcs

**Winner:** `With(orm.EagerHasMany[...])`.  
**Loser:** `With("comments")` (not an API).

---

## C12 — Generated `AGENTS.md` vs this constitution

| A | B |
|---|---|
| `agents:generate` describe dump | Framework `AGENTS.md` + `docs/architecture` |

**Winner:** B for architecture. A remains a live routing/catalog snapshot. They must link, not compete.  
**Roadmap:** prepend constitution pointer in generated file.

---

## C13 — Laravel-shaped folders vs Laravel behavior

Familiar names (`controllers`, `requests`, `providers`) are **intentional ergonomics**. Eloquent, Artisan, Blade, HTMX Livewire, Facades-on-App, `app.Auth()` are **not** ZATRANO. Do not copy Laravel internals because a folder looks similar. Do not rename folders just to look unlike Laravel.
